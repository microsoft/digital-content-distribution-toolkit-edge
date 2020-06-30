package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

//const _interval time.Duration = 120

func checkIntegrity(interval int) {
	for true {
		logger.Log("Telemetry", "Liveness", map[string]string{"STATUS": "ALIVE"})
		time.Sleep(time.Duration(interval) * time.Second)
		fmt.Println("Info", "Checking files integrity from background thread")
		fmt.Println("------------------------------------------------")
		children, _ := fs.GetChildrenForNode(fs.GetHomeNode())
		//fmt.Println(children)
		for i := 0; i < len(children); i += fs.GetNodeLength() {
			if i == 0 {
				continue
			}
			folder_name, _ := fs.GetFolderNameForNode(children[i : i+fs.GetNodeLength()])
			fmt.Println("-----------", folder_name)
			c, t := checkheirarchy(children[i:i+fs.GetNodeLength()], 0, 0)
			c += checkfiles(filepath.Join(fs.GetHomeFolder(), string(children[i:i+fs.GetNodeLength()])))
			t += getTotalFiles(filepath.Join(fs.GetHomeFolder(), string(children[i:i+fs.GetNodeLength()])))
			//fmt.Println(c, t)
			fmt.Println("Telemetry", "[IntegityStats] "+folder_name+" :Total no. of files checked: "+strconv.Itoa(t))
			fmt.Println("Telemetry", "[IntegityStats] "+folder_name+" :No. of files corrupted: "+strconv.Itoa(c))
			logger.Log("Telemetry", "IntegityStats", map[string]string{"FolderName": folder_name, "Total no. of files checked": strconv.Itoa(t)})
			logger.Log("Telemetry", "IntegityStats", map[string]string{"FolderName": folder_name, "No. of files corrupted": strconv.Itoa(c)})
		}
	}
}
func checkheirarchy(node []byte, c, t int) (int, int) {
	//fmt.Println("NODE__", string(node))
	children, err := fs.GetChildrenForNode(node)
	if err != nil {
		log.Println(err)
		return 0, 0
	}
	for i := 0; i < len(children); i += fs.GetNodeLength() {
		if i == 0 {
			continue
		}
		//if dir not empty
		//check files
		fmt.Println(filepath.Join(fs.GetHomeFolder(), string(children[i:i+fs.GetNodeLength()])))
		t += getTotalFiles(filepath.Join(fs.GetHomeFolder(), string(children[i:i+fs.GetNodeLength()])))
		c += checkfiles(filepath.Join(fs.GetHomeFolder(), string(children[i:i+fs.GetNodeLength()])))
		c, t = checkheirarchy(children[i:i+fs.GetNodeLength()], c, t)
	}
	return c, t
}

func checkfiles(folderpath string) int {
	c := 0
	if _, err := os.Stat(filepath.Join(folderpath, "metadatafiles")); !os.IsNotExist(err) {
		hashsummap, f_err := gethashsum(filepath.Join(folderpath, "metadatafiles", "hashsum.txt"))
		if f_err != nil {
			fmt.Println("Could not open hashsum.txt file", f_err)
		}
		//fmt.Println(hashsummap)
		files, _ := ioutil.ReadDir(filepath.Join(folderpath, "metadatafiles"))
		for _, file := range files {
			if file.Name() == "hashsum.txt" {
				continue
			}
			//fmt.Println(filepath.Join(folderpath, "metadatafiles", file.Name()))
			err = matchSHA256(filepath.Join(folderpath, "metadatafiles", file.Name()), hashsummap[file.Name()])
			if err != nil {
				fmt.Println("Telemetry", "[IntegrityStats] "+filepath.Join(folderpath, "metadatafiles", file.Name())+" marked for deletion")
				c++
				continue
			}
		}
	}
	if _, err := os.Stat(filepath.Join(folderpath, "bulkfiles")); !os.IsNotExist(err) {
		hashsummap, f_err := gethashsum(filepath.Join(folderpath, "bulkfiles", "hashsum.txt"))
		if f_err != nil {
			fmt.Println("Could not open hashsum.txt file", f_err)
		}
		//fmt.Println(hashsummap)
		files, _ := ioutil.ReadDir(filepath.Join(folderpath, "bulkfiles"))
		for _, file := range files {
			if file.Name() == "hashsum.txt" {
				continue
			}
			//fmt.Println(filepath.Join(folderpath, "bulkfiles", file.Name()))
			err = matchSHA256(filepath.Join(folderpath, "bulkfiles", file.Name()), hashsummap[file.Name()])
			if err != nil {
				fmt.Println("Telemetry", "[IntegrityStats] "+filepath.Join(folderpath, "bulkfiles", file.Name())+" marked for deletion")
				c++
				continue
			}
		}
	}
	return c
}
func gethashsum(folderpath string) (map[string]string, error) {
	hashmap := make(map[string]string)
	f, err := os.Open(folderpath)
	if err != nil {
		return hashmap, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		linestrA := regexp.MustCompile(`[=>]`).Split(line, -1)
		hashmap[linestrA[0]] = linestrA[2]
	}
	return hashmap, nil
}
func getTotalFiles(folderpath string) int {
	t := 0
	if _, err := os.Stat(filepath.Join(folderpath, "metadatafiles")); !os.IsNotExist(err) {
		files, _ := ioutil.ReadDir(filepath.Join(folderpath, "metadatafiles"))
		t += len(files) - 1
	}
	if _, err := os.Stat(filepath.Join(folderpath, "bulkfiles")); !os.IsNotExist(err) {
		files, _ := ioutil.ReadDir(filepath.Join(folderpath, "bulkfiles"))
		t += len(files) - 1
	}
	return t
}
