package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type VodList struct {
	Status       string `json:"status"`
	ListContents []struct {
		ContentID string `json:"ContentId"`
	} `json:"listContents"`
}

type VodInfo struct {
	Status   string `json:"status"`
	Metadata struct {
		UserDefined struct {
			MediaId     string `json:"mediaId"`
			MediaHouse  string `json:"mediaHouse"`
			AncestorIds struct {
				File []string `json:"file"`
			} `json:"ancestorIds"`
			MetadataFiles struct {
				File []struct {
					Filename string `json:"filename"`
					Filesize int    `json:"filesize"`
					Checksum string `json:"checksum"`
					FolderId string `json:"folderId"`
				} `json:"file"`
			} `json:"metadataFiles"`
			BulkFiles struct {
				File struct {
					Filename string `json:"filename"`
					Filesize int    `json:"filesize"`
					Checksum string `json:"checksum"`
					FolderId string `json:"folderId"`
				} `json:"file"`
			} `json:"bulkFiles"`
			PushId int `json:"pushId"`
		} `json:"userDefined"`
		MovieId         string `json:"movieID"`
		CID             string `json:"CID"`
		VideoFilename   string `json:"video filename"`
		TrailerFilename string `json:"trailer filename"`
		CoverFilename   string `json:"cover filename"`
		URLForDataFiles string `json:"urlForDataFiles"`
		DataFiles       struct {
			File []struct {
				Filename string `json:"filename"`
				Filesize int    `json:"filesize"`
			} `json:"file"`
		} `json:"dataFiles"`
		ThumbnailFilename string `json:"thumbnail filename"`
	} `json:"metadata"`
}

func checkForVODViaMstore() error {
	res, err := http.Get("http://localhost:8134/listcontents")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	jsonRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	//fmt.Println("RESPONSE::::::::::", string(jsonRes))
	var vodlist VodList
	jsonErr := json.Unmarshal(jsonRes, &vodlist)
	if jsonErr != nil {
		return jsonErr
	}
	fmt.Println(":::::::::::::::::NO. OF CONTENTS:::::::::::", len(vodlist.ListContents))
	for _, id := range vodlist.ListContents {
		fmt.Println("======= Processing for CID=======", id.ContentID)
		err := getMetadataAPI(id.ContentID)
		if err != nil {
			log.Println(err)
			logger.Log("Error", fmt.Sprintf("%s", err))
			//return err
			continue
		}
	}
	return nil
}

func getMetadataAPI(contentId string) error {
	query := "http://localhost:8134/getmetadata/" + contentId
	res, err := http.Get(query)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	jsonRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var vod VodInfo
	jsonErr := json.Unmarshal(jsonRes, &vod)
	if jsonErr != nil {
		return jsonErr
	}
	// check if the status is 0
	if vod.Status == "15" {
		return errors.New("No metadata available")
	}
	if err = getMstoreFiles(vod); err != nil {
		return err
	}
	return nil
}

