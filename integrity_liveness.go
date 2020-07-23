package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

func checkIntegrity(interval int) {
	for true {
		time.Sleep(time.Duration(interval) * time.Second)
		log.Println("------Checking for files integrity-------")
		children, _ := fs.GetChildrenForNode(fs.GetHomeNode())
		for i := 0; i < len(children); i += fs.GetNodeLength() {
			if i == 0 {
				continue
			}
			folder_name, _ := fs.GetFolderNameForNode(children[i : i+fs.GetNodeLength()])
			fmt.Println("-----------", folder_name)
			abstractedPath := filepath.Join(fs.GetHomeFolder(), string(children[i:i+fs.GetNodeLength()]))
			fmt.Println(abstractedPath)
			m_c, m_t := checkfiles(abstractedPath, cfg.Section("DEVICE_INFO").Key("METADATA_FOLDER").String())
			b_c, b_t := checkfiles(abstractedPath, cfg.Section("DEVICE_INFO").Key("BULKFILE_FOLDER").String())
			c, t := checkheirarchy(children[i:i+fs.GetNodeLength()], 0, 0)
			c += m_c + b_c
			t += m_t + b_t
			log.Println("[Integrity_Liveness][checkIntegrity]", fmt.Sprintf("%s :Total no. of files checked: %d", folder_name, t))
			log.Println("[Integrity_Liveness][checkIntegrity]", fmt.Sprintf("%s :No. of files corrupted: %d", folder_name, c))
		}
	}
}
func checkheirarchy(node []byte, c, t int) (int, int) {
	children, err := fs.GetChildrenForNode(node)
	if err != nil {
		log.Println("[Integrity_Liveness][checkheirarchy] Error", fmt.Sprintf("%s", err))
		return 0, 0
	}
	for i := 0; i < len(children); i += fs.GetNodeLength() {
		if i == 0 {
			continue
		}
		//if dir not empty
		//check files
		abstractedPath := filepath.Join(fs.GetHomeFolder(), string(children[i:i+fs.GetNodeLength()]))
		log.Println("[Integrity_Liveness][checkheirarchy] Checking abstracted Path: ", abstractedPath)
		m_c, m_t := checkfiles(abstractedPath, cfg.Section("DEVICE_INFO").Key("METADATA_FOLDER").String())
		b_c, b_t := checkfiles(abstractedPath, cfg.Section("DEVICE_INFO").Key("BULKFILE_FOLDER").String())
		c += m_c + b_c
		t += m_t + b_t
		c, t = checkheirarchy(children[i:i+fs.GetNodeLength()], c, t)
	}
	return c, t
}

func checkfiles(folderpath string, subfolderName string) (int, int) {
	c := 0
	t := 0
	if _, err := os.Stat(filepath.Join(folderpath, subfolderName)); !os.IsNotExist(err) {
		hashsummap, fErr := gethashsum(filepath.Join(folderpath, subfolderName, "hashsum.txt"))
		if fErr != nil {
			log.Println(" [Integrity_Liveness][checkfiles] Error: Could not open hashsum.txt file", fErr)
			return 0, 0
		}
		files, _ := ioutil.ReadDir(filepath.Join(folderpath, subfolderName))
		for _, file := range files {
			if file.Name() == "hashsum.txt" {
				continue
			}
			t++
			err = matchSHA256(filepath.Join(folderpath, subfolderName, file.Name()), hashsummap[file.Name()])
			if err != nil {
				log.Println("[Integrity_Liveness][checkfiles]", fmt.Sprintf("Found corrupted file- AbstractedPath: %s", filepath.Join(folderpath, subfolderName, file.Name())))
				//TODO: get abstracted path for actual path
				//folderName, _ := fs.GetFolderNameForNode([]byte(filepath.Base(folderpath)))
				logger.Log("Telemetry", "IntegrityStats", map[string]string{"FileName": filepath.Join(folderpath, subfolderName, file.Name()), "IntegrityStatus": "Corrupted. Should be deleted"})
				c++
				continue
			}
		}
	}
	return c, t
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
		logger.Log("Liveness", "Liveness", map[string]string{"STATUS": "ALIVE"})
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
