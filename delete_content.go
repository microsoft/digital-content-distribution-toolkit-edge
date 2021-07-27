package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	l "./logger"
)

func deleteContent(interval int) {
	for true {
		time.Sleep(time.Duration(interval) * time.Second)
		log.Println("--------Checking for validity dates of the Contents----------")
		fmt.Println("--------Checking for validity dates of the Contents----------")
		fmt.Println("Printing buckets before deletion")
		fs.PrintBuckets()
		fmt.Println("Printing file sys before deletion")
		fs.PrintFileSystem()
		fmt.Println("====================")
		traverse(fs.GetHomeNode(), "")
		fmt.Println("Printing buckets after deletion")
		fs.PrintBuckets()
		fmt.Println("Printing file sys after deletion")
		fs.PrintFileSystem()
		fmt.Println("====================")
	}
}
func traverse(node []byte, prefix string) string {
	children, err := fs.GetChildrenForNode(node)
	if err != nil {
		log.Println("[DeleteContentOnExpiry] Error", fmt.Sprintf("%s", err))
		sm := l.MessageSubType{StringMessage: "DeleteContentOnExpiry: " + err.Error()}
		logger.Log("Error", &sm)
		//logger.Log("Error", "DeleteContentOnExpiry", map[string]string{"Message": err.Error()})
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
		log.Println("[DeleteContentOnExpiry] Checking folder path:", prefix)
		log.Println("[DeleteContentOnExpiry] Checking corresponding abstracted path:", abstractedPath)
		err := checkDeadlineAndDelete(abstractedPath, prefix)
		if err != nil {
			log.Println("[DeleteContentOnExpiry] Error", fmt.Sprintf("%s", err))
			sm := l.MessageSubType{StringMessage: "DeleteContentOnExpiry: " + err.Error()}
			logger.Log("Error", &sm)
			//logger.Log("Error", "DeleteContentOnExpiry", map[string]string{"Message": err.Error()})
			return prefix
		}
		prefix = traverse(children[i:i+fs.GetNodeLength()], prefix)
	}
	return prefix
}
func getDeadline(path string) (int64, error) {
	f, err := os.OpenFile(filepath.Join(path, "deadline.txt"), os.O_RDONLY, 0700)
	if err != nil {
		log.Println(err)
		return -1, err
	}
	defer f.Close()
	b := make([]byte, 64)
	read, err := f.Read(b)
	deadline, _ := strconv.ParseInt(string(b[:read]), 10, 64)
	return deadline, nil
}
func checkDeadlineAndDelete(path string, actualPathPrefix string) error {
	deadline, err := getDeadline(path)
	if err != nil || deadline < 0 {
		return errors.New(fmt.Sprintf("Error in getting the deadline: ", err))
	}
	curr := time.Now().Unix()
	if curr > deadline {
		log.Println("[DeleteContentOnExpiry] Validity date expired. Deleting ", actualPathPrefix)
		deletesize := getDirSizeinMB(path)
		hierarchy := strings.Split(strings.Trim(actualPathPrefix, "/"), "/")
		err := fs.RecursiveDeleteFolder(hierarchy)
		if err != nil {
			return err
		}
		log.Println("[DeleteContentOnExpiry] Deleted folder %s and its subtree..", hierarchy)

		// logger.Log("Telemetry", "ContentDeleteInfo", map[string]string{
		// 	"DeleteStatus": "SUCCESS", "FolderPath": actualPathPrefix,
		// 	"Mode": "Expired"})
		//logger.Log("Telemetry", "HubStorage", map[string]string{"AvailableDiskSpace(MB)": getDiskInfo()})
		sm := new(l.MessageSubType)
		sm.AssetInfo.AssetId = hierarchy[len(hierarchy)-1]
		sm.AssetInfo.Size = deletesize
		sm.AssetInfo.RelativeLocation = actualPathPrefix
		logger.Log("AssetDeleteOnDeviceByScheduler", sm)
		sm = new(l.MessageSubType)
		sm.FloatValue = getDiskInfo()
		logger.Log("HubStorage", sm)

	}
	return nil
}
