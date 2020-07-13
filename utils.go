package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
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
	fmt.Printf("\r%s", strings.Repeat(" ", 50))
	fmt.Printf("\r Downloaded: %d MB of %d MB", (pw.Counter / 1024 / 1024), pw.Total)
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
func getDirSizeinMB(path string) float64 {
	var size float64 = 0
	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			//fmt.Println(path, info.Size())
			size += float64(info.Size())
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	sizeMB := (size) / 1024.0 / 1024.0
	return math.Round(sizeMB*100) / 100
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
