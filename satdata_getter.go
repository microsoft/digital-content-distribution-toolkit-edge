package main

import (
	"fmt"
	"io/ioutil"
	"io"
	"os"
	"time"
	"net/http"
	"encoding/json"
	"path/filepath"
	"errors"
)

const interval time.Duration = 20

type VodObj struct{
	Source string
	Content Content
}
type Content struct{
	UserDefined UserDefined
	Videos Videos
	Pictures Pictures
}
type UserDefined struct{
	Media_id string
	MediaHouse string
	Parent_id string
	HierarchyLevels int		//number of levels excluding the leaf node
	Ancestor_ids []string	//ids of all the parents of a leaf node from top to bottom
	Files Metafiles
}
type Metafiles struct{
	Thumbnail  MetafileInfo
	Video MetafileInfo
}

type MetafileInfo struct{
	Filename string
	Size uint64
	Checksum string
}

type  File struct{
	Filesize uint64
	Filename string
}

type Videos struct{
	Movie Movie
}
type Movie struct{
	Duration string
	File File
}
type Pictures struct{
	Thumbnail Thumbnail
	//Cover Cover
}
type Thumbnail struct{
	File File
}
// type Cover struct{
// 	File File
// }

// Info of the files to be downloaded
type FileInfo struct{  
	Filename string
	Url string
	Size uint64
	
}

func checkForVOD(){
	callNoovoAPI()
	for true{
		time.Sleep(interval * time.Minute)
		fmt.Println("==================Polling NOOVO API for the content==============");
		callNoovoAPI()	
	}
}

func callNoovoAPI(){
	res, err := http.Get("http://localhost:40000/vod/list")
	if err != nil {
		fmt.Println("ERROR in API CALL::", err)
		return
	}
	defer res.Body.Close()
	jsonRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("ERROR in READING RESPONSE::", err)
		return
	}
	//fmt.Println("RESPONSE::::::::::", string(jsonRes))
	var vods []VodObj
	jsonErr := json.Unmarshal(jsonRes, &vods)
	if jsonErr != nil{
		fmt.Println(jsonErr)
		return
	}
	for i := range vods{
		if vods[i].Source == "mstore"{
			id := vods[i].Content.UserDefined.Media_id
			mediaHouse := vods[i].Content.UserDefined.MediaHouse
			parent := vods[i].Content.UserDefined.Parent_id
			if !KeyExistsInDb(mediaHouse, parent, id){
				fmt.Println("========NEW CONTENT=======Downloading files for ID::", id)
				var filesToBeDownloaded []FileInfo
				file := FileInfo{vods[i].Content.UserDefined.Files.Thumbnail.Filename, vods[i].Content.Pictures.Thumbnail.File.Filename, vods[i].Content.UserDefined.Files.Thumbnail.Size}
				filesToBeDownloaded = append(filesToBeDownloaded, file)
				file = FileInfo{vods[i].Content.UserDefined.Files.Video.Filename, vods[i].Content.Videos.Movie.File.Filename, vods[i].Content.UserDefined.Files.Video.Size}
				filesToBeDownloaded = append(filesToBeDownloaded, file)
				err := downloadFiles(mediaHouse, parent, id, filesToBeDownloaded)
				if err == nil{
					//Update DB
					err:= updateDb(mediaHouse, id, vods[i].Content.UserDefined.HierarchyLevels, vods[i].Content.UserDefined.Ancestor_ids,vods[i].Content.UserDefined.Files )
					if err!= nil{
						fmt.Println("ERROR Updating the DB::", err)
						//TODO: delete the downloaded files?? OR Reattempt updating the DB ???
					}

				}else{
					fmt.Println("COULD NOT DOWNLOAD:::", err)
					//delete the partially downloaded folder
					deleteFolder(mediaHouse, parent, id)
				}
				
			}else{
				fmt.Println(id," Already exist. Nothing New to Add")
			}
		
		}
	}
}
func downloadFiles(mediaHouse string, parent string, id string, files []FileInfo) error{
	// Create output dir
	outpathDir := filepath.Join(".",mediaHouse, id)
	if err:= os.MkdirAll(outpathDir,os.ModeDir); err!=nil{
		return err
	}
	fmt.Println("=========CREATED FOLDER AT=======", outpathDir)
	for i := range files{
		outpath := outpathDir + "/" + files[i].Filename
		f, err := os.Create(outpath)
	if err != nil{
		fmt.Println("Error in creating ouptput file:: ", err)
		return err
	} 
	fmt.Println("WRITING TO FILE===================", outpath)
	fmt.Println("GETTING FILE AT URL>>>>>>>>>>>>>" , files[i].Url)
	res, err := http.Get(files[i].Url)
	if err != nil{
		fmt.Println("Error in getting the response from File URL::::: ",err)
		return err
	}
	defer res.Body.Close()
	defer f.Close()
	written, err := io.Copy(f, res.Body)
	//written, err := io.Copy(f, io.TeeReader(res.Body, &WrittenProgress{}))
	if err != nil{
		fmt.Println("ERROR WRITING TO THE FILE::::", err)
		return err
	}
	fmt.Println("=============DONE DOWNLOADING============")
	fmt.Println("--------No. of bytes written::: ",written)
	fmt.Println("------------ Actual size:::",files[i].Size)
	//Check if downloaded size same as the filesize received in the response
	if uint64(written) != files[i].Size{
		return errors.New("Mismatch in the actual filesize and downloaded filesize")
	}
	fmt.Println("-------Correct Size Downloaded-------")
	}
	return nil
		

}
func deleteFolder(mediaHouse string, parent string, id string){
	pathToBeDeleted := filepath.Join(mediaHouse, id)
	if err := os.RemoveAll(pathToBeDeleted); err!=nil {
		fmt.Println("Error in deleting the folder id ", id, "::::", err)
	}
}
func updateDb(mediaHouse string, id string, levels int, ancestor_ids []string, files Metafiles) error{
	// check and update directory str for the parents of the VOD
	for i := 0; i < (levels - 1) ; i++ {
		if(!KeyExistsInDb(mediaHouse, ancestor_ids[0], ancestor_ids[i+1])){
			folderStructureEntry := FolderStructureEntry{ancestor_ids[i+1], true, ""} //infometadatafilename field empty for now
			if err := updateFolderStr(mediaHouse, ancestor_ids[i], folderStructureEntry); err!=nil {
				return err
			}
			//TODO: update metadata files of the folder
		}	
	}
	// update dir str for the VOD(leaf node)
	folderStructureEntry := FolderStructureEntry{id, false, ""}
	if err:= updateFolderStr(mediaHouse, ancestor_ids[levels - 1], folderStructureEntry); err!=nil {
		return err
	}
	//update metadata and bulk files for VOD
	bulkFiles := []FileEntry{FileEntry{files.Video.Filename, files.Video.Checksum}}
	metadataFiles := []FileEntry{FileEntry{files.Thumbnail.Filename, files.Thumbnail.Checksum}}
	if err:= updateFileInfoInDb(mediaHouse, id, metadataFiles, bulkFiles); err!= nil {
		return err
	}
	return nil

}




