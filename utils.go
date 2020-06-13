package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

//ProgressWriter this prints progress of a download
type ProgressWriter struct {
	Counter int64
	Total   int64
}

func (pw *ProgressWriter) Write(b []byte) (int, error) {
	written := len(b)
	pw.Counter += int64(written)
	pw.print()
	return written, nil
}

func (pw *ProgressWriter) print() {
	fmt.Printf("\r%s", strings.Repeat(" ", 40))
	fmt.Printf("\rDownloaded: %d MB of %d MB", (pw.Counter / 1024 / 1024), pw.Total)
}

func calculateSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
