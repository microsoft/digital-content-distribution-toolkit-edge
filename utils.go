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
	"time"

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

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

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

func getDiskInfo() string {
	disk := DiskUsage("./")
	result := fmt.Sprintf("%.2f", float64(disk.Avail)/float64(MB)) + "/" + fmt.Sprintf("%.2f", float64(disk.Total)/float64(MB))
	return result
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

func storeHashsum(pathdir, filename, hash string) error {
	fileHashStr := filename + "=>" + hash + "\n"
	fmt.Println("StoreHashsum: ", pathdir)
	f, err := os.OpenFile(filepath.Join(pathdir, "hashsum.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fileHashStr)
	if err != nil {
		return err
	}
	return nil
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

func testContentSyncInfo(interval int) {
	for true {
		logger.Log("ContentSyncInfo", "ContentSyncInfo", map[string]string{"DownloadStatus": "SUCCESS", "FolderPath": "a/b/c", "AssetSize": fmt.Sprintf("%.2f", 213.56), "Channel": "SES", "AssetUpdate": "No"})
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
