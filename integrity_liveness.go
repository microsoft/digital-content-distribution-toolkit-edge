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
			nodeToCheck := children[i : i+fs.GetNodeLength()]
			foldername, _ := fs.GetFolderNameForNode(nodeToCheck)
			log.Println("----checking Parent Folder: ------", foldername)
			_c, _t := folderToCheck(string(nodeToCheck))
			c, t := checkheirarchy(children[i:i+fs.GetNodeLength()], 0, 0)
			c += _c
			t += _t
			log.Println("[Integrity_Liveness][checkIntegrity]", fmt.Sprintf("Mediahouse - %s :Total no. of files checked: %d", foldername, t))
			log.Println("[Integrity_Liveness][checkIntegrity]", fmt.Sprintf("Mediahouse - %s :No. of files corrupted: %d", foldername, c))
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

		nodeToCheck := (children[i : i+fs.GetNodeLength()])
		_c, _t := folderToCheck(string(nodeToCheck))
		c += _c
		t += _t
		c, t = checkheirarchy(nodeToCheck, c, t)
	}
	return c, t
}
func folderToCheck(folderName string) (int, int) {
	c := 0
	t := 0
	actualPath := filepath.Join(fs.GetHomeFolder(), folderName)
	abstractedFolderName, _ := fs.GetFolderNameForNode([]byte(folderName))
	//check if path exist in the zzzz folder
	if _, err := os.Stat(actualPath); os.IsNotExist(err) {
		log.Println("[Integrity_Liveness][checkheirarchy] ", fmt.Sprintf("Directory MISSING in the home folder for folderName : (%s) and actual path: (%s)", abstractedFolderName, actualPath))
		logger.Log("Error", "Integrity_Liveness", map[string]string{"Message": "Directory MISSING in the HOME folder", "FolderName": abstractedFolderName, "Actual Path": actualPath})
		return c, t
	}
	log.Println("[Integrity_Liveness][checkheirarchy] ", fmt.Sprintf("Checking folder Path (%s) for Heirarchy (%s): ", actualPath, abstractedFolderName))
	if fs.IsSatelliteFolder(folderName) {
		satellitePath, _ := fs.GetSatelliteFolderPath(folderName)
		//check if the satellite path exist
		if _, err := os.Stat(satellitePath); os.IsNotExist(err) {
			log.Println("[Integrity_Liveness][checkheirarchy] ", fmt.Sprintf("Directory MISSING in the mstore folder for folderName : (%s) and actual path: (%s)", abstractedFolderName, satellitePath))
			logger.Log("Error", "Integrity_Liveness", map[string]string{"Message": "Directory MISSING in the MSTORE folder", "FolderName": abstractedFolderName, "Actual Path": satellitePath})
			return c, t
		}
		log.Println("[Integrity_Liveness][checkheirarchy] Found to be a Satellite content...Satellite folder path: ", satellitePath)
		//gethashums for the folder
		hashsummap, fErr := gethashsum(filepath.Join(actualPath, "hashsum.txt"))
		if fErr != nil {
			log.Println(" [Integrity_Liveness][checkfiles][gethashsum] Error: ", fmt.Sprintf("%s", fErr))
			return 0, 0
		}
		_c, _t := checkfiles(abstractedFolderName, satellitePath, hashsummap)
		c += _c
		t += _t
	} else {
		// get hashsums for every subfolder(metadatafiles and bulkfiles) and check
		subfolders, _ := ioutil.ReadDir(actualPath)
		for _, subfolder := range subfolders {
			if subfolder.IsDir() {
				log.Println(fmt.Sprintf("In dir: %s", filepath.Join(actualPath, subfolder.Name())))
				hashsummap, fErr := gethashsum(filepath.Join(actualPath, subfolder.Name(), "hashsum.txt"))
				if fErr != nil {
					log.Println(" [Integrity_Liveness][checkfiles][gethashsum] Error: ", fmt.Sprintf("%s", fErr))
					return 0, 0
				}
				_c, _t := checkfiles(abstractedFolderName, filepath.Join(actualPath, subfolder.Name()), hashsummap)
				c += _c
				t += _t
			}
		}

	}
	return c, t
}
func checkfiles(heirarchy string, folderpath string, hashsummap map[string]string) (int, int) {
	c := 0
	t := 0
	files, _ := ioutil.ReadDir(folderpath)
	for _, file := range files {
		if _, exist := hashsummap[file.Name()]; !exist {
			log.Println(fmt.Sprintf("Skipping file..... (%s).... Either the hashsum of the file does not exist in this folder or file of not a valid type(metdata/bulkfile) ", file.Name()))
			continue
		}
		t++
		err := matchSHA256(filepath.Join(folderpath, file.Name()), hashsummap[file.Name()])
		if err != nil {
			log.Println("[Integrity_Liveness][checkfiles]", fmt.Sprintf("Filename (%s), Status: CORRUPTED", filepath.Join(folderpath, file.Name())))
			log.Println("[Integrity_Liveness][checkfiles]", fmt.Sprintf("File (%s) corrupted for folder (%s) at directory location (%s) : %s", file.Name(), heirarchy, folderpath))
			//TODO: get abstracted path for actual path
			logger.Log("Telemetry", "IntegrityStats", map[string]string{"FileName": filepath.Join(folderpath, file.Name()), "Folder": heirarchy, "Directory Location": folderpath, "IntegrityStatus": "Corrupted. Hashum did not match"})
			c++
			continue
		}
		log.Println("[Integrity_Liveness][checkfiles]", fmt.Sprintf("Filename (%s), Status: MATCHED", filepath.Join(folderpath, file.Name())))
	}
	// check if all the files listed in hashsum.txt exist in the folder or not
	if hashsummap != nil {
		for k := range hashsummap {
			if _, err := os.Stat(filepath.Join(folderpath, k)); os.IsNotExist(err) {
				log.Println("[Integrity_Liveness][checkfiles] ", fmt.Sprintf("Filename (%s) MISSING in the folder for folder path : (%s) whereas the entry exist in hashsum.txt", k, folderpath))
				logger.Log("Error", "Integrity_Liveness", map[string]string{"FileName": k, "Folder": folderpath, "Message": "File MISSING in the folder whereas entry exist in the hashsum.txt file"})
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
	metadataFolderName := "metadatafiles"
	bulkFolderName := "bulkfiles"
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
