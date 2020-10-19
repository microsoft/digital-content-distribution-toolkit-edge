package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const source string = "mstore"

type VodObj struct {
	Source  string `json:"source"`
	Content struct {
		Data struct {
			File []struct {
				Filename string `json:"filename"`
				Filesize int    `json:"filesize"`
			} `json:"file"`
		} `json:"data"`
		UserDefined struct {
			MediaHouse    string `json:"mediaHouse"`
			MediaId       string `json:"mediaId"`
			MetadataFiles struct {
				File []struct {
					Filename string `json:"filename"`
					Checksum string `json:"checksum"`
					FolderId string `json:"folderId"`
				} `json:"file"`
			} `json:"metadataFiles"`
			PushId      int    `json:"pushId"`
			AncestorIds string `json:"ancestorIds"`
			BulkFiles   struct {
				File []struct {
					Filename string `json:"filename"`
					Checksum string `json:"checksum"`
					FolderId string `json:"folderId"`
				} `json:"file"`
			} `json:"bulkFiles"`
		} `json:"userDefined"`
		Videos struct {
			Movie struct {
				Duration string `json:"duration"`
				File     struct {
					Filename string `json:"filename"`
					Filesize int    `json:"filesize"`
				} `json:"file"`
			} `json:"movie"`
		} `json:"videos"`
		VODInfo struct {
			MovieID         string    `json:"movieID"`
			ValidityEndDate time.Time `json:"validityEndDate"`
		} `json:"VODInfo"`
		Pictures struct {
			Cover struct {
				File struct {
					Filename string `json:"filename"`
					Filesize int    `json:"filesize"`
				} `json:"file"`
				Resolution string `json:"resolution"`
			} `json:"cover"`
			Thumbnail struct {
				File struct {
					Filename string `json:"filename"`
					Filesize int    `json:"filesize"`
				} `json:"file"`
				Resolution string `json:"resolution"`
			} `json:"thumbnail"`
		} `json:"pictures"`
		PushInfo struct {
			CID int `json:"CID"`
		} `json:"pushInfo"`
	} `json:"content"`
}

