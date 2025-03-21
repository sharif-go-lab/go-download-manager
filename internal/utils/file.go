package utils

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func FileName(resp *http.Response) string {
	cd := resp.Header.Get("Content-Disposition")
	if cd != "" && strings.Contains(cd, "filename=") {
		parts := strings.Split(cd, "filename=")
		return strings.Trim(parts[1], "\"")
	}

	filename := path.Base(resp.Request.URL.Path)
	if filename == "" || filename == "/" {
		contentType := resp.Header.Get("Content-Type")
		exts, err := mime.ExtensionsByType(contentType)
		if err == nil && len(exts) > 0 {
			return "unknown_file" + exts[0]
		}
		return "unknown_file"
	}
	return filename
}
// findUniqueFilePath checks if `path` exists. If it does, it appends (1), (2), etc.
// before the file extension until it finds a path that does not exist.
func FindUniqueFilePath(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	nameOnly := strings.TrimSuffix(base, ext)

	candidate := path
	i := 1

	for {
		_, err := os.Stat(candidate)
		if os.IsNotExist(err) {
			// This candidate doesn't exist, so we can use it
			return candidate
		}
		// File exists, so build a new candidate with (i)
		candidate = filepath.Join(
			dir,
			fmt.Sprintf("%s(%d)%s", nameOnly, i, ext),
		)
		i++
	}
}
