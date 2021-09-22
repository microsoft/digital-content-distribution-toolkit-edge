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
	"regexp"
	"strconv"
	"strings"
	"time"

	capi "binehub/cloudapihandler"
	l "binehub/logger"
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
			Title     string `json:"title"`
			Contentid string `json:"contentid"`
		} `json:"userDefined"`
		MovieId           string    `json:"movieID"`
		ValidityStartDate time.Time `json:"validityStartDate"`
		ValidityEndDate   time.Time `json:"validityEndDate"`
		CID               string    `json:"CID"`
		VideoFilename     string    `json:"video filename"`
		TrailerFilename   string    `json:"trailer filename"`
		CoverFilename     string    `json:"cover filename"`
		URLForDataFiles   string    `json:"urlForDataFiles"`
		DataFiles         struct {
			File []struct {
				Filename string `json:"filename"`
				Filesize int    `json:"filesize"`
			} `json:"file"`
		} `json:"dataFiles"`
		ThumbnailFilename string `json:"thumbnail filename"`
	} `json:"metadata"`
}
type OldVodInfo struct {
	Status   string `json:"status"`
	Metadata struct {
		UserDefined struct {
			MediaId       string `json:"mediaId"`
			MediaHouse    string `json:"mediaHouse"`
			AncestorIds   string `json:"ancestorIds"`
			MetadataFiles struct {
				File []struct {
					Filename string `json:"filename"`
					Filesize int    `json:"filesize"`
					Checksum string `json:"checksum"`
					FolderId string `json:"folderId"`
				} `json:"file"`
			} `json:"metadataFiles"`
			BulkFiles struct {
				File []struct {
					Filename string `json:"filename"`
					Filesize int    `json:"filesize"`
					Checksum string `json:"checksum"`
					FolderId string `json:"folderId"`
				} `json:"file"`
			} `json:"bulkFiles"`
			PushId string `json:"pushId"`
		} `json:"userDefined"`
		MovieId         string    `json:"movieID"`
		CID             string    `json:"CID"`
		ValidityEndDate time.Time `json:"validityEndDate"`
		VideoFilename   string    `json:"video filename"`
		TrailerFilename string    `json:"trailer filename"`
		CoverFilename   string    `json:"cover filename"`
		URLForDataFiles string    `json:"urlForDataFiles"`
		DataFiles       struct {
			File []struct {
				Filename string `json:"filename"`
				Filesize int    `json:"filesize"`
			} `json:"file"`
		} `json:"dataFiles"`
		ThumbnailFilename string `json:"thumbnail filename"`
	} `json:"metadata"`
}

