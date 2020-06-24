package main

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
		EditorID              string `json:"editorID"`
		DisplayedName         string `json:"displayedName"`
		Category              string `json:"category"`
		ShortSummary          string `json:"shortSummary"`
		ProductionNationality string `json:"productionNationality"`
		ShortTitle            string `json:"shortTitle"`
		LongTitle             string `json:"longTitle"`
		Series                string `json:"series"`
		UserDefined           struct {
			MediaHouse       string `json:"mediaHouse"`
			GenreDescription string `json:"genreDescription"`
			Description      string `json:"description"`
			IsDRM            bool   `json:"isDRM"`
			GenreTitle       string `json:"genreTitle"`
			MediaId          string `json:"mediaId"`
			ParentId         string `json:"parentId"`
			Title            string `json:"title"`
			ParentTitle      string `json:"parentTitle"`
			MetadataFiles    struct {
				File []struct {
					Filename string `json:"filename"`
					Checksum string `json:"checksum"`
					Filesize int    `json:"filesize"`
					FolderId string `json:"folderId"`
					Type     string `json:"type"`
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
					FolderId string `json:"folderId"`
					Type     string `json:"type"`
				} `json:"file"`
			} `json:"bulkFiles"`
		} `json:"userDefined"`
		MovieId           string `json:"movieID"`
		Price             string `json:"price"`
		CID               string `json:"CID"`
		VideoFilename     string `json:"video filename"`
		TrailerFilename   string `json:"trailer filename"`
		CoverFilename     string `json:"cover filename"`
		ThumbnailFilename string `json:"thumbnail filename"`
	} `json:"metadata"`
}

func checkForVODViaMstore() error {
	res, err := http.Get("http://localhost:8134/listcontentsbycategory/MSR")
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
			fmt.Println(err)
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
	id := vod.Metadata.UserDefined.MediaId
	mediaHouse := vod.Metadata.UserDefined.MediaHouse
	pushId := strconv.Itoa(vod.Metadata.UserDefined.PushId)
	filepathMap := make(map[string]string)
	//TODO: Check if id already exist or not
	fmt.Println("Downloading files for MediaHouse:: ", mediaHouse, " Content ID::", id)
	filepathMap[lastString(vod.Metadata.ThumbnailFilename, "/")] = vod.Metadata.ThumbnailFilename
	filepathMap[lastString(vod.Metadata.CoverFilename, "/")] = vod.Metadata.CoverFilename
	filepathMap[lastString(vod.Metadata.VideoFilename, "/")] = vod.Metadata.VideoFilename
	// add data section filepaths
	folderMetadataFilesMap := make(map[string][]FileEntry)
	for _, metadatafileEntry := range vod.Metadata.UserDefined.MetadataFiles.File {
		folderMetadataFilesMap[metadatafileEntry.FolderId] = append(folderMetadataFilesMap[metadatafileEntry.FolderId], FileEntry{metadatafileEntry.Filename, metadatafileEntry.Checksum})
		//fmt.Println("MAP APPENDED:::::", folderMetadataFilesMap[metadatafileEntry.FolderId])
	}
	fmt.Println(folderMetadataFilesMap)
	folderBulkFilesMap := make(map[string][]FileEntry)
	folderBulkFilesMap[vod.Metadata.UserDefined.BulkFiles.File.FolderId] = []FileEntry{FileEntry{vod.Metadata.UserDefined.BulkFiles.File.Filename, vod.Metadata.UserDefined.BulkFiles.File.Checksum}}

	fmt.Println("\nBulkfiles Map", folderBulkFilesMap)

	for k, v := range folderMetadataFilesMap {
		fmt.Println("KEY---------------", k)
		//TODO: to be removed
		if k != id {
			continue
		}
		metadataFiles := getMetadataFiles(mediaHouse, k)
		if len(metadataFiles) > 0 {
			fmt.Println("Files for FolderId: ", k, " already exist.")
			continue
		}
		fmt.Println("Downloading metadata files")
		err := copyFiles(mediaHouse, pushId, k, v, filepathMap)
		if err != nil {
			return errors.New("Could not download metadta files for FolderId ::: " + k + ">>>>>" + err.Error())
		}
		//update db
		err = addMetadataFiles(mediaHouse, k, v)
		if err != nil {
			return err
		}
		fmt.Println("Updated the DB::::::: for ", k)
	}
	for k, v := range folderBulkFilesMap {
		fmt.Println("Downloading Bulk files")
		err := copyFiles(mediaHouse, pushId, k, v, filepathMap)
		if err != nil {
			return errors.New("Could not download bulk files for FolderId ::: " + k + ">>>>>" + err.Error())
		}
		if err = addBulkFiles(mediaHouse, k, v); err != nil {
			return err
		}
	}
	fmt.Println("adding heirarchy")
	ancestorIds := vod.Metadata.UserDefined.AncestorIds.File
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
	return nil
}

func copyFiles(mediaHouse, pushId, folderId string, fileEntries []FileEntry, filepathMap map[string]string) error {
	folderDir := path.Join(".", mediaHouse, folderId)
	if err := os.MkdirAll(folderDir, 0700); err != nil {
		return err
	}
	fmt.Println("=========CREATED FOLDER AT=======", folderDir)
	for _, fileEntry := range fileEntries {
		destpath := path.Join(folderDir, fileEntry.Name)
		sourcepath := filepathMap[pushId+"_"+folderId+"_"+fileEntry.Name]
		//TODO: to be removed
		if sourcepath == "" {
			continue
		}
		actualHash := fileEntry.HashSum
		if err := copySingleFile(destpath, sourcepath); err != nil {
			return err
		} else {
			fmt.Println("========CHECKING CHECKSUM=========")
			computedHash := computeSHA256(destpath)
			if computedHash != actualHash {
				//TODO: delete the folder????
				return errors.New("Checksum did not match for: " + pushId + "_" + folderId + "_" + fileEntry.Name)
			}
			fmt.Println("Checksum matched for FILE::: ", pushId+"_"+folderId+"_"+fileEntry.Name)
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
