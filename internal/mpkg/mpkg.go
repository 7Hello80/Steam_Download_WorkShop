package mpkg

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nigeltao/etc2/lib/etc2"
	"github.com/pierrec/lz4/v4"
)

const (
	mpkgVersion = "PKGM0018"
	// WallpaperEngineAppID is the Steam app ID for Wallpaper Engine.
	WallpaperEngineAppID = 431960
)

// videoExtensions lists file extensions recognized as video files.
var videoExtensions = map[string]bool{
	".mp4":  true,
	".avi":  true,
	".mkv":  true,
	".webm": true,
	".mov":  true,
	".wmv":  true,
	".m4v":  true,
	".flv":  true,
}

// imageExtensions lists file extensions recognized as preview images.
var imageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".webp": true,
}

// IsVideoFile checks if a filename has a recognized video extension.
func IsVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return videoExtensions[ext]
}

// IsImageFile checks if a filename has a recognized image extension.
func IsImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return imageExtensions[ext]
}

// WorkshopInfo holds information extracted from a workshop zip.
type WorkshopInfo struct {
	VideoFiles   []string
	PreviewImage string
	Title        string
	IsVideoType  bool
	HasScenePkg  bool // whether scene.pkg exists in the zip (scene-type wallpaper)
}

// AnalyzeZip reads a workshop zip and extracts information about its contents.
func AnalyzeZip(zipPath string) (*WorkshopInfo, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	info := &WorkshopInfo{}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(f.Name)

		// Skip DepotDownloader metadata
		if strings.HasPrefix(f.Name, ".DepotDownloader/") || strings.HasPrefix(f.Name, ".DepotDownloader") {
			continue
		}

		if strings.EqualFold(name, "project.json") {
			data, err := readZipFile(f)
			if err == nil {
				info.Title = extractJSONField(data, "title")
				fileType := extractJSONField(data, "type")
				info.IsVideoType = strings.EqualFold(fileType, "video")
			}
			continue
		}

		if strings.EqualFold(name, "scene.pkg") {
			info.HasScenePkg = true
			continue
		}

		if IsVideoFile(name) {
			info.VideoFiles = append(info.VideoFiles, f.Name)
			continue
		}

		if IsImageFile(name) && info.PreviewImage == "" {
			info.PreviewImage = f.Name
		}
	}

	return info, nil
}

// ConvertZipToMPKG converts a workshop zip file to mpkg binary format using
// a two-pass streaming approach so large files (100MB+) don't exhaust memory.
//
// Pass 1 — collect metadata (names + sizes only, no data read):
//   Build the header section and compute data offsets.
//
// Pass 2 — stream data:
//   Re-open the zip and stream each file's bytes directly from the zip reader
//   to the output file, keeping only a small buffer in memory.
//
// MPKG format:
//
//	[4 bytes LE: version_string_length]
//	[version_string_bytes ("PKGM0018")]
//	[4 bytes LE: file_count]
//	For each file:
//	  [4 bytes LE: name_length]
//	  [name_bytes]
//	  [4 bytes LE: index]     — cumulative byte offset of this file's data
//	  [4 bytes LE: size]      — size of this file's data
//	[all file data concatenated in order]
func ConvertZipToMPKG(zipPath, outputPath string) error {
	// --- Pass 1: collect metadata only (names and uncompressed sizes) ---
	type fileMeta struct {
		name string
		size int64
	}

	var entries []fileMeta
	if err := func() error {
		r, err := zip.OpenReader(zipPath)
		if err != nil {
			return fmt.Errorf("open zip: %w", err)
		}
		defer r.Close()

		for _, f := range r.File {
			if f.FileInfo().IsDir() {
				continue
			}
			if strings.HasPrefix(f.Name, ".DepotDownloader/") || strings.HasPrefix(f.Name, ".DepotDownloader") {
				continue
			}
			// Exclude DepotDownloader log files from workshop downloads
			if strings.EqualFold(f.Name, "terminal.log") {
				continue
			}
			entries = append(entries, fileMeta{
				name: f.Name,
				size: int64(f.UncompressedSize64),
			})
		}
		return nil
	}(); err != nil {
		return err
	}

	if len(entries) == 0 {
		return fmt.Errorf("no files found in zip")
	}

	// Create output and write header
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}

	// Track whether we need to close on error
	writeOK := false
	defer func() {
		if !writeOK {
			out.Close()
			os.Remove(outputPath)
		}
	}()

	// Write version
	verBytes := []byte(mpkgVersion)
	if err := writeInt32LE(out, int32(len(verBytes))); err != nil {
		return err
	}
	if _, err := out.Write(verBytes); err != nil {
		return err
	}

	// Write file count
	if err := writeInt32LE(out, int32(len(entries))); err != nil {
		return err
	}

	// Write header entries (names + index + size)
	var cumulativeIndex int64
	for _, e := range entries {
		nameBytes := []byte(e.name)
		if err := writeInt32LE(out, int32(len(nameBytes))); err != nil {
			return err
		}
		if _, err := out.Write(nameBytes); err != nil {
			return err
		}
		if err := writeInt32LE(out, int32(cumulativeIndex)); err != nil {
			return err
		}
		if err := writeInt32LE(out, int32(e.size)); err != nil {
			return err
		}
		cumulativeIndex += e.size
	}

	// --- Pass 2: stream file data from zip directly to output ---
	if err := func() error {
		r, err := zip.OpenReader(zipPath)
		if err != nil {
			return fmt.Errorf("reopen zip for data pass: %w", err)
		}
		defer r.Close()

		// Build lookup: filename -> zip.File for the data pass
		// The entries slice order matches the header order; find matching zip entries
		zipByName := make(map[string]*zip.File, len(r.File))
		for _, f := range r.File {
			zipByName[f.Name] = f
		}

		buf := make([]byte, 256*1024) // 256KB streaming buffer for faster I/O
		for _, e := range entries {
			zf, ok := zipByName[e.name]
			if !ok {
				return fmt.Errorf("zip entry %s not found on second pass", e.name)
			}
			rc, err := zf.Open()
			if err != nil {
				return fmt.Errorf("open zip entry %s: %w", e.name, err)
			}
			// Stream from zip entry to output file via buffer
			if _, err := io.CopyBuffer(out, rc, buf); err != nil {
				rc.Close()
				return fmt.Errorf("stream %s: %w", e.name, err)
			}
			rc.Close()
		}
		return nil
	}(); err != nil {
		return err
	}

	if err := out.Close(); err != nil {
		return err
	}
	writeOK = true
	return nil
}