func pollNoovo(interval int) {
	for true {
		log.Println("==================Polling NOOVO API for the content==============")
		if err := callNoovoAPI(); err != nil {
			log.Println("[Satdata_noovo][pollNoovo] Error", fmt.Sprintf("%s", err))
			logger.Log("Error", "SatdataNoovo", map[string]string{"Message": err.Error()})
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func callNoovoAPI() error {
	res, err := http.Get("http://localhost:40000/vod/list")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	jsonRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var vods []VodObj
	jsonErr := json.Unmarshal(jsonRes, &vods)
	if jsonErr != nil {
		return jsonErr
	}
	log.Println("[Satdata_noovo][callNoovoAPI] NO. OF CONTENTS ON THE SAT: ", len(vods))
	for _, vod := range vods {
		if matched, _ := regexp.MatchString(`^BINE.`, vod.Content.VODInfo.MovieID); matched {
			var _heirarchy string
			if vod.Content.UserDefined.AncestorIds != "" {
				_heirarchy = vod.Content.UserDefined.MediaHouse + vod.Content.UserDefined.AncestorIds + "/" + vod.Content.UserDefined.MediaId
			} else {
				_heirarchy = vod.Content.UserDefined.MediaHouse + "/" + vod.Content.UserDefined.MediaId
			}
			path, _ := fs.GetActualPathForAbstractedPath(_heirarchy)
			if path != "" {
				log.Println(_heirarchy + " already exist.")
				continue
			}
			if err := downloadContent(vod, _heirarchy); err != nil {
				log.Println("[Satdata_noovo][callNoovoAPI] Error", fmt.Sprintf("%s", err))
				logger.Log("Error", "SatdataNoovo", map[string]string{"Message": err.Error()})
				logger.Log("Telemetry", "ContentSyncInfo", map[string]string{"DownloadStatus": "FAIL", "FolderPath": _heirarchy, "Channel": "SES"})
				continue
			}
			path, _ = fs.GetActualPathForAbstractedPath(_heirarchy)
			logger.Log("Telemetry", "ContentSyncInfo", map[string]string{"DownloadStatus": "SUCCESS", "FolderPath": _heirarchy, "AssetSize(MB)": fmt.Sprintf("%.2f", getDirSizeinMB(path)), "Channel": "SES"})
			logger.Log("Telemetry", "HubStorage", map[string]string{"AvailableDiskSpace(MB)": getDiskInfo()})
			log.Println("[Satdata_noovo][callNoovoAPI] Info ", fmt.Sprintf("Downloaded on the HUB: %s", _heirarchy))
		}
	}
	return nil
}

func downloadContent(vod VodObj, _heirarchy string) error {
	log.Println("[Satdata_noovo][callNoovoAPI] Info ", fmt.Sprintf("Downloading : %s", _heirarchy))
	pushId := strconv.Itoa(vod.Content.UserDefined.PushId)
	deadline := vod.Content.VODInfo.ValidityEndDate

	// filesUrlMap of the files with the url
	filesURLMap := make(map[string]string)
	filesURLMap[filepath.Base(vod.Content.Pictures.Thumbnail.File.Filename)] = vod.Content.Pictures.Thumbnail.File.Filename
	filesURLMap[filepath.Base(vod.Content.Pictures.Cover.File.Filename)] = vod.Content.Pictures.Cover.File.Filename
	filesURLMap[filepath.Base(vod.Content.Videos.Movie.File.Filename)] = vod.Content.Videos.Movie.File.Filename
	for _, datafile := range vod.Content.Data.File {
		filesURLMap[filepath.Base(datafile.Filename)] = datafile.Filename
	}
	folderMetadataFilesMap := make(map[string][]FileInfo)
	for _, metadatafileEntry := range vod.Content.UserDefined.MetadataFiles.File {
		folderMetadataFilesMap[metadatafileEntry.FolderId] = append(folderMetadataFilesMap[metadatafileEntry.FolderId], FileInfo{metadatafileEntry.Filename, metadatafileEntry.Checksum})
	}
	folderBulkFilesMap := make(map[string][]FileInfo)
	for _, bulkfileEntry := range vod.Content.UserDefined.BulkFiles.File {
		folderBulkFilesMap[bulkfileEntry.FolderId] = append(folderBulkFilesMap[bulkfileEntry.FolderId], FileInfo{bulkfileEntry.Filename, bulkfileEntry.Checksum})
	}
	hierarchy := strings.Split(strings.Trim(_heirarchy, "/"), "/")
	log.Println(hierarchy)
	subpath := ""
	for _, folder := range hierarchy {
		subpath = subpath + folder + "/"
		log.Println("Printing buckets")
		fs.PrintBuckets()
		log.Println("Printing file sys")
		fs.PrintFileSystem()
		log.Println("====================")
		log.Println(subpath)
		metafilesLen, bulkfilesLen := len(folderMetadataFilesMap[folder]), len(folderBulkFilesMap[folder])
		fileInfos := make([][]string, metafilesLen+bulkfilesLen+1)
		for i, x := range folderMetadataFilesMap[folder] {
			fileInfos[i] = make([]string, 5)
			fileInfos[i][0] = x.Name
			fileInfos[i][1] = filesURLMap[pushId+"_"+folder+"_"+x.Name]
			fileInfos[i][2] = x.Hashsum
			fileInfos[i][3] = "metadata"
			fileInfos[i][4] = strconv.FormatInt(deadline.Unix(), 10)
		}
		for i, x := range folderBulkFilesMap[folder] {
			fileInfos[i] = make([]string, 5)
			fileInfos[metafilesLen+i][0] = x.Name
			fileInfos[metafilesLen+i][1] = filesURLMap[pushId+"_"+folder+"_"+x.Name]
			fileInfos[metafilesLen+i][2] = x.Hashsum
			fileInfos[metafilesLen+i][3] = "bulkfile"
			fileInfos[metafilesLen+i][4] = strconv.FormatInt(deadline.Unix(), 10)
		}
		fileInfos[metafilesLen+bulkfilesLen] = make([]string, 5)
		fileInfos[metafilesLen+bulkfilesLen][4] = strconv.FormatInt(deadline.Unix(), 10)
		subpathA := strings.Split(strings.Trim(subpath, "/"), "/")
		err := fs.CreateDownloadNewFolder(subpathA, DownloadFiles, fileInfos, false, "")
		if err != nil {
			log.Println(err)
			// if eval, ok := err.(*fs.FolderExistError); ok {
			// 	continue
			// }
			if err.Error() == "[Filesystem][CreateFolder]A folder with the same name at the requested level already exists" {
				continue
			}
			log.Println("[Satdata_noovo][DownloadContent] Error", fmt.Sprintf("%s", err))
			return err
		}
		log.Println("")
		fs.PrintBuckets()
		fs.PrintFileSystem()
		log.Println("")
	}

	return nil

}

func dummyTest() error {
	fmt.Println("TEST")
	file, err := os.Open("test/sampleResp.json")
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer file.Close()
	bytevalue, _ := ioutil.ReadAll(file)
	var vods []VodObj
	if err = json.Unmarshal(bytevalue, &vods); err != nil {
		fmt.Println(err)
		return err
	}
	for _, vod := range vods {
		if vod.Source == source {
			if matched, _ := regexp.MatchString(`^BINE.`, vod.Content.VODInfo.MovieID); matched {
				var _heirarchy string
				if vod.Content.UserDefined.AncestorIds != "" {
					_heirarchy = vod.Content.UserDefined.MediaHouse + "/" + vod.Content.UserDefined.AncestorIds + "/" + vod.Content.UserDefined.MediaId
				} else {
					_heirarchy = vod.Content.UserDefined.MediaHouse + "/" + vod.Content.UserDefined.MediaId
				}
				err := downloadContent(vod, _heirarchy)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
	return nil

}
