package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func deleteContent(interval int) {
	for true {
		time.Sleep(time.Duration(interval) * time.Second)
		fmt.Println("--------Checking for validity dates of the Contents----------")
		traverse(fs.GetHomeNode(), "")
	}
}
func traverse(node []byte, prefix string) string {
	children, err := fs.GetChildrenForNode(node)
	if err != nil {
		log.Println("[DeleteContentOnExpiry] Error", fmt.Sprintf("%s", err))
		logger.Log("Error", "DeleteContentOnExpiry", map[string]string{"Message": err.Error()})
	}
	if len(children) == fs.GetNodeLength() && strings.LastIndex(prefix, "/") >= 0 {
		prefix = prefix[:strings.LastIndex(prefix, "/")]
		return prefix
	}
	for i := 0; i < len(children); i += fs.GetNodeLength() {
		if i == 0 {
			continue
		}
		actualName, _ := fs.GetFolderNameForNode(children[i : i+fs.GetNodeLength()])
		prefix := prefix + "/" + actualName
		abstractedPath := filepath.Join(fs.GetHomeFolder(), string(children[i:i+fs.GetNodeLength()]))
		log.Println("[DeleteContentOnExpiry] Checking Actual Path:", prefix)
		log.Println("[DeleteContentOnExpiry] Checking corresponding Abstracted path:", abstractedPath)
		err := checkDeadlineAndDelete(abstractedPath, prefix)
		if err != nil {
			log.Println("[DeleteContentOnExpiry] Error", fmt.Sprintf("%s", err))
			logger.Log("Error", "DeleteContentOnExpiry", map[string]string{"Message": err.Error()})
			return prefix
		}
		prefix = traverse(children[i:i+fs.GetNodeLength()], prefix)
	}
	return prefix
}

func checkDeadlineAndDelete(path string, actualPathPrefix string) error {
	f, err := os.OpenFile(filepath.Join(path, "deadline.txt"), os.O_RDONLY, 0700)
	if err != nil {
		log.Println(err)
		return err
	}
	defer f.Close()
	b := make([]byte, 64)
	read, err := f.Read(b)
	deadline, _ := strconv.ParseInt(string(b[:read]), 10, 64)
	curr := time.Now().Unix()
	if curr > deadline {
		log.Println("[DeleteContentOnExpiry] Validity date expired. Deleting ", actualPathPrefix)
		//deletesize := getDirSizeinMB(path)
		hierarchy := strings.Split(strings.Trim(actualPathPrefix, "/"), "/")
		err := fs.RecursiveDeleteFolder(hierarchy)
		if err != nil {
			return err
		}
		logger.Log("Telemetry", "ContentDeleteInfo", map[string]string{
			"DeleteStatus": "SUCCESS", "FolderPath": actualPathPrefix,
			"Mode": "Expired"})
		logger.Log("Telemetry", "HubStorage", map[string]string{"AvailableDiskSpace(MB)": getDiskInfo()})

	}
	return nil
}