// writeInt32LE writes a 32-bit integer in little-endian byte order.
func writeInt32LE(w io.Writer, v int32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	_, err := w.Write(buf[:])
	return err
}

// readInt32LE reads a 32-bit little-endian integer from a reader.
func readInt32LE(r io.Reader) (int32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("read int32: %w", err)
	}
	return int32(binary.LittleEndian.Uint32(buf[:])), nil
}

// UnpackMPKG extracts all files from an mpkg binary file into outputDir.
// Supports both "PKGM0018" (mobile) and "PKGV0023" (PC/Steam Workshop) version strings.
//
// Binary format:
//
//	[4 bytes LE: version_string_length]
//	[version_string_bytes]
//	[4 bytes LE: file_count]
//	For each file:
//	  [4 bytes LE: name_length]
//	  [name_bytes]
//	  [4 bytes LE: index]  — cumulative byte offset of this file's data
//	  [4 bytes LE: size]
//	[all file data concatenated in order]
func UnpackMPKG(inputPath, outputDir string) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer f.Close()

	// Read and validate version
	verLen, err := readInt32LE(f)
	if err != nil {
		return fmt.Errorf("read version length: %w", err)
	}
	verBytes := make([]byte, verLen)
	if _, err := io.ReadFull(f, verBytes); err != nil {
		return fmt.Errorf("read version string: %w", err)
	}
	version := string(verBytes)
	// Accept all PKG-prefixed versions — the binary format is identical across versions
	if !strings.HasPrefix(version, "PKG") {
		return fmt.Errorf("unsupported mpkg version: %q (expected PKG-prefixed version)", version)
	}

	// Read file count
	fileCount, err := readInt32LE(f)
	if err != nil {
		return fmt.Errorf("read file count: %w", err)
	}
	if fileCount <= 0 {
		return fmt.Errorf("mpkg contains 0 files")
	}

	// Parse header entries
	type entry struct {
		name string
		size int64
	}
	entries := make([]entry, 0, fileCount)
	var expectedIndex int64
	for i := int32(0); i < fileCount; i++ {
		nameLen, err := readInt32LE(f)
		if err != nil {
			return fmt.Errorf("read name length for entry %d: %w", i, err)
		}
		nameBytes := make([]byte, nameLen)
		if _, err := io.ReadFull(f, nameBytes); err != nil {
			return fmt.Errorf("read name for entry %d: %w", i, err)
		}

		index, err := readInt32LE(f)
		if err != nil {
			return fmt.Errorf("read index for entry %d: %w", i, err)
		}
		if int64(index) != expectedIndex {
			return fmt.Errorf("entry %d (%q): expected index %d, got %d (corrupt mpkg)",
				i, string(nameBytes), expectedIndex, index)
		}

		size, err := readInt32LE(f)
		if err != nil {
			return fmt.Errorf("read size for entry %d: %w", i, err)
		}

		entries = append(entries, entry{
			name: string(nameBytes),
			size: int64(size),
		})
		expectedIndex += int64(size)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Stream data section to output files
	buf := make([]byte, 256*1024) // 256KB streaming buffer
	for _, entry := range entries {
		outputPath := filepath.Join(outputDir, entry.name)

		// Create parent directories
		parentDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("create parent dir for %q: %w", entry.name, err)
		}

		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("create output file %q: %w", entry.name, err)
		}

		// Stream exactly entry.size bytes from input to output
		if _, err := io.CopyBuffer(outFile, io.LimitReader(f, entry.size), buf); err != nil {
			outFile.Close()
			return fmt.Errorf("write file %q: %w", entry.name, err)
		}

		if err := outFile.Close(); err != nil {
			return fmt.Errorf("close file %q: %w", entry.name, err)
		}
	}

	return nil
}

// shaderIntRe matches all digit sequences in a shader line.
// Used by convertLineIntsToFloats to find candidates for int→float conversion.
var shaderIntRe = regexp.MustCompile(`\d+`)

// ProcessSceneFiles post-processes files unpacked from a PC scene.pkg to make them
// compatible with the mobile Wallpaper Engine format.
//
// Transformations applied:
//   - materials/*.json: replace genericimage4/3 → genericimage2, fix combos {}
//   - shaders/**/*.vert: add g_PointerState uniform, convert int→float literals
//   - shaders/**/*.frag: add g_PointerState uniform + pointer scale line, convert int→float literals
func ProcessSceneFiles(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return nil // skip paths we can't relativize
		}
		relPath = filepath.ToSlash(relPath)

		ext := strings.ToLower(filepath.Ext(relPath))

		// --- Material JSON files (all levels, including effects/) ---
		if strings.HasPrefix(relPath, "materials/") && ext == ".json" {
			return processMaterialJSON(path, relPath, info)
		}

		// --- Vertex shader files ---
		if ext == ".vert" {
			return processVertShader(path, info)
		}

		// --- Fragment shader files ---
		if ext == ".frag" {
			return processFragShader(path, info)
		}

		// --- Texture files (.tex) --- convert PC format to mobile-compatible
		if ext == ".tex" {
			return convertTexForMobile(path, info)
		}

		return nil
	})
}

// processMaterialJSON fixes a material JSON file for mobile compatibility.
func processMaterialJSON(path, relPath string, info os.FileInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", relPath, err)
	}

	modified := strings.ReplaceAll(string(data),
		`"combos" : {}`,
		`"combos" : {"VERSION" : 2}`)
	modified = strings.ReplaceAll(modified,
		`"combos":{}`,
		`"combos":{"VERSION":2}`)

	// Replace genericimage4/3 shader with genericimage2 (mobile shader)
	modified = strings.ReplaceAll(modified,
		`"shader" : "genericimage4"`,
		`"shader" : "genericimage2"`)
	modified = strings.ReplaceAll(modified,
		`"shader": "genericimage4"`,
		`"shader": "genericimage2"`)
	modified = strings.ReplaceAll(modified,
		`"shader" : "genericimage3"`,
		`"shader" : "genericimage2"`)
	modified = strings.ReplaceAll(modified,
		`"shader": "genericimage3"`,
		`"shader": "genericimage2"`)

	if modified != string(data) {
		if err := os.WriteFile(path, []byte(modified), info.Mode()); err != nil {
			return fmt.Errorf("write %s: %w", relPath, err)
		}
	}
	return nil
}

