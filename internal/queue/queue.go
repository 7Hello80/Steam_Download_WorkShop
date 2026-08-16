package queue

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"steam-download-tool/internal/config"
	"steam-download-tool/internal/crypto"
	"steam-download-tool/internal/depot"
	"steam-download-tool/internal/model"
	"steam-download-tool/internal/storage"
	"steam-download-tool/internal/ws"
)

// Queue manages the FIFO download queue with worker pool.
type Queue struct {
	mu         sync.Mutex
	tasks      []*model.DownloadTask
	active     map[string]context.CancelFunc // taskID -> cancel
	maxWorkers int
	sem        chan struct{} // semaphore channel
	db         *sql.DB
	wsHub      *ws.Hub
	cfg        *config.Config
	ptyInputs  map[string]chan string // taskID -> channel for PTY input
	stopCh     chan struct{}
	enqueueCh  chan struct{} // signals new task enqueued
}

// NewQueue creates a new download queue.
func NewQueue(db *sql.DB, wsHub *ws.Hub, cfg *config.Config) *Queue {
	q := &Queue{
		maxWorkers: cfg.MaxWorkers,
		sem:        make(chan struct{}, cfg.MaxWorkers),
		active:     make(map[string]context.CancelFunc),
		db:         db,
		wsHub:      wsHub,
		cfg:        cfg,
		ptyInputs:  make(map[string]chan string),
		stopCh:     make(chan struct{}),
		enqueueCh:  make(chan struct{}, 1),
	}

	// Register WebSocket PTY input handler
	wsHub.OnPtyInput(func(userID, taskID, input string) {
		q.mu.Lock()
		ch, ok := q.ptyInputs[taskID]
		q.mu.Unlock()
		if ok {
			select {
			case ch <- input:
			default:
			}
		}
	})

	// Recover orphaned tasks on startup
	q.recoverOrphans()

	// Start main queue loop
	go q.loop()

	return q
}

// recoverOrphans marks any downloading/queued tasks as failed after restart.
func (q *Queue) recoverOrphans() {
	_, err := q.db.Exec(
		"UPDATE download_tasks SET status = ?, error_message = ? WHERE status IN (?, ?)",
		model.StatusFailed, "Server restarted - download interrupted",
		model.StatusDownloading, model.StatusQueued,
	)
	if err != nil {
		log.Printf("Queue: failed to recover orphans: %v", err)
	} else {
		log.Printf("Queue: recovered orphaned tasks")
	}
}

// Enqueue adds a download task to the queue.
func (q *Queue) Enqueue(task *model.DownloadTask) (int, error) {
	q.mu.Lock()

	// Assign position
	position := len(q.tasks) + len(q.active) + 1
	task.Status = model.StatusQueued
	task.QueuePosition = position
	q.tasks = append(q.tasks, task)
	q.mu.Unlock()

	// Persist
	_, err := q.db.Exec(
		"UPDATE download_tasks SET status = ?, queue_position = ? WHERE id = ?",
		task.Status, task.QueuePosition, task.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("persist queue status: %w", err)
	}

		// Notify user with full queue info
		q.wsHub.SendToUser(task.UserID, ws.MsgQueueUpdate, q.GetQueueInfo(task.UserID))

	// Signal the loop that a new task is available (non-blocking)
	select {
	case q.enqueueCh <- struct{}{}:
	default:
	}

	return position, nil
}

// GetQueueInfo returns current queue information for a user.
func (q *Queue) GetQueueInfo(userID string) map[string]interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.getQueueInfoLocked(userID)
}

// getQueueInfoLocked returns queue info without locking (caller must hold q.mu).
func (q *Queue) getQueueInfoLocked(userID string) map[string]interface{} {
	activeCount := len(q.active)
	var userPosition int
	var userTaskID string

	for i, t := range q.tasks {
		if t.UserID == userID && t.Status == model.StatusQueued {
			userPosition = i + 1
			userTaskID = t.ID
			break
		}
	}

	totalAhead := activeCount + userPosition - 1
	if totalAhead < 0 {
		totalAhead = 0
	}

	return map[string]interface{}{
		"active_count":  activeCount,
		"queue_length":  len(q.tasks),
		"your_position": userPosition,
		"your_task_id":  userTaskID,
		"total_ahead":   totalAhead,
	}
}

