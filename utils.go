package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	syscall "golang.org/x/sys/unix"
)

//ProgressWriter this prints progress of a download
type ProgressWriter struct {
	Counter int64
	Total   int64
}
type FileInfo struct {
	Name, Hashsum string
}

type DiskStatus struct {
	Total uint64
	Avail uint64
}

func DiskUsage(path string) (disk DiskStatus) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.Total = fs.Blocks * uint64(fs.Bsize)
	disk.Avail = fs.Bavail * uint64(fs.Bsize)
	return
}

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

func getDiskInfo() string {
	disk := DiskUsage("./")
	result := fmt.Sprintf("%.2fGB", float64(disk.Avail)/float64(GB)) + "/" + fmt.Sprintf("%.2fGB", float64(disk.Total)/float64(GB))
	return result
}

func (pw *ProgressWriter) Write(b []byte) (int, error) {
	written := len(b)
	pw.Counter += int64(written)
	pw.print()
	return written, nil
}

func (pw *ProgressWriter) print() {
	fmt.Printf("\r%s", strings.Repeat(" ", 60))
	fmt.Printf("\rDownloaded: %d MB of %d MB", (pw.Counter / 1024 / 1024), pw.Total)
	fmt.Println()
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
	sizeMB := (size) / float64(MB)
	return sizeMB
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