// processVertShader converts a PC vertex shader to mobile-compatible format.
// Transformations:
//  1. Add g_PointerState uniform before void main() (mobile touch/pointer support)
//  2. Convert integer literals to float literals (OpenGL ES requirement)
func processVertShader(path string, info os.FileInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	modified := content

	// Detect line ending style
	lineEnding := detectLineEnding(modified)

	// Normalize to \n for processing
	modified = strings.ReplaceAll(modified, "\r\n", "\n")

	// Add g_PointerState uniform before void main() if not already present
	if !strings.Contains(modified, "g_PointerState") {
		modified = addUniformBeforeMain(modified, "uniform vec4 g_PointerState;")
	}

	// Convert integer literals to float literals (skip preprocessor lines)
	modified = convertShaderIntsToFloats(modified)

	// Rename reserved words that conflict with GLSL ES 3.00 keywords on mobile GPUs.
	// Some mobile GPU drivers (e.g. Mali, Adreno) treat "sample" as a reserved word
	// related to multisampling operations, even though it's not in the GLSL ES 3.00 spec.
	// Renaming to "baseSample" avoids this conflict while keeping the variable clear.
	modified = renameReservedShaderWords(modified)

	// Restore line endings
	if lineEnding == "\r\n" {
		modified = strings.ReplaceAll(modified, "\n", "\r\n")
	}

	if modified != content {
		return os.WriteFile(path, []byte(modified), info.Mode())
	}
	return nil
}

// processFragShader converts a PC fragment shader to mobile-compatible format.
// Transformations:
//  1. Add g_PointerState uniform after varying declarations
//  2. Add pointer scale adjustment before UV normalization (unprojectedUVs += 0.5)
//  3. Convert integer literals to float literals (OpenGL ES requirement)
func processFragShader(path string, info os.FileInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	modified := content

	// Detect line ending style
	lineEnding := detectLineEnding(modified)

	// Normalize to \n for processing
	modified = strings.ReplaceAll(modified, "\r\n", "\n")

	// Add g_PointerState uniform after last varying, before first uniform
	if !strings.Contains(modified, "g_PointerState") {
		modified = addUniformAfterVaryings(modified, "uniform vec4 g_PointerState;")
	}

	// Add pointer scale line before "unprojectedUVs += 0.5;" if not already present
	if !strings.Contains(modified, "CAST2(1.0 / max(0.00001, g_PointerState.y))") {
		modified = addPointerScaleLine(modified)
	}

	// Convert integer literals to float literals (skip preprocessor lines)
	modified = convertShaderIntsToFloats(modified)

	// Rename reserved words that conflict with GLSL ES 3.00 keywords on mobile GPUs.
	modified = renameReservedShaderWords(modified)

	// Restore line endings
	if lineEnding == "\r\n" {
		modified = strings.ReplaceAll(modified, "\n", "\r\n")
	}

	if modified != content {
		return os.WriteFile(path, []byte(modified), info.Mode())
	}
	return nil
}

// detectLineEnding detects the line ending style of the content.
// Returns "\r\n" for CRLF or "\n" for LF.
func detectLineEnding(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

// addUniformBeforeMain inserts a uniform declaration before the "void main()" line.
// The uniform replaces the blank line that typically precedes void main().
// If no blank line precedes void main(), the uniform is inserted on its own line
// before void main() with a blank line separating them.
func addUniformBeforeMain(content, uniformLine string) string {
	// Strategy 1: Insert before void main() with blank line preceding it
	re := regexp.MustCompile(`\n\n([ \t]*)void\s+main\s*\(\)`)
	if re.MatchString(content) {
		return re.ReplaceAllString(content, "\n"+uniformLine+"\n\n${1}void main()")
	}

	// Strategy 2: Insert before void main() when on the very next line (no blank line)
	re2 := regexp.MustCompile(`\n([ \t]*)void\s+main\s*\(\)`)
	if re2.MatchString(content) {
		return re2.ReplaceAllString(content, "\n"+uniformLine+"\n\n${1}void main()")
	}

	// No void main() found — return unchanged
	return content
}

// addUniformAfterVaryings inserts a uniform declaration after the last varying
// declaration, before the blank line that separates varyings from uniforms.
// If no blank line follows the varyings, the uniform is inserted directly after
// the last varying with a blank line for separation.
func addUniformAfterVaryings(content, uniformLine string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inserted := false
	sawVarying := false
	lastVaryingIdx := -1
	var varyingIndent string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		isVarying := strings.HasPrefix(trimmed, "varying ")

		if isVarying {
			sawVarying = true
			lastVaryingIdx = i
			// Capture indentation of varying declarations for consistent formatting
			if varyingIndent == "" {
				varyingIndent = line[:len(line)-len(trimmed)]
			}
		}

		// Insert after varying block: when we hit the blank line after varyings
		if !inserted && sawVarying && !isVarying && trimmed == "" {
			result = append(result, uniformLine)
			inserted = true
		}

		result = append(result, line)

		// Once we hit a non-empty, non-varying line we've left the varying block
		if !isVarying && trimmed != "" {
			sawVarying = false
		}
	}

	// If we found varyings but no blank line after them, insert the uniform
	// right after the last varying line
	if !inserted && lastVaryingIdx >= 0 {
		var newResult []string
		for i, line := range result {
			newResult = append(newResult, line)
			if i == lastVaryingIdx {
				newResult = append(newResult, uniformLine)
				newResult = append(newResult, "") // blank line for separation
			}
		}
		result = newResult
		inserted = true
	}

	// Fallback: if we didn't insert anywhere, try before void main
	if !inserted {
		return addUniformBeforeMain(content, uniformLine)
	}

	return strings.Join(result, "\n")
}

// addPointerScaleLine inserts the mobile pointer scale adjustment line before
// the "unprojectedUVs += 0.5;" line which is common in pointer-based effects.
func addPointerScaleLine(content string) string {
	// Match any leading whitespace + "unprojectedUVs += 0.5;"
	re := regexp.MustCompile(`(?m)^[ \t]*unprojectedUVs \+= 0\.5;`)
	if !re.MatchString(content) {
		return content // no match, nothing to do
	}
	// Insert the pointer scale line with one tab, and remove indentation from the
	// existing += 0.5 line to match the mobile format.
	return re.ReplaceAllString(content,
		"\tunprojectedUVs *= CAST2(1.0 / max(0.00001, g_PointerState.y));\nunprojectedUVs += 0.5;")
}

