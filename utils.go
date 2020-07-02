package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//ProgressWriter this prints progress of a download
type ProgressWriter struct {
	Counter int64
	Total   int64
}
type FileInfo struct {
	Name, Hashsum string
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

func matchSHA256(filePath string, trueSHA string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if actual == trueSHA {
		return nil
	}
	return errors.New("hashsum did not match")
}
func getDirSizeinMB(path string) int64 {
	var size int64 = 0
	calcsize := func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	}
	filepath.Walk(filepath.Join(path, "bulkfiles"), calcsize)
	filepath.Walk(filepath.Join(path, "metadatafiles"), calcsize)
	return (size) / 1024.0 / 1024.0
}
func storeDeadline(path, deadline string) error {
	if _, f_err := os.Stat(filepath.Join(path, "deadline.txt")); os.IsNotExist(f_err) {
		f, err := os.OpenFile(filepath.Join(path, "deadline.txt"), os.O_RDWR|os.O_CREATE, 0700)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(deadline)
		if err != nil {
			return err
		}
	}
	return nil
}