// broadcastQueueInfoLocked sends full queue info to a user (caller must hold q.mu).
func (q *Queue) broadcastQueueInfoLocked(userID string) {
	q.wsHub.SendToUser(userID, ws.MsgQueueUpdate, q.getQueueInfoLocked(userID))
}

// Cancel removes a task from the queue or stops an active download.
func (q *Queue) Cancel(taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check active downloads
	if cancel, ok := q.active[taskID]; ok {
		cancel()
		delete(q.active, taskID)
		q.db.Exec("UPDATE download_tasks SET status = ? WHERE id = ?", model.StatusCancelled, taskID)
		q.recalculatePositions()
		return nil
	}

	// Check pending queue
	for i, t := range q.tasks {
		if t.ID == taskID {
			q.tasks = append(q.tasks[:i], q.tasks[i+1:]...)
			q.db.Exec("UPDATE download_tasks SET status = ? WHERE id = ?", model.StatusCancelled, taskID)
			q.recalculatePositions()
			return nil
		}
	}

	return fmt.Errorf("task %s not found in queue", taskID)
}

// recalculatePositions updates queue positions and broadcasts changes.
func (q *Queue) recalculatePositions() {
	notified := make(map[string]bool)
	for i, t := range q.tasks {
		t.QueuePosition = i + 1 + len(q.active)
		q.db.Exec("UPDATE download_tasks SET queue_position = ? WHERE id = ?", t.QueuePosition, t.ID)
		// Send full queue info per user (deduped)
		if !notified[t.UserID] {
			notified[t.UserID] = true
			q.broadcastQueueInfoLocked(t.UserID)
		}
	}
}

// loop is the main queue processing loop, triggered by new tasks.
func (q *Queue) loop() {
	// Initial processing
	q.tryProcess()

	for {
		select {
		case <-q.stopCh:
			return
		case <-q.enqueueCh:
			q.tryProcess()
		case <-time.After(5 * time.Second):
			// Periodic check to process any tasks that may have been missed
			q.tryProcess()
		}
	}
}

// tryProcess tries to start as many tasks as worker slots allow.
func (q *Queue) tryProcess() {
	for {
		if !q.processNext() {
			break // no more tasks or no worker slots
		}
	}
}

// processNext tries to start the next task if a worker slot is available.
// Returns true if a task was started, false otherwise.
func (q *Queue) processNext() bool {
	q.mu.Lock()

	if len(q.tasks) == 0 {
		q.mu.Unlock()
		return false
	}

	// Try to acquire semaphore (non-blocking)
	select {
	case q.sem <- struct{}{}:
		// Got a slot
	default:
		q.mu.Unlock()
		return false // no available workers
	}

	// Pop first task
	task := q.tasks[0]
	q.tasks = q.tasks[1:]

	ctx, cancel := context.WithCancel(context.Background())
	q.active[task.ID] = cancel

	// Create PTY input channel
	inputCh := make(chan string, 1)
	q.ptyInputs[task.ID] = inputCh

	// Update positions for remaining queued tasks
	q.recalculatePositions()

	q.mu.Unlock()

	// Process task in background
	go q.processTask(ctx, task, inputCh)

	return true
}

// parseProgress extracts downloaded bytes and total bytes from DepotDownloader output.
// DepotDownloader outputs lines like: "Downloaded 12.34 MB / 56.78 MB" or "Progress: 45.2%"
func parseProgress(line string) (downloaded, total int64, percent float64) {
	lower := strings.ToLower(line)

	// Pattern: "Downloaded X / Y" or "X / Y bytes"
	// Try to find patterns like "12.34 MB / 56.78 MB"
	re := strings.Index(lower, "/")
	if re < 0 {
		return 0, 0, 0
	}

	// Look for numeric values around the "/"
	before := strings.TrimSpace(lower[max(0, re-20):re])
	after := strings.TrimSpace(lower[re+1 : min(len(lower), re+21)])

	// Try to parse as "value unit" format
	downloaded = parseSize(before)
	total = parseSize(after)

	if total > 0 {
		percent = float64(downloaded) / float64(total) * 100
	}

	return
}

