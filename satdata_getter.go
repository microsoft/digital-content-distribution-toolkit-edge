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
	"log"
	//"strconv"
)

const interval time.Duration = 20
const source string = "mstore"

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
	MediaId string
	MediaHouse string
	ParentId string
	HierarchyLevels int		//number of levels excluding the leaf node
	AncestorIds []string	//ids of all the parents of a leaf node from top to bottom
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
	for true{
		fmt.Println("==================Polling NOOVO API for the content==============");
		if err := callNoovoAPI(); err!= nil {
			log.Println(err)
		}	
		time.Sleep(interval * time.Minute)
	}
}

func callNoovoAPI() error{
	res, err := http.Get("http://localhost:40000/vod/list")
	if err != nil {
		//fmt.Println("ERROR in API CALL::", err)
		return err
	}
	defer res.Body.Close()
	jsonRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		//fmt.Println("ERROR in READING RESPONSE::", err)
		return err
	}
	//fmt.Println("RESPONSE::::::::::", string(jsonRes))
	var vods []VodObj
	jsonErr := json.Unmarshal(jsonRes, &vods)
	if jsonErr != nil{
		return err
	}
	for i := range vods{
		if vods[i].Source == source{
			id := vods[i].Content.UserDefined.MediaId
			//id := strconv.Itoa(vods[i].Content.PushInfo.CID)
			mediaHouse := vods[i].Content.UserDefined.MediaHouse
			parent := vods[i].Content.UserDefined.ParentId
			if haskey, _ := KeyExistsInDb(mediaHouse, parent, id); !haskey{
				fmt.Println("========NEW CONTENT=======Downloading files for ID::", id)
				var filesToBeDownloaded []FileInfo
				file := FileInfo{vods[i].Content.UserDefined.Files.Thumbnail.Filename, vods[i].Content.Pictures.Thumbnail.File.Filename, vods[i].Content.UserDefined.Files.Thumbnail.Size}
				filesToBeDownloaded = append(filesToBeDownloaded, file)
				file = FileInfo{vods[i].Content.UserDefined.Files.Video.Filename, vods[i].Content.Videos.Movie.File.Filename, vods[i].Content.UserDefined.Files.Video.Size}
				filesToBeDownloaded = append(filesToBeDownloaded, file)
				err := downloadFiles(mediaHouse, parent, id, filesToBeDownloaded)
				if err != nil{
					//delete the partially downloaded folder
					deleteMediaFolder(mediaHouse, parent, id)
					return errors.New("COULD NOT DOWNLOAD:::"+ err.Error())

				}
				//Update DB
				if err:= updateDb(mediaHouse, id, vods[i].Content.UserDefined.HierarchyLevels, vods[i].Content.UserDefined.AncestorIds,vods[i].Content.UserDefined.Files ); err!=nil {
					return err
				}		
			}else{
				fmt.Println(id," Already exist. Nothing New to Add")
			}
		
		}
	}
	return nil
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
		//fmt.Println("Error in creating ouptput file:: ", err)
		return err
	} 
	fmt.Println("WRITING TO FILE===================", outpath)
	fmt.Println("GETTING FILE AT URL>>>>>>>>>>>>>" , files[i].Url)
	res, err := http.Get(files[i].Url)
	if err != nil{
		//fmt.Println("Error in getting the response from File URL::::: ",err)
		return err
	}
	defer res.Body.Close()
	defer f.Close()
	written, err := io.Copy(f, res.Body)
	//written, err := io.Copy(f, io.TeeReader(res.Body, &WrittenProgress{}))
	if err != nil{
		//fmt.Println("ERROR WRITING TO THE FILE::::", err)
		return err
	}
	fmt.Println("=============DONE DOWNLOADING============")
	fmt.Println("--------No. of bytes written::: ",written)
	fmt.Println("------------ Actual size:::",files[i].Size)
	//Check if downloaded size same as the filesize received in the response
	// if uint64(written) != files[i].Size{
	// 	return errors.New("Mismatch in the actual filesize and downloaded filesize")
	// }
	//fmt.Println("-------Correct Size Downloaded-------")
	}
	//TODO: Compute and check hashsum
	return nil
		

}
func deleteMediaFolder(mediaHouse string, parent string, id string){
	pathToBeDeleted := filepath.Join(mediaHouse, id)
	if err := os.RemoveAll(pathToBeDeleted); err!=nil {
		log.Println("Error in deleting the folder id ", id, "::::", err)
	}else{
		fmt.Println("Deleted Directory::::", pathToBeDeleted)
	}
}
func updateDb(mediaHouse string, id string, levels int, ancestorIds []string, files Metafiles) error{	
	// check and update directory str for the parents of the VOD
	for i := 0; i < (levels - 1) ; i++ {
		if haskey, _ := KeyExistsInDb(mediaHouse, ancestorIds[i], ancestorIds[i+1]); !haskey{
			folderStructureEntry := FolderStructureEntry{ancestorIds[i+1], true, ""} //infometadatafilename field empty for now
			if err := addFolder( mediaHouse, ancestorIds[i], folderStructureEntry); err!= nil {
				return err
			}
			//TODO: update metadata files of the folder
		}	
	}
	// update dir str for the VOD(leaf node)
	folderStructureEntry := FolderStructureEntry{id, false, ""}
	//fmt.Println(ancestorIds[levels - 1])
	if err := addFolder( mediaHouse, ancestorIds[levels - 1], folderStructureEntry); err!= nil {
		return err
	}
	
	//update metadata and bulk files for VOD
	bulkFiles := []FileEntry{FileEntry{files.Video.Filename, files.Video.Checksum}}
	metadataFiles := []FileEntry{FileEntry{files.Thumbnail.Filename, files.Thumbnail.Checksum}}
	if err := addMetadataFiles( mediaHouse, id, metadataFiles); err!= nil {
		return err
	}
	if err := addBulkFiles( mediaHouse, id, bulkFiles); err!= nil {
		return err
	}
	return nil

}