func getMstoreFiles(vod VodInfo) error {
	pushId := strconv.Itoa(vod.Metadata.UserDefined.PushId)
	_heirarchy := vod.Metadata.UserDefined.MediaHouse + "/"
	for i, x := range vod.Metadata.UserDefined.AncestorIds.File {
		if i == 0 {
			continue
		}
		_heirarchy = _heirarchy + x + "/"
	}
	_heirarchy = _heirarchy + vod.Metadata.UserDefined.MediaId + "/"
	path, _ := fs.GetActualPathForAbstractedPath(_heirarchy)
	if path != "" {
		log.Println(_heirarchy + " already exist.")
		return errors.New(_heirarchy + " already exist.")
	}
	filepathMap := make(map[string]string)
	filepathMap[filepath.Base(vod.Metadata.ThumbnailFilename)] = vod.Metadata.ThumbnailFilename
	filepathMap[filepath.Base(vod.Metadata.CoverFilename)] = vod.Metadata.CoverFilename
	filepathMap[filepath.Base(vod.Metadata.VideoFilename)] = vod.Metadata.VideoFilename
	// add data section filepaths
	for _, datafile := range vod.Metadata.DataFiles.File {
		filepathMap[filepath.Base(datafile.Filename)] = vod.Metadata.URLForDataFiles + datafile.Filename
	}
	folderMetadataFilesMap := make(map[string][]FileInfo)
	for _, metadatafileEntry := range vod.Metadata.UserDefined.MetadataFiles.File {
		folderMetadataFilesMap[metadatafileEntry.FolderId] = append(folderMetadataFilesMap[metadatafileEntry.FolderId], FileInfo{metadatafileEntry.Filename, metadatafileEntry.Checksum})
		//fmt.Println("MAP APPENDED:::::", folderMetadataFilesMap[metadatafileEntry.FolderId])
	}
	fmt.Println(folderMetadataFilesMap)
	folderBulkFilesMap := make(map[string][]FileInfo)
	folderBulkFilesMap[vod.Metadata.UserDefined.BulkFiles.File.FolderId] = []FileInfo{FileInfo{vod.Metadata.UserDefined.BulkFiles.File.Filename, vod.Metadata.UserDefined.BulkFiles.File.Checksum}}

	fmt.Println("\nBulkfiles Map", folderBulkFilesMap)
	hierarchy := strings.Split(strings.Trim(_heirarchy, "/"), "/")
	log.Println(hierarchy)
	subpath := ""
	for _, folder := range hierarchy {
		subpath = subpath + folder + "/"
		fmt.Println("Printing buckets")
		fs.PrintBuckets()
		fmt.Println("Printing file sys")
		fs.PrintFileSystem()
		fmt.Println("====================")
		fmt.Println(subpath)
		metafilesLen, bulkfilesLen := len(folderMetadataFilesMap[folder]), len(folderBulkFilesMap[folder])
		fileInfos := make([][]string, metafilesLen+bulkfilesLen)
		for i, x := range folderMetadataFilesMap[folder] {
			fileInfos[i][0] = x.Name
			fileInfos[i][1] = filepathMap[pushId+"_"+folder+"_"+x.Name]
			fileInfos[i][2] = x.Hashsum
			fileInfos[i][3] = "metadata"
		}
		for i, x := range folderBulkFilesMap[folder] {
			fileInfos[metafilesLen+i][0] = x.Name
			fileInfos[metafilesLen+i][1] = filepathMap[pushId+"_"+folder+"_"+x.Name]
			fileInfos[metafilesLen+i][2] = x.Hashsum
			fileInfos[metafilesLen+i][3] = "bulkfile"
		}
		subpathA := strings.Split(strings.Trim(subpath, "/"), "/")
		err := fs.CreateDownloadNewFolder(subpathA, copyFiles, fileInfos)
		if err != nil {
			log.Println(err)
			// if eval, ok := err.(*fs.FolderExistError); ok {
			// 	continue
			// }
			if err.Error() == "[Filesystem][CreateFolder]A folder with the same name at the requested level already exists" {
				continue
			}
			logger.Log("Error", fmt.Sprintf("%s", err))
			return err
		}
		log.Println("")
		fs.PrintBuckets()
		fs.PrintFileSystem()
		log.Println("")
	}
	path, _ = fs.GetActualPathForAbstractedPath(_heirarchy)
	logger.Log("Telemetry", "[DownloadSize] "+_heirarchy+" of size :"+strconv.FormatInt(getDirSizeinMB(path), 10)+"downloaded on the Hub")
	logger.Log("Telemetry", "[ContentSyncChannel] "+_heirarchy+" synced via SES channel: SUCCESS")
	logger.Log("Telemetry", "[Storage] "+"Disk space available on the Hub: "+getDiskInfo())
	return nil
}

func copyFiles(filePath string, fileInfos [][]string) error {
	for _, x := range fileInfos {
		var downloadpath string
		switch x[3] {
		case "metadata":
			downloadpath = filepath.Join(filePath, "metadatafiles", x[0])
		case "bulkfile":
			downloadpath = filepath.Join(filePath, "bulkfiles", x[0])
		default:
			log.Println("Invalid File type: ", x[0])
			continue
		}
		if err := os.MkdirAll(filepath.Dir(downloadpath), 0700); err != nil {
			logger.Log("Error", fmt.Sprintf("%s", err))
			return err
		}
		err := copySingleFile(downloadpath, x[1])
		if err != nil {
			logger.Log("Error", fmt.Sprintf("%s", err))
			return err
		}
		calculatedHash, err := calculateSHA256(downloadpath)
		if err != nil || calculatedHash != x[2] {
			logger.Log("Error", fmt.Sprintf("Hashsum did not match: %s", err))
			return err
		}
		//store it in a file
		if err := storeHashsum(downloadpath, calculatedHash); err != nil {
			logger.Log("Error", fmt.Sprintf("Could not store Hashsum in the text file: %s", err))
			return err
		}
	}
	return nil
}

func copySingleFile(dest, source string) error {
	//TODO: handle if the source does not exist
	sfile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sfile.Close()

	dfile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dfile.Close()
	written, err := io.Copy(dfile, sfile)
	fmt.Println("======== Written bytes======== ", written)
	return nil
}

func testMstore() error {
	fmt.Println("TEST")
	file, err := os.Open("resp3.json")
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer file.Close()
	bytevalue, _ := ioutil.ReadAll(file)
	var vod VodInfo
	if err = json.Unmarshal(bytevalue, &vod); err != nil {
		fmt.Println(err)
		return err
	}
	if vod.Status == "15" {
		return errors.New("No metadata available")
	}
	if err = getMstoreFiles(vod); err != nil {
		return err
	}
	return nil

}
