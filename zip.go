package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func extractZip(strZipPath string, objLogger *Logger) (string, error) {
	// Create temp directory
	strTempDir, err := os.MkdirTemp("", "zoho_attachments_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	objLogger.Log(fmt.Sprintf("Extracting %s to temp directory %s", strZipPath, strTempDir))

	// Open zip file
	objZip, err := zip.OpenReader(strZipPath)
	if err != nil {
		os.RemoveAll(strTempDir)
		return "", fmt.Errorf("failed to open zip file: %w", err)
	}
	defer objZip.Close()

	// Extract each file
	for _, objFile := range objZip.File {
		err := extractZipFile(objFile, strTempDir)
		if err != nil {
			os.RemoveAll(strTempDir)
			return "", fmt.Errorf("failed to extract %s: %w", objFile.Name, err)
		}
	}

	objLogger.Log(fmt.Sprintf("Extracted %d files from %s", len(objZip.File), strZipPath))
	return strTempDir, nil
}

func extractZipFile(objFile *zip.File, strDestDir string) error {
	// Sanitize path — prevent zip slip attack
	strDestPath := filepath.Join(strDestDir, objFile.Name)
	if !strings.HasPrefix(strDestPath, filepath.Clean(strDestDir)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path in zip: %s", objFile.Name)
	}

	// Create directories if needed
	if objFile.FileInfo().IsDir() {
		return os.MkdirAll(strDestPath, 0755)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(strDestPath), 0755); err != nil {
		return err
	}

	// Extract file
	objSrc, err := objFile.Open()
	if err != nil {
		return err
	}
	defer objSrc.Close()

	objDest, err := os.Create(strDestPath)
	if err != nil {
		return err
	}
	defer objDest.Close()

	_, err = io.Copy(objDest, objSrc)
	return err
}

func isZipFile(strPath string) bool {
	return strings.ToLower(filepath.Ext(strPath)) == ".zip"
}