// convertShaderIntsToFloats converts integer literals to float literals in shader
// source code. Preprocessor lines (starting with #) and JSON metadata comment lines
// (containing { or }) are left unchanged.
// This is required for OpenGL ES which disallows implicit int→float conversion.
func convertShaderIntsToFloats(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip preprocessor directives
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Skip JSON metadata comments (e.g. // {"material":"...","default":0})
		// These contain integer values that are part of JSON, not GLSL.
		if strings.HasPrefix(trimmed, "//") && (strings.Contains(trimmed, "{") || strings.Contains(trimmed, "}")) {
			continue
		}
		lines[i] = convertLineIntsToFloats(line)
	}
	return strings.Join(lines, "\n")
}

// convertLineIntsToFloats finds integer literals in a single shader line and appends
// ".0" to convert them to float literals. An integer literal is a digit sequence
// that is NOT:
//   - Part of a float literal (preceded or followed by '.')
//   - Part of an identifier (preceded or followed by [a-zA-Z_])
//   - An array index (inside brackets like [0])
//   - Inside an inline JSON comment (after // that contains { or })
//
// If the line contains an inline JSON comment (e.g. "uniform float g_Foo; // {\"default\":1}"),
// only the code portion before "//" is converted; the comment is preserved verbatim.
func convertLineIntsToFloats(line string) string {
	codePart := line
	commentPart := ""

	// Split at inline comment if it contains JSON metadata (curly braces)
	if idx := strings.Index(line, "//"); idx >= 0 {
		rest := line[idx:]
		if strings.Contains(rest, "{") || strings.Contains(rest, "}") {
			codePart = line[:idx]
			commentPart = line[idx:]
		}
	}

	matches := shaderIntRe.FindAllStringIndex(codePart, -1)
	if len(matches) == 0 {
		return line // no changes
	}

	// Process matches in order, building result with a strings.Builder
	var b strings.Builder
	lastEnd := 0
	for _, m := range matches {
		start, end := m[0], m[1]

		// Check character before: must not be a word char, bracket, or '.'
		if start > 0 {
			prev := codePart[start-1]
			if isShaderWordChar(prev) || prev == ']' || prev == '[' || prev == '.' {
				continue // part of identifier, array index, or float literal
			}
		}

		// Check character after: must not be a word char, bracket, or '.'
		if end < len(codePart) {
			next := codePart[end]
			if isShaderWordChar(next) || next == '[' || next == ']' || next == '.' {
				continue
			}
		}

		// Standalone integer — write everything up to here, then the int + ".0"
		b.WriteString(codePart[lastEnd:start])
		b.WriteString(codePart[start:end])
		b.WriteString(".0")
		lastEnd = end
	}
	b.WriteString(codePart[lastEnd:])
	b.WriteString(commentPart)
	return b.String()
}

// isShaderWordChar reports whether c is a letter, digit, or underscore.
func isShaderWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (c >= '0' && c <= '9')
}

// renameReservedShaderWords renames variable identifiers that conflict with
// reserved words in GLSL ES 3.00 on certain mobile GPU implementations.
//
// In particular, "sample" is treated as a reserved word by some Mali and Adreno
// GPU drivers (related to gl_SampleID / gl_SamplePosition in GLSL 4.00+),
// even though it is not listed as reserved in the GLSL ES 3.00 specification.
//
// This function renames standalone occurrences of "sample" to "baseSample"
// across the entire shader source, excluding comment lines and preprocessor
// directives.
func renameReservedShaderWords(content string) string {
	re := regexp.MustCompile(`\bsample\b`)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip preprocessor directives
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Skip JSON metadata comments (e.g. // {"material":"...","default":0})
		if strings.HasPrefix(trimmed, "//") && (strings.Contains(trimmed, "{") || strings.Contains(trimmed, "}")) {
			continue
		}

		// Split at inline comment if it contains JSON metadata
		codePart := line
		commentPart := ""
		if idx := strings.Index(line, "//"); idx >= 0 {
			rest := line[idx:]
			if strings.Contains(rest, "{") || strings.Contains(rest, "}") {
				codePart = line[:idx]
				commentPart = line[idx:]
			}
		}

		// Only rename in the code portion
		if re.MatchString(codePart) {
			newCode := re.ReplaceAllString(codePart, "baseSample")
			lines[i] = newCode + commentPart
		}
	}
	return strings.Join(lines, "\n")
}

// =============================================================================
// Texture (.tex) conversion — PC format 0 (ARGB8888) → mobile-compatible
// =============================================================================
//
// Wallpaper Engine TEXV0005 texture format:
//   TEXV0005\0          — container header (9 bytes)
//   TEXI0001\0          — texture info block (9 bytes)
//     format            — TextureFormat enum (0=ARGB8888, 4=DXT5, 5=GPU, ...)
//     flags             — texture flags
//     textureWidth      — power-of-2 GPU allocation width
//     textureHeight     — power-of-2 GPU allocation height
//     width             — actual image width
//     height            — actual image height
//     unknown1          — texture ID/hash (preserved across platforms)
//   TEXB0003\0          — image data container (9 bytes)
//     bField1..bField5  — sub-header fields (format-dependent)
//     bWidth, bHeight   — data dimensions
//     bField4, bField5  — additional parameters
//   [data]              — image payload (JPEG via FreeImage for format 0;
//                         GPU-compressed for format 5)
//
// Format 0 (ARGB8888): image data is stored as JPEG via FreeImage, indicated
// by an optional "fX\0" marker before the JPEG SOI. When decoded, it becomes
// standard RGBA8 for GPU use.

const (
	texMagicContainer = "TEXV0005"
	texMagicInfo      = "TEXI0001"
	texMagicBody      = "TEXB0003"
	texFormatARGB8888 = 0 // PC format: ARGB8888, typically stored as JPEG via FreeImage
	texFormatGPU      = 5 // Mobile format: GPU-compressed (ETC2/ASTC)

	// Format 5 (mobile GPU-compressed) TEXB field values observed in mobile samples.
	// These differ from the PC format 0 TEXB fields and are necessary for the mobile
	// Wallpaper Engine app to correctly interpret the texture data.
	texBField2Mobile = 0xFFFFFFFF
	texBField3Mobile = 1
	texBField4Mobile = 1
	texBField5Mobile = 0x0084bfc0

	// Format 5 data section constants.
	texFormat5Magic = 0x0100ff22 // LE: 22 ff 00 01, constant across all mobile samples
)

