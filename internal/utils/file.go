package utils

import (
	"mime"
	"net/http"
	"path"
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