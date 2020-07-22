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
		log.Println("[DeleteContentAfterValidity] Error", fmt.Sprintf("%s", err))
		logger.Log("Error", "DeleteContentAfterValidity", map[string]string{"Message": err.Error()})
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
		log.Println("[DeleteContentAfterValidity] Checking Actual Path:", prefix)
		log.Println("[DeleteContentAfterValidity] Checking corresponding Abstracted path:", abstractedPath)
		err := checkDeadlineAndDelete(abstractedPath, prefix)
		if err != nil {
			log.Println("[DeleteContentAfterValidity] Error", fmt.Sprintf("%s", err))
			logger.Log("Error", "DeleteContentAfterValidity", map[string]string{"Message": err.Error()})
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
		log.Println("[DeleteContentAfterValidity] Validity date expired. Deleting ", actualPathPrefix)
		deletesize := getDirSizeinMB(path)
		hierarchy := strings.Split(strings.Trim(actualPathPrefix, "/"), "/")
		err := fs.RecursiveDeleteFolder(hierarchy)
		if err != nil {
			return err
		}
		logger.Log("Telemetry", "DeleteContentAfterValidity", map[string]string{
			"Status": "SUCCESS", "FolderPath": actualPathPrefix,
			"Size": fmt.Sprintf("%f", deletesize) + " MB", "Message": "Recursively deleted the folder"})
	}
	return nil
}