// parseSize parses a size string like "12.34 MB" or "567 KB" into bytes.
func parseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Extract the last numeric value and unit from the string
	// Split by whitespace, get last two tokens
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return 0
	}

	// Try last token as unit
	var numPart, unitPart string
	if len(parts) >= 2 {
		numPart = parts[len(parts)-2]
		unitPart = strings.ToUpper(parts[len(parts)-1])
	} else {
		numPart = parts[0]
	}

	val, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0
	}

	switch unitPart {
	case "B", "BYTES":
		return int64(val)
	case "KB":
		return int64(val * 1024)
	case "MB":
		return int64(val * 1024 * 1024)
	case "GB":
		return int64(val * 1024 * 1024 * 1024)
	default:
		// No unit recognized, treat as raw bytes
		if err == nil {
			return int64(val)
		}
		return 0
	}
}

// processTask handles the full lifecycle of a download task.
func (q *Queue) processTask(ctx context.Context, task *model.DownloadTask, inputCh chan string) {
	defer func() {
		<-q.sem // Release semaphore

		q.mu.Lock()
		delete(q.active, task.ID)
		delete(q.ptyInputs, task.ID)
		q.mu.Unlock()

		q.recalculatePositions()
	}()

	// failOnce ensures failTask is called at most once for this task.
	var failOnce sync.Once
	fail := func(errMsg string) {
		failOnce.Do(func() {
			q.failTask(task, errMsg)
		})
	}

	// --- Stuck detection: if no progress for ~10 minutes, auto-cancel ---
	const stuckTimeout = 10 * time.Minute
	var progressMu sync.Mutex
	lastProgressTime := time.Now()
	var lastDownloadedBytes int64

	recordProgress := func(downloaded int64) {
		progressMu.Lock()
		if downloaded > lastDownloadedBytes {
			lastDownloadedBytes = downloaded
			lastProgressTime = time.Now()
		}
		progressMu.Unlock()
	}

	// Watchdog goroutine: check every 30s if download is stuck
	stuckCtx, stuckCancelFunc := context.WithCancel(ctx)
	defer stuckCancelFunc()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				progressMu.Lock()
				stuck := time.Since(lastProgressTime) > stuckTimeout
				progressMu.Unlock()
				if stuck {
					log.Printf("Queue: task %s stuck for >%v, auto-cancelling", task.ID, stuckTimeout)
					fail(fmt.Sprintf("下载超时：任务在 %.0f 分钟内无任何进度变化，已自动取消", stuckTimeout.Minutes()))
					stuckCancelFunc()
					return
				}
			case <-stuckCtx.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Update status to downloading
	now := time.Now()
	task.Status = model.StatusDownloading
	task.StartedAt = &now
	q.db.Exec(
		"UPDATE download_tasks SET status = ?, started_at = ? WHERE id = ?",
		task.Status, task.StartedAt, task.ID,
	)

	q.wsHub.SendToUser(task.UserID, ws.MsgTaskUpdate, map[string]interface{}{
		"task_id": task.ID,
		"status":  task.Status,
	})

	// Get output directory
	outputDir := task.OutputDir
	if outputDir == "" {
		outputDir = fmt.Sprintf("%s/%s", q.cfg.OutputDir, task.ID)
	}
	if err := storage.EnsureDir(outputDir); err != nil {
		fail(fmt.Sprintf("create output dir: %v", err))
		return
	}

	// Run DepotDownloader via PTY
	runner := depot.NewRunner(task.ID)
	defer runner.Stop()

	// Retrieve Steam password from DB
	var encryptedPwd string
	if err := q.db.QueryRow(
		"SELECT encrypted_password FROM steam_accounts WHERE user_id = ? AND steam_username = ?",
		task.UserID, task.SteamUsername,
	).Scan(&encryptedPwd); err != nil {
		fail(fmt.Sprintf("retrieve Steam credentials: %v", err))
		return
	}

	decryptedPwd, err := crypto.Decrypt(encryptedPwd)
	if err != nil {
		fail(fmt.Sprintf("decrypt Steam password: %v", err))
		return
	}

	// Generate unique login ID
	loginID := task.LoginID
	if loginID == 0 {
		loginID = rand.Intn(2147483647) + 1
	}

	if err := runner.Start(stuckCtx, q.cfg.DepotDownloaderPath, task.AppID, task.PubfileID, task.SteamUsername, decryptedPwd, outputDir, loginID); err != nil {
		fail(fmt.Sprintf("start download: %v", err))
		return
	}

	// doneCh is closed when runner completes or is stopped
	doneCh := make(chan struct{})

	// Collect output lines for error reporting
	var outputLines []string
	var outputMu sync.Mutex
	appendOutput := func(line string) {
		outputMu.Lock()
		if len(outputLines) < 200 {
			outputLines = append(outputLines, line)
		}
		outputMu.Unlock()
	}

	// Track when output reader goroutine finishes
	var outputWg sync.WaitGroup

	// Start 2FA prompt handler FIRST (before output consumer) to avoid race
	promptReady := make(chan struct{})
	go func() {
		close(promptReady)
		for {
			select {
			case prompt, ok := <-runner.PromptCh():
				if !ok {
					return
				}
				appendOutput("[PROMPT] " + prompt)
				q.wsHub.SendToUser(task.UserID, ws.MsgPtyPrompt, map[string]interface{}{
					"task_id": task.ID,
					"prompt":  prompt,
				})

				// Wait for user input with timeout
				select {
				case input := <-inputCh:
					if err := runner.Write(input); err != nil {
						log.Printf("Queue: failed to write PTY input for task %s: %v", task.ID, err)
						q.wsHub.SendPtyInputAck(task.UserID, task.ID, "error")
					} else {
						q.wsHub.SendPtyInputAck(task.UserID, task.ID, "sent")
					}
				case <-time.After(5 * time.Minute):
					fail("2FA input timeout (5 minutes)")
					runner.Stop()
					return
				case <-ctx.Done():
					return
				case <-doneCh:
					return
				}
			case <-ctx.Done():
				return
			case <-doneCh:
				return
			}
		}
	}()

	// Open terminal log file for output persistence
	logPath := outputDir + "/terminal.log"
	logFile, logErr := os.Create(logPath)
	if logErr != nil {
		log.Printf("Queue: failed to create terminal log %s: %v", logPath, logErr)
	}

	// Route PTY output to WebSocket with batching
	// Wait for prompt handler to be ready before consuming output
	<-promptReady
	outputWg.Add(1)
	go func() {
		defer outputWg.Done()
		if logFile != nil {
			defer logFile.Close()
		}
		const batchInterval = 80 * time.Millisecond
		const maxBatchSize = 40

		batch := make([]string, 0, maxBatchSize)
		ticker := time.NewTicker(batchInterval)
		defer ticker.Stop()

		flush := func() {
			if len(batch) == 0 {
				return
			}
			lines := make([]string, len(batch))
			copy(lines, batch)
			batch = batch[:0]

			q.wsHub.SendToUser(task.UserID, ws.MsgPtyOutputBatch, map[string]interface{}{
				"task_id": task.ID,
				"lines":   lines,
			})
		}

		var lastProgressSent time.Time
		for {
			select {
			case line, ok := <-runner.Output():
				if !ok {
					flush()
					return
				}
				batch = append(batch, line)
				appendOutput(line)

				// Check for workshop item not-found / error patterns early
				if isNotFound, errMsg := depot.IsNotFoundError(line); isNotFound {
					log.Printf("Queue: task %s early validation failed: %s", task.ID, errMsg)
					fail(fmt.Sprintf("下载验证失败：%s", errMsg))
					stuckCancelFunc()
					// still drain remaining lines so the goroutine can exit
				}

				// Write to terminal log file
				if logFile != nil {
					logFile.WriteString(line + "\n")
				}

				// Parse real download progress from output
				if downloaded, total, pct := parseProgress(line); total > 0 && pct > 0 {
					recordProgress(downloaded) // reset stuck timer on progress
					now := time.Now()
					if now.Sub(lastProgressSent) > 2*time.Second {
						lastProgressSent = now
						q.wsHub.SendToUser(task.UserID, ws.MsgTaskUpdate, map[string]interface{}{
							"task_id":          task.ID,
							"status":           task.Status,
							"downloaded_bytes": downloaded,
							"total_bytes":      total,
							"progress_pct":     pct,
						})
					}
				}

				if len(batch) >= maxBatchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()

	// Wait for download completion
	waitErr := runner.Wait()
	close(doneCh)

	// Wait for all PTY output to be drained before checking output directory
	outputWg.Wait()

	if waitErr != nil {
		// Include last output lines in error message for debugging
		outputMu.Lock()
		lastOutput := strings.Join(outputLines, "\n")
		outputMu.Unlock()
		errMsg := fmt.Sprintf("DepotDownloader exited with error: %v", waitErr)
		if len(lastOutput) > 0 {
			// Truncate to fit in DB field
			if len(lastOutput) > 800 {
				lastOutput = "..." + lastOutput[len(lastOutput)-800:]
			}
			errMsg = errMsg + "\nOutput:\n" + lastOutput
		}
		fail(errMsg)
		return
	}

	// Check if task was already failed (e.g. 2FA timeout)
	if task.Status == model.StatusFailed {
		return
	}

	// Verify output directory has actual files (not just .DepotDownloader cache)
	entries, readErr := os.ReadDir(outputDir)
	if readErr != nil {
		fail(fmt.Sprintf("failed to read output directory: %v", readErr))
		return
	}
	hasFiles := false
	for _, entry := range entries {
		if entry.Name() != ".DepotDownloader" {
			hasFiles = true
			break
		}
	}
	if !hasFiles {
		outputMu.Lock()
		lastOutput := strings.Join(outputLines, "\n")
		outputMu.Unlock()
		errMsg := "download produced no files — the Steam account may not own this game, or workshop content is not accessible"
		if len(lastOutput) > 0 {
			if len(lastOutput) > 800 {
				lastOutput = "..." + lastOutput[len(lastOutput)-800:]
			}
			errMsg = errMsg + "\nOutput:\n" + lastOutput
		}
		fail(errMsg)
		return
	}

	// Zip and move to static
	zipPath, zipFilename, fileSize, err := depot.ZipAndMove(task.ID, outputDir, q.cfg.StaticDir)
	if err != nil {
		fail(fmt.Sprintf("zip/move: %v", err))
		return
	}

	// Update task as completed
	completeTime := time.Now()
	expiresAt := completeTime.Add(time.Duration(q.cfg.FileTTLHours) * time.Hour)
	task.Status = model.StatusCompleted
	task.CompletedAt = &completeTime
	task.ExpiresAt = &expiresAt
	task.ZipPath = zipPath
	task.ZipFilename = zipFilename
	task.FileSize = fileSize

	q.db.Exec(
		`UPDATE download_tasks SET status = ?, completed_at = ?, expires_at = ?, zip_path = ?, zip_filename = ?, file_size = ? WHERE id = ?`,
		task.Status, task.CompletedAt, task.ExpiresAt, task.ZipPath, task.ZipFilename, task.FileSize, task.ID,
	)

	q.wsHub.SendToUser(task.UserID, ws.MsgTaskUpdate, map[string]interface{}{
		"task_id":      task.ID,
		"status":       task.Status,
		"zip_filename": task.ZipFilename,
		"file_size":    task.FileSize,
		"expires_at":   task.ExpiresAt,
	})
}

// failTask marks a task as failed and notifies the user.
func (q *Queue) failTask(task *model.DownloadTask, errMsg string) {
	task.Status = model.StatusFailed
	task.ErrorMessage = errMsg

	q.db.Exec(
		"UPDATE download_tasks SET status = ?, error_message = ? WHERE id = ?",
		task.Status, task.ErrorMessage, task.ID,
	)

	q.wsHub.SendToUser(task.UserID, ws.MsgTaskUpdate, map[string]interface{}{
		"task_id":       task.ID,
		"status":        task.Status,
		"error_message": task.ErrorMessage,
	})
}

// Stop gracefully shuts down the queue.
func (q *Queue) Stop() {
	close(q.stopCh)

	q.mu.Lock()
	defer q.mu.Unlock()

	// Cancel all active downloads
	for taskID, cancel := range q.active {
		cancel()
		q.db.Exec("UPDATE download_tasks SET status = ?, error_message = ? WHERE id = ?",
			model.StatusFailed, "Server shutting down", taskID)
	}
}