func pollMstore(interval int) {
	for true {
		fmt.Println("==================Polling MStore API ==============")
		if err := checkForVODViaMstore(); err != nil {
			log.Println("[SatdataMstore][pollMstore] Error", fmt.Sprintf("%s", err))
			sm := new(l.MessageSubType)
			sm.StringMessage = "SatdataMstore: " + err.Error()
			logger.Log("Error", sm)
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
func checkForVODViaMstore() error {
	res, err := http.Get("http://host.docker.internal:8134/listcontents")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	jsonRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var vodlist VodList
	jsonErr := json.Unmarshal(jsonRes, &vodlist)
	if jsonErr != nil {
		return jsonErr
	}
	fmt.Println("[Satdata_mstore][checkForVODViaMstore] NO. OF CONTENTS ON THE SAT: ", len(vodlist.ListContents))
	for i, id := range vodlist.ListContents {
		fmt.Println("=======(", i, ") Processing for CID:=====", id.ContentID)
		err := getMetadataAPI(id.ContentID)
		if err != nil {
			log.Println("[SatdataMstore][checkForVODViaMstore] Error ", fmt.Sprintf("%s", err))
			//logger.Log("Error", "SatdataMstore", map[string]string{"Message": err.Error()})
			sm := l.MessageSubType{StringMessage: "SatdataMstore: " + err.Error()}
			logger.Log("Error", &sm)
			continue
		}
	}
	// Printing the final file sys after processing all the SAT contents
	fmt.Println("=========================")
	fmt.Println("Printing final buckets after processing all the contents on the SAT....")
	fs.PrintBuckets()
	fmt.Println("==========================")
	return nil
}

func getMetadataAPI(contentId string) error {
	query := "http://host.docker.internal:8134/getmetadata/" + contentId
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
	if vod.Status == "15" {
		return fmt.Errorf("getMetadataAPI: No metadata available for CID %s ", contentId)
	}
	if len(vod.Metadata.UserDefined.Contentid) == 0 {
		log.Println("[SatdataMstore][checkForVODViaMstore] Invalid Content Metadata")
		return nil
	}

	if _heirarchy, err := getMstoreFiles(vod); err != nil {
		//TODO:
		logger.Log_old("Telemetry", "ContentSyncInfo", map[string]string{"DownloadStatus": "FAIL", "FolderPath": _heirarchy, "Channel": "SES"})
		return err
	}
	return nil
}
func getMstoreFiles(vod VodInfo) (string, error) {
	//TODO: rebroadcast of the same contentID only after the first one is deleted ? Assumption ??
	cid := vod.Metadata.CID
	contentid := vod.Metadata.UserDefined.Contentid
	path, _ := fs.GetAssetFolderPathFromDB(contentid)
	if path != "" {
		fmt.Println("Mapping already exist for the AssetID: ", contentid)
		log.Println(fmt.Sprintf("skipping AssetId: %s as the mapping with the same id already exist", contentid))
		sm := new(l.MessageSubType)
		sm.StringMessage = "SatdataMstore: Mapping for " + contentid + " already exist."
		logger.Log("Info", sm)
		return contentid, nil
	}
	pathToAsset := vod.Metadata.VideoFilename
	err := fs.CreateSatelliteIndexing(cid, contentid, pathToAsset)
	if err != nil {
		log.Println("[SatdataMstore][getMstoreFiles] ", fmt.Sprintf("AssetId: %s cannot be indexed", contentid, " Error: ", err))
		return contentid, err
	}
	fmt.Println("Printing buckets")
	fs.PrintBuckets()
	fmt.Println("====================")
	log.Println(fmt.Sprintf("[SatdataMstore][getMstoreFiles] AssetID: %s synced via SES", contentid))
	log.Println(fmt.Sprintf("[SatdataMstore][getMstoreFiles] AssetSize: %2f MB", getDirSizeinMB(pathToAsset)))
	log.Println(fmt.Sprintf("[SatdataMstore][getMstoreFiles] Disk space available on the Hub: %2f", getDiskInfo()))
	//send telemetry
	_sm := l.AssetInfo{AssetId: vod.Metadata.UserDefined.Contentid, Size: getDirSizeinMB(pathToAsset)}
	sm := l.MessageSubType{AssetInfo: _sm}
	logger.Log("AssetDownloadOnDeviceFromSES", &sm)
	sm = l.MessageSubType{FloatValue: getDiskInfo()}
	logger.Log("HubStorage", &sm)

	//call to CMS API
	var apidata = new(capi.ApiData)
	apidata.Id = contentid
	apidata.ApiType = capi.Downloaded
	apidata.OperationTime = time.Now().UTC()
	apidata.RetryCount = 0

	data, err := json.Marshal(apidata)

	if err != nil {
		log.Printf("Error in marshalling api data for downloaded content with id %v and error %s ", contentid, err)
	} else {
		err = fs.AddContent(contentid, data)

		if err != nil {
			log.Printf("Error while storing downloaded content in pending requests db %s ", err)
		}
	}

	return contentid, nil
}

func populateFileInfos(folder string, folderMetadataFilesMap map[string][]FileInfo, folderBulkFilesMap map[string][]FileInfo, filepathMap map[string]string, deadline time.Time) [][]string {
	metafilesLen, bulkfilesLen := len(folderMetadataFilesMap[folder]), len(folderBulkFilesMap[folder])
	fileInfos := make([][]string, metafilesLen+bulkfilesLen+1)
	for i, x := range folderMetadataFilesMap[folder] {
		fileInfos[i] = make([]string, 5)
		fileInfos[i][0] = x.Name
		fileInfos[i][1] = filepathMap[x.Name]
		fileInfos[i][2] = x.Hashsum
		fileInfos[i][3] = "metadata"
		fileInfos[i][4] = strconv.FormatInt(deadline.Unix(), 10)
	}
	for i, x := range folderBulkFilesMap[folder] {
		fileInfos[metafilesLen+i] = make([]string, 5)
		fileInfos[metafilesLen+i][0] = x.Name
		fileInfos[metafilesLen+i][1] = filepathMap[x.Name]
		fileInfos[metafilesLen+i][2] = x.Hashsum
		fileInfos[metafilesLen+i][3] = "bulkfile"
		fileInfos[metafilesLen+i][4] = strconv.FormatInt(deadline.Unix(), 10)
	}
	// info for the folder deadline
	fileInfos[metafilesLen+bulkfilesLen] = make([]string, 5)
	fileInfos[metafilesLen+bulkfilesLen][4] = strconv.FormatInt(deadline.Unix(), 10)
	return fileInfos
}

func getMstoreFiles_Old(vod OldVodInfo) (string, error) {

	cid := vod.Metadata.CID
	deadline := vod.Metadata.ValidityEndDate
	var _heirarchy string
	if vod.Metadata.UserDefined.AncestorIds != "" {
		_heirarchy = vod.Metadata.UserDefined.MediaHouse + vod.Metadata.UserDefined.AncestorIds + "/" + vod.Metadata.UserDefined.MediaId
	} else {
		_heirarchy = vod.Metadata.UserDefined.MediaHouse + "/" + vod.Metadata.UserDefined.MediaId
	}
	fmt.Println("Heirarchy ", _heirarchy)
	deleteFlag := false
	path, _ := fs.GetActualPathForAbstractedPath(_heirarchy)
	if path != "" {
		log.Println(_heirarchy + " already exist.")
		sm := new(l.MessageSubType)
		sm.StringMessage = "SatdataMstore: " + _heirarchy + " already exist."
		logger.Log("Info", sm)
		if deleteFlag {
			if err := deleteAPI(cid); err != nil {
				log.Println("[SatdataMstore] Error", fmt.Sprintf("%s", err))
			}
			//logger.Log("Info", "SatdataMstore", map[string]string{"CID": cid, "Message": "Deleted from Mstore"})
			log.Println("[SatdataMstore][getMstoreFiles] Info ", fmt.Sprintf("Already Exist. Deleted from Mstore Db: %s", _heirarchy))
		}
		return _heirarchy, nil
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
	}
	folderBulkFilesMap := make(map[string][]FileInfo)
	for _, bulkfileEntry := range vod.Metadata.UserDefined.BulkFiles.File {
		folderBulkFilesMap[bulkfileEntry.FolderId] = append(folderBulkFilesMap[bulkfileEntry.FolderId], FileInfo{bulkfileEntry.Filename, bulkfileEntry.Checksum})
	}

	hierarchy := strings.Split(strings.Trim(_heirarchy, "/"), "/")
	log.Println("heirarchy of Contents from SAT before splicing: ", hierarchy)
	log.Println("heirarchy of the Content from SAT after splicing: ", hierarchy)
	subpath := ""
	var leafFileInfos [][]string
	for idx, folder := range hierarchy {
		subpath = subpath + folder + "/"
		log.Println("Printing buckets")
		fs.PrintBuckets()
		log.Println("Printing file sys")
		//fs.PrintFileSystem()
		log.Println("====================")
		log.Println("Creating subpath of the heirarchy: ", subpath)

		if idx != len(hierarchy)-1 {
			fileInfos := populateFileInfos(folder, folderMetadataFilesMap, folderBulkFilesMap, filepathMap, deadline)

			subpathA := strings.Split(strings.Trim(subpath, "/"), "/")
			err := fs.CreateDownloadNewFolder(subpathA, copyFiles, fileInfos, false, "")
			if err != nil {
				// if eval, ok := err.(*fs.FolderExistError); ok {
				// 	continue
				// }
				if err.Error() == "[Filesystem][CreateFolder] A folder with the same name at the requested level already exists" {
					log.Println("[SatdataMstore] ", fmt.Sprintf("Path -> %s ::", subpath), fmt.Sprintf("%s", err))
					continue
				}
				log.Println("[SatdataMstore] Error", fmt.Sprintf("%s", err))
				return _heirarchy, err
			}

			log.Println("Subpath heirarchy created in the file sys.")
		} else {
			leafFileInfos = populateFileInfos(folder, folderMetadataFilesMap, folderBulkFilesMap, filepathMap, deadline)
			log.Println("Skipping last element of hierarchy ")
		}
	}
	subpathA := strings.Split(strings.Trim(_heirarchy, "/"), "/")
	satelliteFilePath := filepath.Dir(vod.Metadata.VideoFilename)
	log.Println("[SatdataMstore] Inserting Satellite folder in db from", satelliteFilePath)
	err := fs.CreateDownloadNewFolder(subpathA, generateSatelliteFolderIntegrity, leafFileInfos, true, satelliteFilePath)
	if err != nil {
		log.Println("[SatdataMstore] ", fmt.Sprintf("Path %s not downloaded", _heirarchy, " Error: ", err))
		return _heirarchy, err
	}
	log.Println("Printing the heirarchy created...")
	log.Println("Printing buckets")
	fs.PrintBuckets()

	log.Println("====================")
	//trigger DELETE API--- if to be removed later
	if deleteFlag {
		if err := deleteAPI(cid); err != nil {
			log.Println("[SatdataMstore] Error", fmt.Sprintf("%s", err))
		}
		log.Println("[SatdataMstore][getMstoreFiles] Info ", fmt.Sprintf("Downloaded to HUB. Deleted from Mstore Db: %s", _heirarchy))
	}
	path, _ = fs.GetActualPathForAbstractedPath(_heirarchy)
	sm2 := l.AssetInfo{AssetId: vod.Metadata.UserDefined.MediaId, Size: getDirSizeinMB(path), RelativeLocation: _heirarchy}
	sm := l.MessageSubType{AssetInfo: sm2}
	// sm.AssetInfo.AssetId = vod.Metadata.UserDefined.MediaId
	// sm.AssetInfo.Size = getDirSizeinMB(path)
	// sm.AssetInfo.RelativeLocation = _heirarchy
	logger.Log("AssetDownloadOnDeviceFromSES", &sm)
	//logger.Log("Telemetry", "ContentSyncInfo", map[string]string{"DownloadStatus": "SUCCESS", "FolderPath": _heirarchy, "AssetSize(MB)": fmt.Sprintf("%.2f", getDirSizeinMB(path)), "Channel": "SES"})
	// logger.Log("Telemetry", "HubStorage", map[string]string{"AvailableDiskSpace(MB)": getDiskInfo()})
	sm = l.MessageSubType{FloatValue: getDiskInfo()}
	//sm.FloatValue = getDiskInfo()
	logger.Log("HubStorage", &sm)
	log.Println(fmt.Sprintf("[Satdata_mstore][getMstoreFiles] Heirarchy: %s synced via SES", _heirarchy))
	log.Println(fmt.Sprintf("[Satdata_mstore][getMstoreFiles] AssetSize: %f MB", getDirSizeinMB(path)))
	log.Println(fmt.Sprintf("[Satdata_mstore][getMstoreFiles] Disk space available on the Hub: %s", getDiskInfo()))
	return _heirarchy, nil
}

func copyFiles(filePath string, fileInfos [][]string) error {
	for i, x := range fileInfos {
		log.Println("CopyFiles: ", x)
		if i == len(fileInfos)-1 {
			break
		}
		var downloadpath string
		switch x[3] {
		case "metadata":
			downloadpath = filepath.Join(filePath, cfg.Section("DEVICE_INFO").Key("METADATA_FOLDER").String(), x[0])
		case "bulkfile":
			downloadpath = filepath.Join(filePath, cfg.Section("DEVICE_INFO").Key("BULKFILE_FOLDER").String(), x[0])
		default:
			log.Println("Invalid File type: ", x[0])
			continue
		}
		log.Println(downloadpath)
		if err := os.MkdirAll(filepath.Dir(downloadpath), 0700); err != nil {
			return err
		}
		err := copySingleFile(downloadpath, x[1])
		if err != nil {
			return err
		}
		err = matchSHA256(downloadpath, x[2])
		if err != nil {
			return errors.New(fmt.Sprintf("Hashsum did not match: %s", err.Error()))
		}
		//store it in a file
		if err := storeHashsum(filepath.Dir(downloadpath), filepath.Base(downloadpath), x[2]); err != nil {
			return errors.New(fmt.Sprintf("Could not store Hashsum in the text file: %s", err))
		}
	}
	// store the deadline for the created folder
	//handled if no files to be downloaded-- only folder created and deadline.txt
	if err := storeDeadline(filePath, fileInfos[0][4]); err != nil {
		return errors.New(fmt.Sprintf("Could not store validity end date: %s", err))
	}
	return nil
}

func generateSatelliteFolderIntegrity(folderPath string, fileInfos [][]string) error {
	if len(fileInfos) == 0 {
		return errors.New(fmt.Sprintf("FileInfos length is 0 for folder ", folderPath, " in generateSatelliteFolder"))
	}
	for i, x := range fileInfos {
		if i == len(fileInfos)-1 {
			break
		}
		log.Println("generateSatelliteFolderIntegrity: ", x)
		err := matchSHA256(x[1], x[2])
		if err != nil {
			return errors.New(fmt.Sprintf("Hashsum did not match: %s for file: %s value: %s", err.Error(), x[1], x[2]))
		}

		//store it in a file
		if err := storeHashsum(folderPath, filepath.Base(x[1]), x[2]); err != nil {
			return errors.New(fmt.Sprintf("Could not store Hashsum in the text file: %s", err))
		}
	}
	if err := storeDeadline(folderPath, fileInfos[0][4]); err != nil {
		return errors.New(fmt.Sprintf("Could not store validity end date: %s", err))
	}
	return nil
}

func copySingleFile(dest, source string) error {
	t1 := time.Now()
	log.Println("Copy single file: ", " Src: ", source, " Dest: ", dest)
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
	t2 := time.Now()
	diff := t2.Sub(t1)
	log.Println("========Copy speed==(MBps)====== ", float64(written/1024/1024)/diff.Seconds())
	return nil
}
func deleteAPI(cid string) error {
	query := "http://localhost:8134/deletecontent/" + cid
	res, err := http.Get(query)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	str := string(response)
	r := regexp.MustCompile(`(?s)<body>(.*)</body>`)
	result := r.FindStringSubmatch(str)
	status := strings.Fields(strings.Trim(result[1], "\n"))
	if status[2] == "OK" {
		log.Println(" [Satdata_mstore][deleteAPI] deleted from mstore: ", cid)
	} else {
		return errors.New("Could not Delete from Mstore db")
	}
	return nil
}
func testMstore() error {
	fmt.Println("Printing buckets")
	fs.PrintBuckets()
	fmt.Println("TEST file")
	file, err := os.Open("test/mstore_test.json")
	if err != nil {
		log.Println("ERROR::::::::", err)
		return err
	}
	defer file.Close()
	bytevalue, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("ERROR::::::::", err)
		return err
	}
	var vod VodInfo
	if err = json.Unmarshal(bytevalue, &vod); err != nil {
		log.Println("ERROR::::::::", err)
		return err
	}
	//fmt.Println("Status.......", vod)
	if vod.Status == "15" {
		return errors.New("No metadata available")
	}
	if _, err = getMstoreFiles(vod); err != nil {
		return err
	}
	fmt.Println("Printing buckets")
	fs.PrintBuckets()
	return nil
}
