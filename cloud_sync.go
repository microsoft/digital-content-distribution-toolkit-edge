package main

// TODO: Implement cloud syncing service
// TODO: Implement LRU cache
// TODO: Implement Logging class
// TODO: Implement telemetry posting
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

const downloadRetries int = 5
const exponentialBackOffInitTime time.Duration = 15

func downloadFolder(downloadCommandBytes []byte) error {
	var downloadCommands []DownloadCommand
	err := json.Unmarshal(downloadCommandBytes, &downloadCommands)
	if err != nil {
		return errors.New("Could not unmarshall file contents in downloadCommand")
	}
	for _, downloadCommand := range downloadCommands {
		metadataFiles := getMetadataFiles(downloadCommand.MediaHouse, downloadCommand.ID)
		if len(metadataFiles) > 0 {
			fmt.Println("Folder: ", downloadCommand.ID, " already exists")
			continue
		}
		println("Downloading folder: " + downloadCommand.ID)
		println("Media House is: " + downloadCommand.MediaHouse)
		println("Ignoring channel now, need to figure out how to create sockets on a particular network")
		println("Verifying hierarchy of the current download request")
		currentNode := downloadCommand.Hierarchy.Child // should start from child because root always exists
		parentNode := downloadCommand.Hierarchy
		for currentNode != nil {
			childNode := currentNode.Child
			children := getChildren(downloadCommand.MediaHouse, currentNode.Level)
			if childNode != nil {
				foundChild := false
				for _, child := range children {
					if child.ID == childNode.Level {
						foundChild = true
						break
					}
				}
				if !foundChild {
					return errors.New("Could not find hierarchy at currentNode: " + currentNode.Level + " and child: ")
				}
			}
			parentNode = currentNode
			currentNode = childNode
		}
		folderPath := getFolderPath(downloadCommand.MediaHouse, downloadCommand.ID)
		println("folder path: " + folderPath)
		err = os.MkdirAll(folderPath, 0700)
		if err != nil {
			println("Could not create directory for ID: " + downloadCommand.ID)
		}

		println("downloading cloud metadata files")
		fileEntries := downloadCloudFiles(folderPath, downloadCommand.MetadataFiles)
		if fileEntries == nil {
			return errors.New("Could not download cloud files for: " + downloadCommand.ID)
		}
		err = addMetadataFiles(downloadCommand.MediaHouse, downloadCommand.ID, fileEntries)
		if err != nil {
			return err
		}

		println("downloading cloud bulk files")
		fileEntries = downloadCloudFiles(folderPath, downloadCommand.BulkFiles)
		if fileEntries == nil {
			return errors.New("Could not download cloud bulk files for: " + downloadCommand.ID)
		}
		err = addBulkFiles(downloadCommand.MediaHouse, downloadCommand.ID, fileEntries)
		if err != nil {
			return err
		}

		print("adding folder to db ")
		if downloadCommand.HasChildren {
			println(downloadCommand.ID + " has children")
		}
		err = addFolder(downloadCommand.MediaHouse, parentNode.Level, FolderStructureEntry{downloadCommand.ID, downloadCommand.HasChildren, ""})
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadExponential(filePath string, url string, trueSha256 string) error {
	initialDelay := exponentialBackOffInitTime
	for retry := 1; retry <= downloadRetries; retry++ {
		err := downloadFile(filePath, url)
		if err != nil {
			println("Could not download file: " + err.Error())
			time.Sleep(initialDelay * time.Second)
			initialDelay *= 2
		} else {
			// verify hashsum
			computedHashsum := computeSHA256(filePath)
			if computedHashsum == trueSha256 {
				return nil
			}
			return errors.New("Hashsum did not match for file: " + url)
		}
	}
	return errors.New("Could not download: " + url)
}

//TODO: Implement download resume? use a golang library maybe
func downloadFile(filePath string, url string) error {
	println("Downloading file: " + url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fileOutputStream, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer fileOutputStream.Close()
	fileLengthString := resp.Header.Get("Content-Length")
	fileLength, err := strconv.Atoi(fileLengthString)
	progressWriter := &ProgressWriter{}
	progressWriter.Total = int64(fileLength / 1024 / 1024)
	_, err = io.Copy(fileOutputStream, io.TeeReader(resp.Body, progressWriter))
	return err
}

func downloadCloudFiles(folderPath string, cloudFileEntries []CloudFileEntry) []FileEntry {
	fileEntries := []FileEntry{}
	for _, cloudFileEntry := range cloudFileEntries {
		err := downloadExponential(path.Join(folderPath, cloudFileEntry.Name), cloudFileEntry.Cdn, cloudFileEntry.Hashsum)
		if err != nil {
			println("Could nto download file: " + cloudFileEntry.Cdn)
			return nil
		}
		fileEntries = append(fileEntries, FileEntry{cloudFileEntry.Name, cloudFileEntry.Hashsum})
	}
	return fileEntries
}

func deleteFolder(deleteCommandBytes []byte) error {
	var children []string
	deleteCommand := new(DeleteCommand)
	err := json.Unmarshal(deleteCommandBytes, deleteCommand)
	if err != nil {
		return err
	}
	println("Deleting folder: " + deleteCommand.ID)
	children = getChildrenToRemove(deleteCommand.MediaHouse, deleteCommand.ID, children)

	err = eraseFolder(deleteCommand.ID, deleteCommand.MediaHouse)
	if err != nil {
		return err
	}
	// delete the directory of id in static/mediaHouse/id
	err = os.RemoveAll(getFolderPath(deleteCommand.MediaHouse, deleteCommand.ID))
	if err != nil {
		return err
	}

	// delete the directories in static/mediaHouse/{id} where id is each entry in children
	for _, id := range children {
		println("Deleting: " + id)
		err = os.RemoveAll(getFolderPath(deleteCommand.MediaHouse, id))
		if err != nil {
			return err
		}
	}
	return nil
}

func getChildrenToRemove(mediaHouse string, parent string, result []string) []string {
	children := getChildren(mediaHouse, parent)
	for _, child := range children {
		result = append(result, child.ID)
		result = getChildrenToRemove(mediaHouse, child.ID, result)
	}
	return result
}

func testCloudSyncServiceDownload() {
	files := []string{"test/download-ars-season.json", "test/download-ss-season.json", "test/download-ss-ep1.json", "test/download-ars-ep1.json"}

	for _, filePath := range files {
		bytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			println("Could not read json:" + err.Error())
		} else {
			err = downloadFolder(bytes)
			if err != nil {
				println(err.Error())
			}
		}
	}
}

func testCloudSyncServiceDelete() {
	bytes, err := ioutil.ReadFile("test/delete.json")
	if err != nil {
		println("Could not read json: " + err.Error())
	} else {
		deleteFolder(bytes)
	}
}

// Structures related to cloud sync messages

//DownloadCommand represents cloud download Command Json
type DownloadCommand struct {
	ID            string           `json:"id"`
	MediaHouse    string           `json:"mediaHouse"`
	Channel       []string         `json:"channel"`
	MetadataFiles []CloudFileEntry `json:"metadataFiles"`
	BulkFiles     []CloudFileEntry `json:"bulkFiles"`
	Deadline      int              `json:"deadline"`
	Hierarchy     *HierarchyNode   `json:"hierarchy"`
	HasChildren   bool             `json:"hasChildren"`
}

//CloudFileEntry represents cloud download Command Json
type CloudFileEntry struct {
	Name    string `json:"name"`
	Cdn     string `json:"cdn"`
	Hashsum string `json:"hashsum"`
}

//HierarchyNode represents cloud download Command Json
type HierarchyNode struct {
	Level string         `json:"level"`
	Child *HierarchyNode `json:"child"`
}

//DeleteCommand represents delete command Json
type DeleteCommand struct {
	ID         string `json:"id"`
	MediaHouse string `json:"mediaHouse"`
}
