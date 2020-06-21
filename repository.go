package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
)

// returns files in the folder of the mediaHouse
func getFilesToCheck(mediaHouse string, folder string) []FileToCheck {
	var result []FileToCheck
	metadataFiles := getMetadataFileEntries(mediaHouse, folder)
	bulkFiles := getBulkFileEntries(mediaHouse, folder)

	if metadataFiles != nil {
		for _, file := range metadataFiles {
			result = append(result, FileToCheck{path: getFilePath(mediaHouse, folder, file.Name), sha256: file.HashSum})
		}
	}

	if bulkFiles != nil {
		for _, file := range bulkFiles {
			result = append(result, FileToCheck{path: getFilePath(mediaHouse, folder, file.Name), sha256: file.HashSum})
		}
	}
	// get metadata files, get normal files
	return result
}

func getFolderInfo(mediaHouse string, contentPath string) *FolderStructureEntry {
	abstractFilePath := mediaHouse + "/" + contentPath
	fmt.Println("abastract file path: ", abstractFilePath)
	actualPath, err := fs.GetActualPathForAbstractedPath(abstractFilePath)
	if err != nil {
		return nil
	}
	size := getFolderSizeParser(actualPath)
	metadataFiles := getMetdataFileParser(actualPath)
	isLeaf, err := fs.IsLeaf(actualPath)
	if err != nil {
		return nil
	}
	return &FolderStructureEntry{"", !isLeaf, "", size, metadataFiles}
}

func getChildrenParser(actualPath string) ([]interface{}, error) {
	fmt.Println("Actual path: " + actualPath)
	result := []interface{}{}
	isLeaf, err := fs.IsLeaf(actualPath)
	if err != nil {
		fmt.Println("Children parse failed", err)
		return result, nil
	}
	size := getFolderSizeParser(actualPath)
	metadataFiles := getMetdataFileParser(actualPath)
	result = append(result, !isLeaf)
	result = append(result, size)
	result = append(result, metadataFiles)
	fmt.Println("returning: ", result)
	return result, nil
}

func getBineFsPath(mediaHouse string, path string) string {
	bineFsPath := "/" + mediaHouse + "/" + path
	bineFsPath = strings.ReplaceAll(bineFsPath, "//", "/")
	return bineFsPath
}

// returns the children of parent in the mediaHouse
func getChildren(mediaHouse string, parent string) []FolderStructureEntry {
	bineFsPath := getBineFsPath(mediaHouse, parent)
	fmt.Println("BineFS path ", bineFsPath)
	response, err := fs.GetChildrenInfo(bineFsPath, getChildrenParser)
	var result []FolderStructureEntry
	if err == nil {
		for _, child := range response {
			err = nil
			if len(child) > 1 && err == nil {
				result = append(result, FolderStructureEntry{child[0].(string), child[1].(bool), "", child[2].(int64), child[3].([]string)})
			} else {
				fmt.Println("ERR: ", err)
			}
		}
	} else {
		fmt.Println(err)
	}
	return result
}

func getMetdataFileParser(actualPath string) []string {
	var result []string
	parentDirectory := actualPath + "/metadatafiles"
	files, err := ioutil.ReadDir(parentDirectory)
	if err != nil {
		logger.Log("Error", "Error while finding files in "+actualPath+" "+err.Error())
		return result
	}
	for _, file := range files {
		if !file.IsDir() {
			result = append(result, parentDirectory+"/"+file.Name())
			fmt.Println("Appending metadata file: ", file.Name())
		}
	}
	return result
}

// returns the list of metadata files of folder in the mediaHouse
func getMetadataFiles(mediaHouse string, path string) []string {
	var result []string
	return result
}

// returns the local file path of the file in folderID in mediaHouse
func getFilePath(mediaHouse string, folderID string, fileName string) string {
	return path.Join(getFolderPath(mediaHouse, folderID), fileName)
}

func getFolderPath(mediaHouse string, folderID string) string {
	return path.Join("static", mediaHouse, folderID)
	//return path.Join(mediaHouse, folderID)
}

func getFolderSizeParser(actualPath string) int64 {
	files, err := ioutil.ReadDir(actualPath + "/bulkfiles")
	fmt.Println("Actual path for folder size parse: ", actualPath)
	if err != nil {
		logger.Log("Error", "Error while finding files in "+actualPath+" "+err.Error())
		return 0
	}
	var size int64 = 0
	for _, file := range files {
		if !file.IsDir() {
			size += file.Size()
			fmt.Println("Adding size of ", file.Name())
		}
	}
	return size
}

func getFolderSize(mediaHouse string, path string) int64 {
	return 0
}

//FileToCheck Struct with file path and it's hashsum (sha256)
type FileToCheck struct {
	path   string
	sha256 string
}