// texv5Header holds the parsed header of a TEXV0005 container.
type texv5Header struct {
	format        uint32 // TextureFormat enum (0=ARGB8888)
	flags         uint32 // Texture flags (was "containerCount")
	textureWidth  uint32 // Power-of-2 GPU allocation width (was skipped as "field1")
	textureHeight uint32 // Power-of-2 GPU allocation height (was skipped as "field2")
	width         uint32 // Actual image width
	height        uint32 // Actual image height
	unknown1      uint32 // Texture ID/hash — preserved across PC↔mobile
	// TEXB0003 sub-header
	bField1  uint32
	bField2  uint32
	bField3  uint32
	bWidth   uint32
	bHeight  uint32
	bField4  uint32
	bField5  uint32
	dataOff      int  // byte offset where image data begins
	hasFXMarker  bool // true if 0x0f + "fX\0" marker precedes image data
}

// maxMobileDim is the maximum texture dimension for mobile. Images larger than
// this are scaled down to reduce ETC2 encoding time and GPU memory usage.
const maxMobileDim = 2048

// convertTexForMobile converts a PC-format .tex file to a mobile-compatible
// format 5 (GPU-compressed ETC2) .tex.
//
// Handles two PC texture formats:
//   - Format 0 (ARGB8888): JPEG/FreeImage color textures → ETC2 RGB8
//   - Format 9 (R8): LZ4-compressed single-channel masks → ETC2 R11
//
// The conversion pipeline:
//  1. Parse the TEXV0005 container
//  2. Decode the image data (JPEG for format 0, LZ4-block R8 for format 9)
//  3. Scale down if larger than maxMobileDim
//  4. Compress to the appropriate ETC2 variant
//  5. Build a format 5 container matching the mobile sample structure
func convertTexForMobile(path string, info os.FileInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if len(data) < 64 || string(data[:7]) != "TEXV000" {
		return nil
	}

	// Parse TEXV0005 and TEXI0001 headers (common to all formats)
	if len(data) < 9+9+28 {
		return nil
	}
	rd := bytes.NewReader(data[9+9:]) // skip TEXV0005\0 TEXI0001\0
	hdr := &texv5Header{}
	hdr.format = readU32LE(rd)
	hdr.flags = readU32LE(rd)
	hdr.textureWidth = readU32LE(rd)
	hdr.textureHeight = readU32LE(rd)
	hdr.width = readU32LE(rd)
	hdr.height = readU32LE(rd)
	hdr.unknown1 = readU32LE(rd)

	switch hdr.format {
	case texFormatARGB8888: // Format 0: ARGB8888 JPEG/FreeImage
		return convertTexFormat0(data, path, hdr, info)
	case 9: // Format 9: R8 single-channel mask (LZ4-compressed)
		return convertTexFormat9(data, path, hdr, info)
	default:
		return nil // Unknown format, leave as-is
	}
}

// convertTexFormat0 converts a PC format 0 (ARGB8888/JPEG) texture to mobile
// format 5 (ETC2 RGB8).
func convertTexFormat0(data []byte, path string, hdr *texv5Header, info os.FileInfo) error {
	// Find JPEG data within the TEXB0003 container
	_, jpegData, err := parseTexv5(data)
	if err != nil {
		return nil
	}
	if len(jpegData) == 0 {
		return nil
	}

	// Step 1: Decode JPEG to raw RGBA.
	img, err := jpeg.Decode(bytes.NewReader(jpegData))
	if err != nil {
		return nil
	}

	// Step 2: Scale down for mobile if necessary.
	img, imgW, imgH := scaleForMobile(img)

	// Step 3: Compress to ETC2 RGB8 (8 bytes per 4x4 block).
	var etc2Buf bytes.Buffer
	if err := etc2.Encode(&etc2Buf, img, etc2.FormatETC2RGB, nil); err != nil {
		return nil
	}
	etc2Data := etc2Buf.Bytes()

	// Step 4: Build format 5 container.
	newData, err := rebuildTexv5Format5(hdr, etc2Data, imgW, imgH)
	if err != nil {
		return nil
	}

	return os.WriteFile(path, newData, info.Mode())
}

// convertTexFormat9 converts a PC format 9 (R8 single-channel, LZ4-compressed)
// mask texture to mobile format 5 (ETC2 R11 unsigned).
//
// Format 9 structure (TEXB0004 container, per RePKG reference):
//
//	TEXV0005\0 TEXI0001\0 [TEXI fields: format=9, …]
//	TEXB0004\0
//	imageCount  (4 bytes LE)
//	FreeImageFormat (4 bytes LE, typically 0xFFFFFFFF = FIF_UNKNOWN)
//	isVideoMp4  (4 bytes LE, typically 0)
//	mipmapCount (4 bytes LE, typically 1)
//	For each mipmap:
//	  width, height, isLZ4Compressed, decompressedBytesCount, byteCount
//	  [LZ4 block-compressed R8 pixel data, decompresses to w*h bytes]
func convertTexFormat9(data []byte, path string, hdr *texv5Header, info os.FileInfo) error {
	// Parse TEXB0004 container and LZ4-decompress the R8 mask data.
	r8Data, mipW, mipH, err := parseTexFormat9(data)
	if err != nil {
		return nil
	}
	if len(r8Data) == 0 {
		return nil
	}

	// Step 1: Create a grayscale image from R8 data.
	gray := image.NewGray(image.Rect(0, 0, int(mipW), int(mipH)))
	copy(gray.Pix, r8Data)

	// Step 2: Scale down for mobile if necessary.
	scaledImg, _, _ := scaleForMobile(gray)

	// Step 3: JPEG encode (fast — milliseconds, not minutes like ETC2_R11).
	// Masks are typically small and compress very well as JPEG grayscale.
	var jpegBuf bytes.Buffer
	if err := jpeg.Encode(&jpegBuf, scaledImg, &jpeg.Options{Quality: 80}); err != nil {
		return nil
	}

	// Step 4: Build format 0 container (ARGB8888/JPEG).
	// Format 0 with NPOT dimensions works on mobile for mask textures;
	// the previous display issue was caused by old code paths, not format 0 itself.
	hdr.format = texFormatARGB8888 // override format 9 → 0 for mobile compatibility
	newData, err := rebuildTexv5(hdr, jpegBuf.Bytes())
	if err != nil {
		return nil
	}

	return os.WriteFile(path, newData, info.Mode())
}

