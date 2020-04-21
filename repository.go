package main

import (
	"path"
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

// returns the children of parent in the mediaHouse
func getChildren(mediaHouse string, parent string) []FolderStructureEntry {
	return getChildrenEntries(mediaHouse, parent)
}

// returns the list of metadata files of folder in the mediaHouse
func getMetadataFiles(mediaHouse string, folder string) []string {
	var result []string
	metadataFiles := getMetadataFileEntries(mediaHouse, folder)
	for _, file := range metadataFiles {
		result = append(result, file.Name)
	}
	return result
}

// returns the local file path of the file in folderID in mediaHouse
func getFilePath(mediaHouse string, folderID string, fileName string) string {
	return path.Join(getFolderPath(mediaHouse, folderID), fileName)
}

func getFolderPath(mediaHouse string, folderID string) string {
	return path.Join("static", mediaHouse, folderID)
}

//FileToCheck Struct with file path and it's hashsum (sha256)
type FileToCheck struct {
	path   string
	sha256 string
}
