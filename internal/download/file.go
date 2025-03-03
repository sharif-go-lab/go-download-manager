package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadFile downloads a file from a given URL (with speed limit) and saves it to the specified directory.
func DownloadFile(url string, destDir string, speedLimitKbps int) (string, error) {
	// Extract file name from URL
	fileName := filepath.Base(url)
	destPath := filepath.Join(destDir, fileName)

	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create download directory: %v", err)
	}

	// Open the file for writing
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer outFile.Close()

	// Send HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Apply speed limit if set
	var reader io.Reader = resp.Body
	if speedLimitKbps > 0 {
		reader = NewSpeedLimiter(resp.Body, speedLimitKbps)
	}

	// Write file in chunks
	_, err = io.Copy(outFile, reader)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return destPath, nil
}