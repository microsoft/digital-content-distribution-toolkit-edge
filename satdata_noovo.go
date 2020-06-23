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
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const interval time.Duration = 2
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
			MediaHouse       string `json:"mediaHouse"`
			GenreDescription string `json:"genreDescription"`
			Description      string `json:"description"`
			IsDRM            bool   `json:"isDRM"`
			GenreTitle       string `json:"genreTitle"`
			MediaId          string `json:"mediaId"`
			Title            string `json:"title"`
			ParentTitle      string `json:"parentTitle"`
			ParentId         string `json:"parentId"`
			MetadataFiles    struct {
				File []struct {
					Filename string `json:"filename"`
					Checksum string `json:"checksum"`
					Filesize int    `json:"filesize"`
					Type     string `json:"type"`
					FolderId string `json:"folderId"`
				} `json:"file"`
			} `json:"metadataFiles"`
			PushId      int `json:"pushId"`
			AncestorIds struct {
				File []string `json:"file"`
			} `json:"ancestorIds"`
			HierarchyLevels int `json:"hierarchyLevels"`
			BulkFiles       struct {
				File struct {
					Filename string `json:"filename"`
					Checksum string `json:"checksum"`
					Filesize int    `json:"filesize"`
					Type     string `json:"type"`
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

type WrittenProgress struct {
	Total int64
}

func checkForVOD() {
	for true {
		fmt.Println("==================Polling NOOVO API for the content==============")
		logger.Log("Info", "Polling NOOVO API for the new content on the SAT")
		if err := callNoovoAPI(); err != nil {
			log.Println(err)
			logger.Log("Error", err.Error())
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
		if vod.Source == source {
			if matched, _ := regexp.MatchString(`^BINE.`, vod.Content.VODInfo.MovieID); matched {
				if err := downloadContent(vod); err != nil {
					fmt.Println(err)
					logger.Log("Error", err.Error())
					continue
				}
				logger.Log("Telemetry", "[DownloadSize] "+vod.Content.UserDefined.MediaHouse+":"+vod.Content.UserDefined.MediaId+" of size :"+strconv.FormatInt(getFolderSize(vod.Content.UserDefined.MediaHouse, vod.Content.UserDefined.MediaId), 10)+"downloaded on the Hub")
				logger.Log("Telemetry", "[ContentSyncChannel] "+vod.Content.UserDefined.MediaHouse+":"+vod.Content.UserDefined.MediaId+"synced successfully via SES Channel")
				logger.Log("Telemetry", "[Storage] "+"Disk space available on the Hub: "+getDiskInfo())
			}
		}
	}
	return nil
}

func downloadContent(vod VodObj) error {
	id := vod.Content.UserDefined.MediaId
	mediaHouse := vod.Content.UserDefined.MediaHouse
	parent := vod.Content.UserDefined.ParentId
	pushId := strconv.Itoa(vod.Content.UserDefined.PushId)
	fmt.Println("=======Processing for content ID::: ", id, " of MediaHouse::: ", mediaHouse)
	if haskey, err := KeyExistsInDb(mediaHouse, parent, id); err != nil {
		return err
	} else if haskey {
		fmt.Println("CONTENT: ", id, " Already exist. Nothing new to Add")
		return nil
	}
	fmt.Println("Downloading files for MediaHouse:: ", mediaHouse, " Content ID::", id)
	logger.Log("Info", "Downloading files for MediaHouse: "+mediaHouse+" Content ID:"+id+"via SES channel")
	// filesUrlMap of the files with the url
	filesURLMap := make(map[string]string)
	filesURLMap[lastString(vod.Content.Pictures.Thumbnail.File.Filename, "/")] = vod.Content.Pictures.Thumbnail.File.Filename
	filesURLMap[lastString(vod.Content.Pictures.Cover.File.Filename, "/")] = vod.Content.Pictures.Cover.File.Filename
	filesURLMap[lastString(vod.Content.Videos.Movie.File.Filename, "/")] = vod.Content.Videos.Movie.File.Filename
	for _, datafile := range vod.Content.Data.File {
		filesURLMap[lastString(datafile.Filename, "/")] = datafile.Filename
	}
	folderMetadataFilesMap := make(map[string][]FileEntry)
	for _, metadatafileEntry := range vod.Content.UserDefined.MetadataFiles.File {
		folderMetadataFilesMap[metadatafileEntry.FolderId] = append(folderMetadataFilesMap[metadatafileEntry.FolderId], FileEntry{metadatafileEntry.Filename, metadatafileEntry.Checksum})
		//fmt.Println("MAP APPENDED:::::", folderMetadataFilesMap[metadatafileEntry.FolderId])
	}
	fmt.Println(folderMetadataFilesMap)
	folderBulkFilesMap := make(map[string][]FileEntry)
	folderBulkFilesMap[vod.Content.UserDefined.BulkFiles.File.FolderId] = []FileEntry{FileEntry{vod.Content.UserDefined.BulkFiles.File.Filename, vod.Content.UserDefined.BulkFiles.File.Checksum}}
	// for _, bulkfileEntry := range vod.Content.UserDefined.BulkFiles.File {
	// 	folderBulkFilesMap[bulkfileEntry.FolderId] = append(folderBulkFilesMap[bulkfileEntry.FolderId], FileEntry{bulkfileEntry.Filename, bulkfileEntry.Checksum})
	// }
	//folderBulkFilesMap := parseBulkFilesFromJson(vod.Content.UserDefined.BulkFiles.File)

	fmt.Println("\nBulkfiles Map", folderBulkFilesMap)
	for k, v := range folderMetadataFilesMap {
		fmt.Println("KEY---------------", k)
		metadataFiles := getMetadataFiles(mediaHouse, k)
		if len(metadataFiles) > 0 {
			fmt.Println("Files for FolderId: ", k, " already exist.")
			continue
		}
		fmt.Println("Downloading metadata files")
		err := downloadSatFiles(mediaHouse, pushId, k, v, filesURLMap)
		if err != nil {
			return errors.New(" FolderId: " + k + "Metadatafiles download via SES: FAILED :" + err.Error())
		}
		//update db
		err = addMetadataFiles(mediaHouse, k, v)
		if err != nil {
			return errors.New(" FolderId: " + k + "Metadatafiles download via SES: FAILED :" + err.Error())
		}
		fmt.Println("Updated the DB::::::: for ", k)
		logger.Log("Info", "Folder Id:"+string(k)+" Metadatafiles download via SES: SUCCESS")
	}
	for k, v := range folderBulkFilesMap {
		fmt.Println("Downloading Bulk files")
		err := downloadSatFiles(mediaHouse, pushId, k, v, filesURLMap)
		if err != nil {
			return errors.New("Could not download bulk files for FolderId ::: " + k + ">>>>>" + err.Error())
		}
		if err = addBulkFiles(mediaHouse, k, v); err != nil {
			return err
		}
		logger.Log("Info", "Folder Id:"+string(k)+" Bulkfiles download via SES: SUCCESS")
	}
	fmt.Println("adding heirarchy")
	ancestorIds := vod.Content.UserDefined.AncestorIds.File
	for i := 0; i < len(ancestorIds)-1; i++ {
		children := getChildren(mediaHouse, ancestorIds[i])
		present := false
		for _, child := range children {
			if child.ID == ancestorIds[i+1] {
				present = true
				break
			}
		}
		if !present {
			if err := addFolder(mediaHouse, ancestorIds[i], FolderStructureEntry{ancestorIds[i+1], true, "bine_metadata.json", 0, []string{}}); err != nil {
				return err
			}
		}
	}
	// update dir str for the VOD(leaf node)
	if err := addFolder(mediaHouse, ancestorIds[len(ancestorIds)-1], FolderStructureEntry{id, false, "bine_metadata.json", 0, []string{}}); err != nil {
		return err
	}
	logger.Log("Info", "Folder Id: "+id+"Adding Heirarchy in the Bolt DB: SUCCESS")
	logger.Log("Info", mediaHouse+" : "+id+" downloaded successfully via SES")
	return nil

}
func downloadSatFiles(mediaHouse, pushId, folderId string, fileEntries []FileEntry, filesURLMap map[string]string) error {
	//TODO: If the folder is partially downloaded???
	// Create output dir
	folderDir := path.Join(".", mediaHouse, folderId)
	if err := os.MkdirAll(folderDir, 0700); err != nil {
		return err
	}
	fmt.Println("=========CREATED FOLDER AT=======", folderDir)
	for _, fileEntry := range fileEntries {
		outpath := path.Join(folderDir, fileEntry.Name)
		url := filesURLMap[pushId+"_"+folderId+"_"+fileEntry.Name]
		actualHash := fileEntry.HashSum
		if err := download(outpath, url); err != nil {
			return err
		}
		fmt.Println("========CHECKING CHECKSUM=========")
		computedHash := computeSHA256(outpath)
		if computedHash != actualHash {
			//TODO: delete the folder????
			return errors.New("Checksum did not match for: " + pushId + "_" + folderId + "_" + fileEntry.Name)
		}
		fmt.Println("Checksum matched for FILE::: ", pushId+"_"+folderId+"_"+fileEntry.Name)
	}
	return nil
}

func download(outpath, url string) error {
	//TODO: handle if the url does not exist
	outputStream, err := os.Create(outpath)
	if err != nil {
		return err
	}
	defer outputStream.Close()
	fmt.Println("WRITING TO FILE===================", outpath)

	fmt.Println("DOWNLOADING FILE AT URL>>>>>>>>>>>>>", url)
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	_, err = io.Copy(outputStream, io.TeeReader(res.Body, &WrittenProgress{}))
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("=============DONE DOWNLOADING============")
	return nil

}

func (wp *WrittenProgress) Write(p []byte) (int, error) {
	written := len(p)
	wp.Total += int64(written)
	fmt.Printf("\r%s", strings.Repeat(" ", 100))
	fmt.Printf("\rDownloaded::: %d MB", wp.Total/1024/1024)
	return written, nil
}

func deleteMediaFolder(mediaHouse string, parent string, id string) {
	pathToBeDeleted := filepath.Join(mediaHouse, id)
	if err := os.RemoveAll(pathToBeDeleted); err != nil {
		log.Println("Error in deleting the folder id ", id, "::::", err)
	} else {
		fmt.Println("Deleted Directory::::", pathToBeDeleted)
	}
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
				err := downloadContent(vod)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
	return nil

}
