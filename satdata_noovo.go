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

//const interval time.Duration = 20
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
			PushId      int `json:"pushId"`
			AncestorIds struct {
				File []string `json:"file"`
			} `json:"ancestorIds"`
			BulkFiles struct {
				File struct {
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
			MovieID string `json:"movieID"`
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

func pollNoovo(interval time.Duration) {
	for true {
		fmt.Println("==================Polling NOOVO API for the content==============")
		//logger.Log("Info", "Polling NOOVO API for the new content on the SAT")
		if err := callNoovoAPI(); err != nil {
			log.Println(err)
			//logger.Log("Error", err.Error())
		}
		time.Sleep(interval * time.Minute)
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
	//fmt.Println("RESPONSE::::::::::", string(jsonRes))
	var vods []VodObj
	jsonErr := json.Unmarshal(jsonRes, &vods)
	if jsonErr != nil {
		return jsonErr
	}
	fmt.Println(":::::::::::::::::NO. OF CONTENTS:::::::::::", len(vods))
	for _, vod := range vods {
		if matched, _ := regexp.MatchString(`^BINE.`, vod.Content.VODInfo.MovieID); matched {
			_heirarchy := vod.Content.UserDefined.MediaHouse + "/"
			for i, x := range vod.Content.UserDefined.AncestorIds.File {
				if i == 0 {
					continue
				}
				_heirarchy = _heirarchy + x + "/"
			}
			_heirarchy = _heirarchy + vod.Content.UserDefined.MediaId + "/"
			path, _ := fs.GetActualPathForAbstractedPath(_heirarchy)
			if path != "" {
				log.Println(_heirarchy + " already exist.")
				continue
			}
			if err := downloadContent(vod, _heirarchy); err != nil {
				log.Println(err)
				//logger.Log("Error", fmt.Sprintf("%s", err))
				//logger.Log("Telemetry", "[ContentSyncChannel] "+_heirarchy+" synced via SES channel: FAILED")
				continue
			}
			path, _ = fs.GetActualPathForAbstractedPath(_heirarchy)
			// logger.Log("Telemetry", "[DownloadSize] "+_heirarchy+" of size :"+strconv.FormatInt(getDirSizeinMB(path), 10)+"downloaded on the Hub")
			// logger.Log("Telemetry", "[ContentSyncChannel] "+_heirarchy+" synced via SES channel: SUCCESS")
			// logger.Log("Telemetry", "[Storage] "+"Disk space available on the Hub: "+getDiskInfo())
		}
	}
	return nil
}

func downloadContent(vod VodObj, _heirarchy string) error {
	pushId := strconv.Itoa(vod.Content.UserDefined.PushId)
	// _heirarchy := vod.Content.UserDefined.MediaHouse + "/"
	// for i, x := range vod.Content.UserDefined.AncestorIds.File {
	// 	if i == 0 {
	// 		continue
	// 	}
	// 	_heirarchy = _heirarchy + x + "/"
	// }
	// _heirarchy = _heirarchy + vod.Content.UserDefined.MediaId + "/"
	//logger.Log("Info", "Downloading files for  "+_heirarchy+" via SES channel")
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
		//fmt.Println("MAP APPENDED:::::", folderMetadataFilesMap[metadatafileEntry.FolderId])
	}
	fmt.Println(folderMetadataFilesMap)
	folderBulkFilesMap := make(map[string][]FileInfo)
	folderBulkFilesMap[vod.Content.UserDefined.BulkFiles.File.FolderId] = []FileInfo{FileInfo{vod.Content.UserDefined.BulkFiles.File.Filename, vod.Content.UserDefined.BulkFiles.File.Checksum}}
	// for _, bulkfileEntry := range vod.Content.UserDefined.BulkFiles.File {
	// 	folderBulkFilesMap[bulkfileEntry.FolderId] = append(folderBulkFilesMap[bulkfileEntry.FolderId], FileEntry{bulkfileEntry.Filename, bulkfileEntry.Checksum})
	// }
	//folderBulkFilesMap := parseBulkFilesFromJson(vod.Content.UserDefined.BulkFiles.File)

	// fmt.Println("\nBulkfiles Map", folderBulkFilesMap)
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
			fileInfos[i] = make([]string, 4)
			fileInfos[i][0] = x.Name
			fileInfos[i][1] = filesURLMap[pushId+"_"+folder+"_"+x.Name]
			fileInfos[i][2] = x.Hashsum
			fileInfos[i][3] = "metadata"
		}
		for i, x := range folderBulkFilesMap[folder] {
			fileInfos[i] = make([]string, 4)
			fileInfos[metafilesLen+i][0] = x.Name
			fileInfos[metafilesLen+i][1] = filesURLMap[pushId+"_"+folder+"_"+x.Name]
			fileInfos[metafilesLen+i][2] = x.Hashsum
			fileInfos[metafilesLen+i][3] = "bulkfile"
		}
		subpathA := strings.Split(strings.Trim(subpath, "/"), "/")
		err := fs.CreateDownloadNewFolder(subpathA, DownloadFiles, fileInfos)
		if err != nil {
			log.Println(err)
			// if eval, ok := err.(*fs.FolderExistError); ok {
			// 	continue
			// }
			if err.Error() == "[Filesystem][CreateFolder]A folder with the same name at the requested level already exists" {
				continue
			}
			//logger.Log("Error", fmt.Sprintf("%s", err))
			log.Println("Error", fmt.Sprintf("%s", err))
			return err
		}
		log.Println("")
		fs.PrintBuckets()
		fs.PrintFileSystem()
		log.Println("")
	}

	return nil

}
func lastString(text string, separator string) string {
	last := strings.Split(text, separator)
	if len(last) > 0 {
		return last[len(last)-1]
	}
	return ""
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
				err := downloadContent(vod, "")
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
	return nil

}
