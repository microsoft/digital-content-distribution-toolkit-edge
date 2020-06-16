package main

import (
	"fmt"

	syscall "golang.org/x/sys/unix"
)

type DiskStatus struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Avail uint64 `json:"avail"`
}

func DiskUsage(path string) (disk DiskStatus) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.Total = fs.Blocks * uint64(fs.Bsize)
	disk.Avail = fs.Bavail * uint64(fs.Bsize)
	disk.Used = disk.Total - disk.Avail
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
	// fmt.Printf("Total: %.2f GB\n", float64(disk.Total)/float64(GB))
	// fmt.Printf("Avail: %.2f GB\n", float64(disk.Avail)/float64(GB))
	// fmt.Printf("Used: %.2f GB\n", float64(disk.Used)/float64(GB))
	result := fmt.Sprintf("%.2fGB", float64(disk.Avail)/float64(GB)) + "/" + fmt.Sprintf("%.2fGB", float64(disk.Total)/float64(GB))
	return result
}
