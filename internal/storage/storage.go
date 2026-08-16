package storage

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// copyBufferSize is the buffer size used for io.Copy operations.
const copyBufferSize = 2 * 1024 * 1024 // 2MB

// fileEntry represents a file to be added to a zip archive.
type fileEntry struct {
	path string
	info os.FileInfo
}

// zipResult holds the result of reading a file in parallel.
type zipResult struct {
	idx  int
	data []byte
	err  error
}

// ZipDirectory compresses a source directory into a zip file.
// Uses Store (no compression) since workshop files are already compressed assets.
// This dramatically speeds up zip creation for large workshop content.
func ZipDirectory(srcDir, destZip string) error {
	if err := os.MkdirAll(filepath.Dir(destZip), 0755); err != nil {
		return fmt.Errorf("create zip parent dir: %w", err)
	}

	zipFile, err := os.Create(destZip)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}

	// Use buffered writer for faster writes
	bufWriter := NewBufferedWriter(zipFile)
	zw := zip.NewWriter(bufWriter)

	// Collect all files first so we can process them
	var files []fileEntry

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}
		files = append(files, fileEntry{path: path, info: info})
		return nil
	})
	if err != nil {
		zipFile.Close()
		return err
	}

	// Process files concurrently for large directories, sequentially for small ones.
	const parallelThreshold = 10
	var processErr error
	if len(files) > parallelThreshold {
		processErr = zipFilesParallel(zw, srcDir, files)
	} else {
		processErr = zipFilesSequential(zw, srcDir, files)
	}

	// Always finalize the zip writer and flush buffered data, regardless of errors.
	if closeErr := zw.Close(); closeErr != nil && processErr == nil {
		processErr = closeErr
	}
	if flushErr := bufWriter.Flush(); flushErr != nil && processErr == nil {
		processErr = flushErr
	}
	if closeErr := zipFile.Close(); closeErr != nil && processErr == nil {
		processErr = closeErr
	}
	if processErr != nil {
		os.Remove(destZip)
	}
	return processErr
}

// zipFilesSequential processes files one by one.
func zipFilesSequential(zw *zip.Writer, srcDir string, files []fileEntry) error {
	buf := make([]byte, copyBufferSize)
	for _, f := range files {
		if err := addFileToZip(zw, srcDir, f.path, f.info, buf); err != nil {
			return err
		}
	}
	return nil
}

// zipFilesParallel processes files concurrently using a worker pool.
// Each worker adds files to the zip writer in order (zip.Writer is not
// safe for concurrent use, so we serialize the zip writes).
type zipJob struct {
	idx  int
	path string
	info os.FileInfo
}

func zipFilesParallel(zw *zip.Writer, srcDir string, files []fileEntry) error {
	const workers = 4

	jobs := make(chan zipJob, len(files))
	results := make(chan zipResult, len(files))

	// Worker pool — just reads file contents in parallel
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, copyBufferSize)
			for j := range jobs {
				data, err := readFileContents(j.path, j.info, buf)
				results <- zipResult{idx: j.idx, data: data, err: err}
			}
		}()
	}

	// Submit jobs
	for i, f := range files {
		jobs <- zipJob{idx: i, path: f.path, info: f.info}
	}
	close(jobs)

	// Collect results in a goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Write to zip in order
	resultsByIndex := make(map[int]zipResult, len(files))
	nextIdx := 0
	for r := range results {
		if r.err != nil {
			return r.err
		}
		resultsByIndex[r.idx] = r

		// Write all consecutive results we have
		for {
			res, ok := resultsByIndex[nextIdx]
			if !ok {
				break
			}
			delete(resultsByIndex, nextIdx)
			nextIdx++

			if err := writeDataToZip(zw, srcDir, res, files); err != nil {
				return err
			}
		}
	}

	return nil
}

// readFileContents reads a file into memory for parallel zip processing.
func readFileContents(path string, info os.FileInfo, buf []byte) ([]byte, error) {
	if info.IsDir() {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data := make([]byte, 0, info.Size())
	for {
		n, err := f.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// writeDataToZip writes a parallel-read result to the zip writer.
func writeDataToZip(zw *zip.Writer, srcDir string, res zipResult, files []fileEntry) error {
	f := files[res.idx]
	relPath, err := filepath.Rel(srcDir, f.path)
	if err != nil {
		return err
	}
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	header, err := zip.FileInfoHeader(f.info)
	if err != nil {
		return err
	}
	header.Name = relPath
	header.Method = zip.Store // No compression — workshop files are already compressed

	if f.info.IsDir() {
		header.Name += "/"
		_, err := zw.CreateHeader(header)
		return err
	}

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	if len(res.data) > 0 {
		_, err = writer.Write(res.data)
	}
	return err
}

// addFileToZip adds a single file to the zip archive.
func addFileToZip(zw *zip.Writer, srcDir, path string, info os.FileInfo, buf []byte) error {
	relPath, err := filepath.Rel(srcDir, path)
	if err != nil {
		return err
	}
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = relPath
	header.Method = zip.Store // No compression — workshop files are already compressed

	if info.IsDir() {
		header.Name += "/"
		_, err := zw.CreateHeader(header)
		return err
	}

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.CopyBuffer(writer, file, buf)
	return err
}

// BufferedWriter wraps os.File with buffered writing for large zip output.
type BufferedWriter struct {
	file *os.File
	buf  []byte
	n    int
}

func NewBufferedWriter(f *os.File) *BufferedWriter {
	return &BufferedWriter{
		file: f,
		buf:  make([]byte, copyBufferSize),
	}
}

func (bw *BufferedWriter) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		n := copy(bw.buf[bw.n:], p)
		bw.n += n
		p = p[n:]
		total += n

		if bw.n == len(bw.buf) {
			if err := bw.Flush(); err != nil {
				return total, err
			}
		}
	}
	return total, nil
}

func (bw *BufferedWriter) Flush() error {
	if bw.n == 0 {
		return nil
	}
	_, err := bw.file.Write(bw.buf[:bw.n])
	bw.n = 0
	return err
}

// MoveDirectory moves all contents from src to dest.
func MoveDirectory(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create dest parent: %w", err)
	}
	return os.Rename(src, dest)
}

// RemoveDirectory removes a directory and all its contents.
func RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}

// GetDirSize returns the total size of all files in a directory.
func GetDirSize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// ExtractZip extracts all files from a zip archive into a destination directory.
// Skips directory entries and .DepotDownloader/ metadata files.
// Uses streaming copy with a 256KB buffer to keep memory usage low.
func ExtractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}

	buf := make([]byte, 256*1024) // 256KB streaming buffer
	for _, f := range r.File {
		// Skip directories
		if f.FileInfo().IsDir() {
			continue
		}

		// Skip DepotDownloader metadata
		baseName := filepath.Base(f.Name)
		if strings.HasPrefix(f.Name, ".DepotDownloader/") || strings.HasPrefix(f.Name, ".DepotDownloader") ||
			strings.HasPrefix(baseName, ".DepotDownloader") {
			continue
		}

		outputPath := filepath.Join(destDir, f.Name)

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("create parent dir for %q: %w", f.Name, err)
		}

		// Open zip entry
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %q: %w", f.Name, err)
		}

		// Create output file
		outFile, err := os.Create(outputPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create output file %q: %w", f.Name, err)
		}

		// Stream copy
		if _, err := io.CopyBuffer(outFile, rc, buf); err != nil {
			rc.Close()
			outFile.Close()
			return fmt.Errorf("extract %q: %w", f.Name, err)
		}

		rc.Close()
		outFile.Close()
	}

	return nil
}
