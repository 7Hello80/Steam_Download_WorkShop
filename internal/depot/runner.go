package depot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"steam-download-tool/internal/storage"
)

// RunDownload executes DepotDownloader and waits for completion.
// Returns the output directory containing downloaded files, or an error.
func RunDownload(ctx context.Context, depotBinPath, taskID string, appID, pubfileID int64, username, password, outputDir string, loginID int, onOutput func(string), onPrompt func(string) string) (string, error) {
	// Validate DepotDownloader exists
	if _, err := os.Stat(depotBinPath); os.IsNotExist(err) {
		return "", fmt.Errorf("DepotDownloader not found at %s", depotBinPath)
	}

	// Ensure output directory
	if err := storage.EnsureDir(outputDir); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	runner := NewRunner(taskID)
	defer runner.Stop()

	if err := runner.Start(ctx, depotBinPath, appID, pubfileID, username, password, outputDir, loginID); err != nil {
		return "", fmt.Errorf("start DepotDownloader: %w", err)
	}

	// Monitor output and handle prompts
	go func() {
		for line := range runner.Output() {
			if onOutput != nil {
				onOutput(line)
			}
		}
	}()

	// Handle 2FA prompts
	go func() {
		for prompt := range runner.PromptCh() {
			if onPrompt != nil {
				input := onPrompt(prompt)
				if input != "" {
					runner.Write(input)
				}
			}
		}
	}()

	// Wait for completion
	if err := runner.Wait(); err != nil {
		return "", fmt.Errorf("DepotDownloader exited with error: %w", err)
	}

	// Verify output directory has files
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return "", fmt.Errorf("read output dir: %w", err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no files downloaded (output directory is empty)")
	}

	return outputDir, nil
}

// ZipAndMove zips the download output and moves it to static storage.
func ZipAndMove(taskID, outputDir, staticDir string) (string, string, int64, error) {
	// Create task directory in static
	staticTaskDir := filepath.Join(staticDir, taskID)
	if err := storage.EnsureDir(staticTaskDir); err != nil {
		return "", "", 0, fmt.Errorf("create static dir: %w", err)
	}

	// Generate zip filename
	zipFilename := fmt.Sprintf("workshop_%s.zip", taskID)
	zipPath := filepath.Join(staticTaskDir, zipFilename)

	// Zip the output directory
	if err := storage.ZipDirectory(outputDir, zipPath); err != nil {
		return "", "", 0, fmt.Errorf("zip directory: %w", err)
	}

	// Get zip file size
	info, err := os.Stat(zipPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("stat zip file: %w", err)
	}
	fileSize := info.Size()

	// Remove the output directory after zipping
	if err := storage.RemoveDirectory(outputDir); err != nil {
		// Non-fatal: log but don't fail
		fmt.Printf("Warning: failed to remove output dir %s: %v\n", outputDir, err)
	}

	return zipPath, zipFilename, fileSize, nil
}
