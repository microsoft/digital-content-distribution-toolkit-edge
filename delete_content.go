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
		// if err != nil {
		// 	logger.Log("Error", "PeriodicDeleteContent", map[string]string{
		// 		"Message" : err.Error()
		// 	})
		// }
	}
}

func traverse(node []byte, prefix string) string {
	children, err := fs.GetChildrenForNode(node)
	if err != nil {
		log.Println(err)
		//return err
	}
	fmt.Println(prefix)
	if len(children) == fs.GetNodeLength() {
		prefix = prefix[:strings.LastIndex(prefix, "/")]
		return prefix
	}
	for i := 0; i < len(children); i += fs.GetNodeLength() {
		if i == 0 {
			continue
		}
		actualName, _ := fs.GetFolderNameForNode(children[i : i+fs.GetNodeLength()])
		fmt.Println("ActualName::::", actualName)
		prefix := prefix + "/" + actualName
		abstractedPath := filepath.Join(fs.GetHomeFolder(), string(children[i:i+fs.GetNodeLength()]))
		fmt.Println("Abstracted path-----", abstractedPath)
		err := checkDeadlineAndDelete(abstractedPath, prefix)
		if err != nil {
			log.Println(err)
			return prefix
		}
		fmt.Println("entire actualPath in next call:::::", prefix)
		prefix = traverse(children[i:i+fs.GetNodeLength()], prefix)
		fmt.Println("New Prefix;;;", prefix)
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
	if time.Now().Unix() > deadline {
		fmt.Println("deleting...... ", actualPathPrefix)
		hierarchy := strings.Split(strings.Trim(actualPathPrefix, "/"), "/")
		err := fs.RecursiveDeleteFolder(hierarchy)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}
