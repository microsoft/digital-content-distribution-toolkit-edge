package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
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
		logger.Log("Error", "RouteHandler", map[string]string{"Message": "Error while finding files in " + actualPath + " " + err.Error()})
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
		logger.Log("Error", "RouteHandler", map[string]string{"Message": "Error while finding files in " + actualPath + " " + err.Error()})
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

func getDownloadableURL(osFolderPath string, abstractFilePath string) string {
	abstractFilePath = strings.ReplaceAll(abstractFilePath, "-", "_") // TODO: Replace this hack with fix in DB after demo
	return fmt.Sprintf("http://{HUB_IP}:5000/static/%s%s", (osFolderPath), (abstractFilePath))
}

func getMetadataStruct(filePath string) (*FolderMetadata, error) {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	data := FolderMetadata{}
	err = json.Unmarshal([]byte(file), &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func getAvailableFolders() []AvailableFolder {
	leaves := fs.GetLeavesList()
	var result []AvailableFolder
	fmt.Println("Leaves length: ", len(leaves))
	for _, leaf := range leaves {
		fmt.Println("Leaf is : ", leaf)
		osFsPath, err := fs.GetActualPathForAbstractedPath(leaf)
		fmt.Println("Os file path: ", osFsPath)
		leaf = strings.Replace(leaf, "MSR", "", 1)
		leaf = strings.Replace(leaf, "//", "/", 1)
		if err == nil {
			metadataFilesDirectory := path.Join(osFsPath, "metadatafiles")
			if _, err := os.Stat(metadataFilesDirectory); err == nil {
				fmt.Println("Metadata files directory for: ", metadataFilesDirectory, " exists")
				metadataJSONFilePath := path.Join(metadataFilesDirectory, "bine_metadata.json")
				if _, err := os.Stat(metadataJSONFilePath); err == nil {
					fmt.Println("Metadata bine_json also exists at: ", metadataJSONFilePath)
					folderMetadata, err := getMetadataStruct(metadataJSONFilePath)
					if err == nil {
						folderSize := getFolderSizeParser(osFsPath)
						folderMetadata.Size = strconv.FormatInt(folderSize, 10)

						folderMetadata.Thumbnail = getDownloadableURL(osFsPath, fmt.Sprintf("/metadatafiles/%s", folderMetadata.Thumbnail))
						fmt.Println("Thumbnail URL: ", folderMetadata.Thumbnail)
						folderMetadata.Thumbnail2X = getDownloadableURL(osFsPath, fmt.Sprintf("/metadatafiles/%s", folderMetadata.Thumbnail2X))
						folderMetadata.Language = "English"
						folderMetadata.Path = osFsPath
						folderMetadata.FolderUrl = "http://{HUB_IP}:5000/static/" + osFsPath

						fmt.Println("Folder size is: ", folderSize)
						parts := strings.Split(leaf, "/")

						availableFolder := AvailableFolder{ID: parts[len(parts)-1], Metadata: folderMetadata}
						result = append(result, availableFolder)
					} else {
						logger.Log("Error", "RouteHander", map[string]string{"Function": "GetAvailableFolders", "Message": fmt.Sprintf("metadata json file %s for abstract path %s is invalid with error %s", metadataJSONFilePath, leaf, err.Error())})
					}
				}
			} else {
				logger.Log("Error", "RouteHander", map[string]string{"Function": "GetAvailableFolders", "Message": fmt.Sprintf("metadata directory %s for abstract path %s threw error %s", metadataFilesDirectory, leaf, err.Error())})
			}
		} else {
			fmt.Println("Error: ", err.Error())
		}
	}
	return result
}

//FileToCheck Struct with file path and it's hashsum (sha256)
type FileToCheck struct {
	path   string
	sha256 string
}