// parseTexFormat9 parses a TEXB0004 format 9 container and returns the
// decompressed R8 pixel data along with the mipmap dimensions.
//
// Based on the RePKG reference implementation:
//
//	https://github.com/notscuffed/repkg
func parseTexFormat9(data []byte) (pixels []byte, width, height uint32, err error) {
	if len(data) < 90 {
		return nil, 0, 0, fmt.Errorf("file too small for TEXV0005 format 9")
	}

	// Skip TEXV0005\0 (9) + TEXI0001\0 (9) + TEXI fields (28) = 46 bytes
	pos := 46

	// Read TEXB magic
	if pos+9 > len(data) {
		return nil, 0, 0, fmt.Errorf("unexpected EOF at TEXB magic")
	}
	magic := string(data[pos : pos+8])
	pos += 9 // skip magic + null

	isTEXB0004 := magic == "TEXB0004"

	// Read imageCount (TEXB0003: 7 fields then image data;
	// TEXB0004: 3 header fields then mipmap data)
	var imageCount uint32
	if isTEXB0004 {
		if pos+12 > len(data) {
			return nil, 0, 0, fmt.Errorf("unexpected EOF at TEXB0004 header")
		}
		imageCount = binary.LittleEndian.Uint32(data[pos:])
		pos += 4
		// FreeImageFormat (4 bytes) — skip
		pos += 4
		// isVideoMp4 (4 bytes) — skip
		pos += 4
	} else {
		// TEXB0003: skip 7×4=28 bytes of TEXB fields, then imageCount
		if pos+32 > len(data) {
			return nil, 0, 0, fmt.Errorf("unexpected EOF at TEXB0003 header")
		}
		pos += 28
		imageCount = binary.LittleEndian.Uint32(data[pos:])
		pos += 4
	}

	if imageCount == 0 {
		return nil, 0, 0, fmt.Errorf("format 9 tex has no images")
	}

	// Read mipmapCount (first image)
	mipmapCount := binary.LittleEndian.Uint32(data[pos:])
	pos += 4
	if mipmapCount == 0 {
		return nil, 0, 0, fmt.Errorf("format 9 tex has no mipmaps")
	}

	// Read the first (largest) mipmap
	// V3/V4 (non-MP4) mipmap format:
	//   width, height, isLZ4Compressed, decompressedBytesCount, byteCount, bytes
	mipWidth := binary.LittleEndian.Uint32(data[pos:])
	pos += 4
	mipHeight := binary.LittleEndian.Uint32(data[pos:])
	pos += 4
	isLZ4 := binary.LittleEndian.Uint32(data[pos:])
	pos += 4
	decompBytes := binary.LittleEndian.Uint32(data[pos:])
	pos += 4
	byteCount := binary.LittleEndian.Uint32(data[pos:])
	pos += 4

	if int(pos)+int(byteCount) > len(data) {
		return nil, 0, 0, fmt.Errorf("mipmap byteCount %d exceeds remaining data", byteCount)
	}

	mipBytes := data[pos : pos+int(byteCount)]

	if isLZ4 != 0 {
		// LZ4 block decompress
		decompressed := make([]byte, decompBytes)
		n, err := lz4.UncompressBlock(mipBytes, decompressed)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("lz4 decompress format 9 mipmap: %w", err)
		}
		if n != len(decompressed) {
			return nil, 0, 0, fmt.Errorf("lz4 decompress: expected %d bytes, got %d", len(decompressed), n)
		}
		return decompressed, mipWidth, mipHeight, nil
	}

	return mipBytes, mipWidth, mipHeight, nil
}

// scaleForMobile scales an image down if either dimension exceeds maxMobileDim.
// Returns the (possibly scaled) image and its width/height.
func scaleForMobile(img image.Image) (image.Image, int, int) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= maxMobileDim && h <= maxMobileDim {
		return img, w, h
	}

	// Compute new dimensions preserving aspect ratio.
	var newW, newH int
	if w > h {
		newW = maxMobileDim
		newH = h * maxMobileDim / w
	} else {
		newH = maxMobileDim
		newW = w * maxMobileDim / h
	}

	// Simple nearest-neighbor downscale using the standard library.
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for dy := 0; dy < newH; dy++ {
		sy := dy * h / newH
		for dx := 0; dx < newW; dx++ {
			sx := dx * w / newW
			dst.Set(dx, dy, img.At(sx, sy))
		}
	}

	return dst, newW, newH
}

// convertTexForMobileFallback re-encodes JPEG at reduced quality and keeps
// format 0. Used when ETC2 compression fails for any reason.
func convertTexForMobileFallback(path string, info os.FileInfo, hdr *texv5Header, jpegData []byte) error {
	img, err := jpeg.Decode(bytes.NewReader(jpegData))
	if err != nil {
		return nil
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 55}); err != nil {
		return nil
	}

	newData, err := rebuildTexv5(hdr, buf.Bytes())
	if err != nil {
		return nil
	}

	return os.WriteFile(path, newData, info.Mode())
}

