package controller

import (
	"exam-paper/internal/model"
	parser "exam-paper/internal/service/parser"
	"exam-paper/internal/utils"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (ctl *Controller) Static(c *gin.Context) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		writeError(c, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := filepath.Clean(strings.TrimPrefix(c.Request.URL.Path, "/"))
	if path == "." || path == "" {
		path = "index.html"
	}
	if strings.HasPrefix(path, "static"+string(filepath.Separator)) {
		path = strings.TrimPrefix(path, "static"+string(filepath.Separator))
	}
	staticDir := filepath.Join(ctl.rootDir, "static")
	target := filepath.Join(staticDir, path)
	if !utils.Within(target, staticDir) {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.File(target)
}

func (ctl *Controller) Library(c *gin.Context) {
	files, err := ctl.discoverLibrary()
	if err != nil {
		writeError(c, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(c, gin.H{"files": files})
}

func (ctl *Controller) Parse(c *gin.Context) {
	requested := c.Query("path")
	if requested == "" {
		writeError(c, "缺少 path 参数。", http.StatusBadRequest)
		return
	}
	target := filepath.Join(ctl.rootDir, filepath.Clean(requested))
	if !utils.Within(target, ctl.rootDir) {
		writeError(c, "文件路径不在工作目录内。", http.StatusBadRequest)
		return
	}
	result, err := parser.ParseQuestionFile(target, requested)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, result)
}

func (ctl *Controller) Import(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize)
	if err := c.Request.ParseMultipartForm(MaxUploadSize); err != nil {
		writeError(c, "上传文件过大或表单格式不正确。", http.StatusBadRequest)
		return
	}

	var (
		results []model.ParseResult
		total   int
	)
	for fieldName, headers := range c.Request.MultipartForm.File {
		for _, header := range headers {
			result, err := ctl.parseUploadedFile(header, uploadRelativePath(c.Request.MultipartForm.Value, fieldName))
			if err != nil {
				writeError(c, err.Error(), http.StatusBadRequest)
				return
			}
			total += result.QuestionCount
			results = append(results, result)
		}
	}
	if len(results) == 0 {
		writeError(c, "没有收到 PDF 或 DOCX 文件。", http.StatusBadRequest)
		return
	}
	writeJSON(c, gin.H{"questionCount": total, "files": results})
}

func (ctl *Controller) parseUploadedFile(header *multipart.FileHeader, relativePath string) (model.ParseResult, error) {
	name := filepath.Base(header.Filename)
	ext := strings.ToLower(filepath.Ext(name))
	if !utils.FileTypes[ext] {
		return model.ParseResult{}, fmt.Errorf("%s 不是支持的 PDF 或 DOCX 文件。", name)
	}
	file, err := header.Open()
	if err != nil {
		return model.ParseResult{}, err
	}
	defer file.Close()

	target := ctl.uploadTargetPath(name, relativePath)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return model.ParseResult{}, err
	}
	targetFile, err := os.Create(target)
	if err != nil {
		return model.ParseResult{}, err
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, file); err != nil {
		return model.ParseResult{}, err
	}
	if err := targetFile.Close(); err != nil {
		return model.ParseResult{}, err
	}
	rel, err := filepath.Rel(ctl.rootDir, target)
	if err != nil {
		return model.ParseResult{}, err
	}
	rel = filepath.ToSlash(rel)
	result, err := parser.ParseQuestionFile(target, rel)
	if err == nil {
		ctl.bank.RememberQuestions(rel, result.Questions)
		if info, statErr := os.Stat(target); statErr == nil {
			_ = ctl.store.ReplaceSourceQuestions(rel, info, result.Questions)
		}
	}
	return result, err
}

func uploadRelativePath(values map[string][]string, fieldName string) string {
	paths := values[fieldName+"_path"]
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (ctl *Controller) uploadTargetPath(name, relativePath string) string {
	relativePath = filepath.ToSlash(strings.TrimSpace(relativePath))
	if relativePath == "" {
		return filepath.Join(ctl.dataDir, "uploads", uniqueUploadName(name))
	}
	dirParts := cleanRelativeParts(filepath.ToSlash(filepath.Dir(relativePath)))
	if len(dirParts) == 0 {
		return filepath.Join(ctl.dataDir, "uploads", uniqueUploadName(name))
	}
	targetParts := append([]string{ctl.dataDir, "imports"}, dirParts...)
	return nonConflictingPath(filepath.Join(targetParts...), filepath.Base(name))
}

func cleanRelativeParts(path string) []string {
	var parts []string
	for _, part := range strings.Split(path, "/") {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `:\`) {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

func nonConflictingPath(dir, name string) string {
	target := filepath.Join(dir, filepath.Base(name))
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return target
	}
	return filepath.Join(dir, uniqueUploadName(name))
}

func (ctl *Controller) discoverLibrary() ([]model.LibraryFile, error) {
	seen := map[string]bool{}
	var files []model.LibraryFile
	err := filepath.WalkDir(ctl.rootDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") && path != ctl.rootDir {
				return filepath.SkipDir
			}
			if path != ctl.rootDir && ctl.shouldSkipLibraryDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !utils.FileTypes[ext] {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(ctl.rootDir, path)
		if err != nil {
			return err
		}
		rel = utils.CanonicalSource(rel)
		seen[rel] = true
		files = append(files, model.LibraryFile{Name: entry.Name(), Path: rel, Suffix: ext, Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	dbFiles, err := ctl.store.LibrarySources()
	if err != nil {
		return nil, err
	}
	for _, file := range dbFiles {
		if seen[file.Path] {
			continue
		}
		files = append(files, file)
		seen[file.Path] = true
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func (ctl *Controller) shouldSkipLibraryDir(path string) bool {
	rel, err := filepath.Rel(ctl.rootDir, path)
	if err != nil {
		return false
	}
	first := strings.Split(filepath.ToSlash(rel), "/")[0]
	switch first {
	case "dist", ".gocache", ".gocache-sqlite":
		return true
	default:
		return false
	}
}

func uniqueUploadName(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(filepath.Base(name), ext)
	base = strings.TrimSpace(base)
	if base == "" {
		base = "upload"
	}
	return fmt.Sprintf("%s-%d%s", base, time.Now().UnixNano(), ext)
}
