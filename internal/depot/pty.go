package depot

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/creack/pty"
)

// PTYRunner manages a DepotDownloader process with pseudo-terminal for 2FA input.
type PTYRunner struct {
	ptmx      *os.File
	cmd       *exec.Cmd
	output    chan string
	promptCh  chan string // signals when 2FA input is needed
	done      chan error
	mu        sync.Mutex
	stopped   bool
	taskID    string
}

// NewRunner creates a new PTYRunner for a download task.
func NewRunner(taskID string) *PTYRunner {
	return &PTYRunner{
		taskID:   taskID,
		output:   make(chan string, 1024),
		promptCh: make(chan string, 8),
		done:     make(chan error, 1),
	}
}

// Start launches DepotDownloader in a PTY session.
func (r *PTYRunner) Start(ctx context.Context, depotBinPath string, appID, pubfileID int64, username, password, outputDir string, loginID int) error {
	args := []string{
		"-app", strconv.FormatInt(appID, 10),
		"-pubfile", strconv.FormatInt(pubfileID, 10),
		"-username", username,
		"-password", password,
		"-dir", outputDir,
		"-loginid", strconv.Itoa(loginID),
	}

	r.cmd = exec.CommandContext(ctx, depotBinPath, args...)
	r.cmd.Env = os.Environ()

	// Create PTY with standard terminal size
	var err error
	r.ptmx, err = pty.StartWithSize(r.cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		return fmt.Errorf("start PTY: %w", err)
	}

	// Start output reader (close output/promptCh when done)
	go r.readOutput()

	// Start wait goroutine — only signals completion, does NOT close channels.
	// Channel lifecycle is owned by readOutput() to avoid "send on closed channel" panics.
	go func() {
		r.done <- r.cmd.Wait()
	}()

	return nil
}

// readOutput continuously reads PTY output and routes it.
// Owns the lifecycle of r.output and r.promptCh channels.
func (r *PTYRunner) readOutput() {
	defer close(r.output)
	defer close(r.promptCh)

	scanner := bufio.NewScanner(r.ptmx)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB buffer for large output

	for scanner.Scan() {
		line := scanner.Text()

		select {
		case r.output <- line:
		default:
			// output buffer full, drop line to avoid blocking PTY reads
		}

		// Check for 2FA/prompt patterns
		if isPromptLine(line) {
			select {
			case r.promptCh <- line:
			default:
			}
		}
	}

	// PTY I/O errors are expected when the process exits and the PTY is closed.
	// Only report unexpected scanner errors (not "input/output error" from /dev/ptmx cleanup).
	if err := scanner.Err(); err != nil && !isPtyCloseError(err) {
		select {
		case r.output <- fmt.Sprintf("[PTY read error: %v]", err):
		default:
		}
	}
}

// isPtyCloseError returns true if the scanner error is an expected PTY I/O error
// that occurs when the child process exits and the PTY master is closed.
func isPtyCloseError(err error) bool {
	if err == nil {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "input/output error") ||
		strings.Contains(msg, "file already closed")
}

// isPromptLine detects Steam Guard / 2FA prompt patterns.
func isPromptLine(line string) bool {
	lower := strings.ToLower(line)
	patterns := []string{
		"steam guard",
		"2fa",
		"two-factor",
		"two factor",
		"authenticator code",
		"enter your code",
		"enter the code",
		"steam guard code",
		"two factor code",
		"mobile authenticator code",
		"email auth code",
		"verification code",
		"security code",
		"access code",
		": code",
		"code:",
		"enter code",
		"type the code",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// Write sends input to the PTY (e.g., 2FA code).
func (r *PTYRunner) Write(input string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return fmt.Errorf("runner is stopped")
	}

	// Append newline if not present
	if !strings.HasSuffix(input, "\n") {
		input += "\n"
	}

	_, err := r.ptmx.Write([]byte(input))
	return err
}

// Output returns the channel for reading PTY stdout lines.
func (r *PTYRunner) Output() <-chan string {
	return r.output
}

// PromptCh returns the channel that signals when 2FA input is needed.
func (r *PTYRunner) PromptCh() <-chan string {
	return r.promptCh
}

// Wait blocks until the DepotDownloader process exits.
func (r *PTYRunner) Wait() error {
	return <-r.done
}

// Stop terminates the DepotDownloader process.
func (r *PTYRunner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return nil
	}
	r.stopped = true

	if r.ptmx != nil {
		r.ptmx.Close()
	}
	return nil
}

// GetPID returns the process ID.
func (r *PTYRunner) GetPID() int {
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Pid
	}
	return 0
}

// DrainOutput reads all remaining output (used for non-interactive collection).
func (r *PTYRunner) DrainOutput(w io.Writer) {
	for line := range r.output {
		fmt.Fprintln(w, line)
	}
}

// IsNotFoundError checks if a DepotDownloader output line indicates the workshop
// item (app_id / pubfile_id) does not exist or is not accessible.
// Patterns are ordered from most specific to least specific to avoid
// broader patterns matching before more precise ones.
func IsNotFoundError(line string) (bool, string) {
	lower := strings.ToLower(line)
	type pattern struct {
		key string
		msg string
	}
	patterns := []pattern{
		{"unable to locate manifest", "工坊文件不存在，无法定位清单 (Unable to locate manifest)"},
		{"no depot license", "Steam 账号没有该游戏的许可 (No depot license)"},
		{"published file not found", "工坊文件不存在 (Published file not found)"},
		{"app not found", "游戏 AppID 不存在 (App not found)"},
		{"invalid published file", "无效的工坊文件 ID (Invalid published file)"},
		{"invalid app", "无效的 AppID (Invalid app)"},
		{"no subscription", "没有订阅权限 (No subscription)"},
		{"no license", "没有许可 (No license)"},
		{"access denied", "访问被拒绝 (Access denied)"},
		{"rate limit", "Steam API 限流，请稍后重试 (Rate limited)"},
		{"connection refused", "无法连接到 Steam 服务器 (Connection refused)"},
		{"could not connect", "无法连接到 Steam 服务器 (Could not connect)"},
		{"password is incorrect", "Steam 密码错误 (Password is incorrect)"},
		{"username is invalid", "Steam 用户名无效 (Username is invalid)"},
		{"login failed", "Steam 登录失败 (Login failed)"},
		// Cautious broader patterns — check after all specific ones
		{"not found", "资源未找到 (Not found)"},
		{"timeout", "连接 Steam 超时 (Timeout)"},
	}
	for _, p := range patterns {
		if strings.Contains(lower, p.key) {
			return true, p.msg
		}
	}
	return false, ""
}