// parseTexv5 parses a TEXV0005 container and returns the header and the raw
// JPEG data embedded within it (for format 0 / ARGB8888 textures).
//
// The TEXV0005 binary layout:
//
//	TEXV0005\0           9 bytes: container magic
//	TEXI0001\0           9 bytes: info block magic
//	format               4 bytes: TextureFormat enum
//	flags                4 bytes: texture flags
//	textureWidth         4 bytes: power-of-2 GPU width
//	textureHeight        4 bytes: power-of-2 GPU height
//	width                4 bytes: actual image width
//	height               4 bytes: actual image height
//	unknown1             4 bytes: texture ID/hash
//	TEXB0003\0           9 bytes: body block magic
//	bField1..bField5     7 × 4 bytes: sub-header fields
//	bWidth, bHeight
//	bField4, bField5
//	[variable marker]   0–4 bytes: optional "0f fX\0" or other prefix
//	JPEG data            starts with FF D8
//
// Robustness: Instead of assuming a fixed marker structure before JPEG data,
// we search forward from the end of the TEXB header for the JPEG SOI marker
// (FF D8). This handles all observed PC format-0 variants.
func parseTexv5(data []byte) (*texv5Header, []byte, error) {
	if len(data) < 90 {
		return nil, nil, fmt.Errorf("file too small for TEXV0005")
	}

	rd := bytes.NewReader(data)

	// Skip "TEXV0005\0" (9 bytes: 8 chars + null terminator)
	rd.Seek(9, io.SeekStart)

	// Read "TEXI0001\0" (9 bytes: 8 chars + null terminator)
	magic := make([]byte, 9)
	if _, err := io.ReadFull(rd, magic); err != nil {
		return nil, nil, err
	}
	if string(magic[:7]) != "TEXI000" || magic[8] != 0x00 {
		return nil, nil, fmt.Errorf("expected TEXI0001 block, got %q", magic)
	}

	// Read TEXI data fields (7 × uint32 LE)
	hdr := &texv5Header{}
	hdr.format = readU32LE(rd)
	hdr.flags = readU32LE(rd)
	hdr.textureWidth = readU32LE(rd)
	hdr.textureHeight = readU32LE(rd)
	hdr.width = readU32LE(rd)
	hdr.height = readU32LE(rd)
	hdr.unknown1 = readU32LE(rd)

	// Read "TEXB0003\0" (9 bytes: 8 chars + null terminator)
	magic = make([]byte, 9)
	if _, err := io.ReadFull(rd, magic); err != nil {
		return nil, nil, err
	}
	if string(magic[:7]) != "TEXB000" || magic[8] != 0x00 {
		return nil, nil, fmt.Errorf("expected TEXB0003 block, got %q", magic)
	}

	// Read TEXB data fields (7 × uint32 LE)
	hdr.bField1 = readU32LE(rd)
	hdr.bField2 = readU32LE(rd)
	hdr.bField3 = readU32LE(rd)
	hdr.bWidth = readU32LE(rd)
	hdr.bHeight = readU32LE(rd)
	hdr.bField4 = readU32LE(rd)
	hdr.bField5 = readU32LE(rd)

	// Search forward for JPEG SOI marker (FF D8) — robust across all
	// observed marker variants (0f 66 58 00, 50 14 54 00, bb ce 58 00, etc.)
	pos, _ := rd.Seek(0, io.SeekCurrent)
	jpegStart := -1
	for i := int(pos); i < len(data)-1; i++ {
		if data[i] == 0xff && data[i+1] == 0xd8 {
			jpegStart = i
			break
		}
	}
	if jpegStart < 0 {
		return nil, nil, fmt.Errorf("no JPEG data found in TEXV0005 container")
	}

	// Check if a 0x0f + "fX\0" marker immediately precedes the JPEG data
	if jpegStart >= 4 &&
		data[jpegStart-4] == 0x0f &&
		data[jpegStart-3] == 'f' &&
		data[jpegStart-2] == 'X' &&
		data[jpegStart-1] == 0x00 {
		hdr.hasFXMarker = true
	}

	hdr.dataOff = jpegStart

	// Extract JPEG data from SOI to EOI (FF D9)
	jpegData := data[jpegStart:]
	jpegEnd := findJPEGEnd(jpegData)
	jpegData = jpegData[:jpegEnd]

	// Verify it looks like JPEG data
	if len(jpegData) < 4 || jpegData[0] != 0xff || jpegData[1] != 0xd8 {
		return nil, nil, fmt.Errorf("no JPEG header at data offset %d", jpegStart)
	}

	return hdr, jpegData, nil
}

// rebuildTexv5 rebuilds a TEXV0005 container with new JPEG data, using the
// parsed header and applying mobile-compatible conventions:
//   - textureWidth/textureHeight set to actual dimensions (NPOT, matching mobile)
//   - All other TEXI/TEXB fields preserved from original
//   - fX marker preserved if originally present; added as default otherwise
func rebuildTexv5(hdr *texv5Header, newJPEG []byte) ([]byte, error) {
	var buf bytes.Buffer

	// TEXV0005\0
	buf.WriteString(texMagicContainer)
	buf.WriteByte(0x00)

	// TEXI0001\0
	buf.WriteString(texMagicInfo)
	buf.WriteByte(0x00)

	// TEXI fields: set textureWidth/Height to actual dimensions (NPOT)
	// matching the mobile convention observed in the mobile samples.
	writeU32LE(&buf, hdr.format)
	writeU32LE(&buf, hdr.flags)
	writeU32LE(&buf, hdr.width)       // textureWidth = actual width (NPOT)
	writeU32LE(&buf, hdr.height)      // textureHeight = actual height (NPOT)
	writeU32LE(&buf, hdr.width)       // width
	writeU32LE(&buf, hdr.height)      // height
	writeU32LE(&buf, hdr.unknown1)    // preserved texture ID/hash

	// TEXB0003\0
	buf.WriteString(texMagicBody)
	buf.WriteByte(0x00)

	// TEXB fields: preserve original structure
	writeU32LE(&buf, hdr.bField1)
	writeU32LE(&buf, hdr.bField2)
	writeU32LE(&buf, hdr.bField3)
	writeU32LE(&buf, hdr.bWidth)
	writeU32LE(&buf, hdr.bHeight)
	writeU32LE(&buf, hdr.bField4)
	writeU32LE(&buf, hdr.bField5)

	// Write fX marker — the standard FreeImage JPEG indicator for TEXV0005
	// format 0 (ARGB8888). Required for the reader to identify the data as
	// FreeImage-encoded JPEG rather than raw RGBA pixels.
	buf.WriteByte(0x0f)
	buf.WriteString("fX")
	buf.WriteByte(0x00)

	// JPEG data
	buf.Write(newJPEG)

	return buf.Bytes(), nil
}

