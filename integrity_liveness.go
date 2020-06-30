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

func checkIntegrity(interval int) {
	for true {
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
			logger.Log("Telemetry", "IntegrityStats", map[string]string{"ParentName": folder_name, "Files Checked": strconv.Itoa(t), "Files Corrupted": strconv.Itoa(c)})
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
	metadataFolderName := cfg.Section("DEVICE_INFO").Key("METADATA_FOLDER").String()
	bulkFolderName := cfg.Section("DEVICE_INFO").Key("BULKFILE_FOLDER").String()
	if _, err := os.Stat(filepath.Join(folderpath, metadataFolderName)); !os.IsNotExist(err) {
		hashsummap, f_err := gethashsum(filepath.Join(folderpath, metadataFolderName, "hashsum.txt"))
		if f_err != nil {
			fmt.Println("Could not open hashsum.txt file", f_err)
		}
		//fmt.Println(hashsummap)
		files, _ := ioutil.ReadDir(filepath.Join(folderpath, metadataFolderName))
		for _, file := range files {
			if file.Name() == "hashsum.txt" {
				continue
			}
			err = matchSHA256(filepath.Join(folderpath, metadataFolderName, file.Name()), hashsummap[file.Name()])
			if err != nil {
				fmt.Println("Telemetry", "[IntegrityStats] "+filepath.Join(folderpath, "metadatafiles", file.Name())+" is corrupted")
				//TODO: get abstracted path for actual path
				folderName, _ := fs.GetFolderNameForNode([]byte(filepath.Base(folderpath)))
				logger.Log("Telemetry", "IntegrityStats", map[string]string{"FolderName": folderName, "IntegrityStatus": "Corrupted"})
				c++
				continue
			}
		}
	}
	if _, err := os.Stat(filepath.Join(folderpath, bulkFolderName)); !os.IsNotExist(err) {
		hashsummap, f_err := gethashsum(filepath.Join(folderpath, bulkFolderName, "hashsum.txt"))
		if f_err != nil {
			fmt.Println("Could not open hashsum.txt file", f_err)
		}
		//fmt.Println(hashsummap)
		files, _ := ioutil.ReadDir(filepath.Join(folderpath, bulkFolderName))
		for _, file := range files {
			if file.Name() == "hashsum.txt" {
				continue
			}
			//fmt.Println(filepath.Join(folderpath, "bulkfiles", file.Name()))
			err = matchSHA256(filepath.Join(folderpath, bulkFolderName, file.Name()), hashsummap[file.Name()])
			if err != nil {
				fmt.Println("Telemetry", "[IntegrityStats] "+filepath.Join(folderpath, "bulkfiles", file.Name())+" marked for deletion")
				//TODO: get abstracted path for actual path
				folderName, _ := fs.GetFolderNameForNode([]byte(filepath.Base(folderpath)))
				logger.Log("Telemetry", "IntegrityStats", map[string]string{"FolderName": folderName, "IntegrityStatus": "Corrupted"})
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
	metadataFolderName := cfg.Section("DEVICE_INFO").Key("METADATA_FOLDER").String()
	bulkFolderName := cfg.Section("DEVICE_INFO").Key("BULKFILE_FOLDER").String()
	if _, err := os.Stat(filepath.Join(folderpath, metadataFolderName)); !os.IsNotExist(err) {
		files, _ := ioutil.ReadDir(filepath.Join(folderpath, metadataFolderName))
		t += len(files) - 1
	}
	if _, err := os.Stat(filepath.Join(folderpath, bulkFolderName)); !os.IsNotExist(err) {
		files, _ := ioutil.ReadDir(filepath.Join(folderpath, bulkFolderName))
		t += len(files) - 1
	}
	return t
}
func liveness(interval int) {
	for true {
		logger.Log("Telemetry", "Liveness", map[string]string{"STATUS": "ALIVE"})
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