// rebuildTexv5Format5 builds a TEXV0005 format 5 (GPU-compressed ETC2) container
// matching the structure observed in the mobile Wallpaper Engine samples.
//
// Format 5 container layout (based on 样本/手机端/ reference files):
//
//	TEXV0005\0           9 bytes: container magic
//	TEXI0001\0           9 bytes: info block magic
//	format=5             4 bytes
//	flags                4 bytes (preserved from PC)
//	textureWidth         4 bytes = actual width (NPOT)
//	textureHeight        4 bytes = actual height (NPOT)
//	width                4 bytes
//	height               4 bytes
//	unknown1             4 bytes (preserved texture ID/hash)
//	TEXB0003\0           9 bytes: body block magic
//	bField1=1            4 bytes
//	bField2=0xFFFFFFFF   4 bytes (mobile-specific)
//	bField3=1            4 bytes (mobile-specific)
//	bWidth               4 bytes
//	bHeight              4 bytes
//	bField4=1            4 bytes (mobile-specific)
//	bField5              4 bytes = width*height (pixel count for GPU allocation)
//	[4 bytes: remaining data size (LE)]
//	[4 bytes: 22 ff 00 01 magic (LE)]
//	[ETC2 compressed data]
func rebuildTexv5Format5(hdr *texv5Header, etc2Data []byte, imgW, imgH int) ([]byte, error) {
	var buf bytes.Buffer

	// TEXV0005\0
	buf.WriteString(texMagicContainer)
	buf.WriteByte(0x00)

	// TEXI0001\0
	buf.WriteString(texMagicInfo)
	buf.WriteByte(0x00)

	// TEXI fields for format 5 (GPU-compressed)
	writeU32LE(&buf, texFormatGPU)  // format = 5
	writeU32LE(&buf, hdr.flags)     // flags (preserved)
	writeU32LE(&buf, uint32(imgW))  // textureWidth = actual (NPOT)
	writeU32LE(&buf, uint32(imgH))  // textureHeight = actual (NPOT)
	writeU32LE(&buf, uint32(imgW))  // width
	writeU32LE(&buf, uint32(imgH))  // height
	writeU32LE(&buf, hdr.unknown1)  // preserved texture ID/hash

	// TEXB0003\0
	buf.WriteString(texMagicBody)
	buf.WriteByte(0x00)

	// TEXB fields for format 5 (mobile-specific values)
	// bField5 stores the total pixel count (width * height), used by the mobile
	// Wallpaper Engine app for GPU memory allocation before decompression.
	pixelCount := uint32(imgW) * uint32(imgH)
	writeU32LE(&buf, 1)                  // bField1 = 1
	writeU32LE(&buf, texBField2Mobile)   // bField2 = 0xFFFFFFFF
	writeU32LE(&buf, texBField3Mobile)   // bField3 = 1
	writeU32LE(&buf, uint32(imgW))       // bWidth
	writeU32LE(&buf, uint32(imgH))       // bHeight
	writeU32LE(&buf, texBField4Mobile)   // bField4 = 1
	writeU32LE(&buf, pixelCount)         // bField5 = pixel count (width*height)

	// Data section: [4-byte remaining size] [4-byte magic] [ETC2 data]
	// Remaining size = size of everything after this 4-byte field
	remainingSize := uint32(4 + len(etc2Data)) // magic(4) + etc2Data
	writeU32LE(&buf, remainingSize)
	writeU32LE(&buf, texFormat5Magic) // 22 ff 00 01
	buf.Write(etc2Data)

	return buf.Bytes(), nil
}

// findJPEGEnd returns the length of JPEG data (from SOI FF D8 to EOI FF D9).
// It searches backward from the end to find the JPEG end-of-image marker.
func findJPEGEnd(data []byte) int {
	for i := len(data) - 1; i > 0; i-- {
		if data[i] == 0xd9 && data[i-1] == 0xff {
			return i + 1
		}
	}
	return len(data)
}

// readU32LE reads a little-endian uint32 from a reader.
func readU32LE(r io.Reader) uint32 {
	var v uint32
	binary.Read(r, binary.LittleEndian, &v)
	return v
}

// writeU32LE writes a little-endian uint32 to a writer.
func writeU32LE(w io.Writer, v uint32) {
	binary.Write(w, binary.LittleEndian, v)
}

// PatchProjectJSON adds mobile-required fields to project.json in the given directory.
// - Adds "workshopurl" field pointing to the Steam Workshop page
// - Normalizes "type" field casing to "Scene" (mobile app is case-sensitive)
func PatchProjectJSON(dir string, pubfileID int64) error {
	projectPath := filepath.Join(dir, "project.json")
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return nil // project.json is optional for scene wallpapers
	}

	content := string(data)

	// Fix type field casing — mobile app expects "Scene" (capital S), but some
	// wallpaper creators set it as "scene" (lowercase). The mobile app appears
	// to be case-sensitive about this value.
	content = strings.ReplaceAll(content, `"type" : "scene"`, `"type" : "Scene"`)
	content = strings.ReplaceAll(content, `"type": "scene"`, `"type": "Scene"`)

	// Only add workshopurl if it doesn't already exist
	if !strings.Contains(content, "workshopurl") {
		workshopURL := fmt.Sprintf("https://steamcommunity.com/sharedfiles/filedetails/?id=%d", pubfileID)

		// Insert workshopurl before the final closing brace.
		lastBrace := strings.LastIndex(content, "}")
		if lastBrace < 0 {
			return nil
		}

		// Find the indentation of the closing brace to match formatting
		braceLineStart := strings.LastIndex(content[:lastBrace], "\n")
		indent := ""
		if braceLineStart >= 0 {
			braceLine := content[braceLineStart+1 : lastBrace]
			indent = braceLine[:len(braceLine)-len(strings.TrimLeft(braceLine, " \t"))]
		}

		// Determine line ending from content
		lineEnd := "\n"
		if strings.Contains(content, "\r\n") {
			lineEnd = "\r\n"
		}

		// Insert with proper comma placement: append comma after the last field,
		// then add workshopurl on a new line with matching indentation.
		modified := content[:lastBrace] + "," + lineEnd +
			indent + `"workshopurl" : "` + workshopURL + `"` + lineEnd +
			content[lastBrace:]

		if err := os.WriteFile(projectPath, []byte(modified), 0644); err != nil {
			return fmt.Errorf("write project.json: %w", err)
		}
	}

	return nil
}

// readZipFile reads the entire contents of a zip file entry.
func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// extractJSONField is a minimal JSON field extractor. It looks for "field" : "value"
// patterns in raw JSON bytes without importing encoding/json.
func extractJSONField(data []byte, field string) string {
	search := []byte(`"` + field + `"`)
	idx := bytes.Index(data, search)
	if idx < 0 {
		return ""
	}

	rest := data[idx+len(search):]

	colonIdx := bytes.IndexByte(rest, ':')
	if colonIdx < 0 {
		return ""
	}
	rest = rest[colonIdx+1:]

	rest = bytes.TrimLeft(rest, " \t\r\n")

	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]

	endIdx := bytes.IndexByte(rest, '"')
	if endIdx < 0 {
		return ""
	}

	return string(rest[:endIdx])
}
