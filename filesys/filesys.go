// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

// file system implementation with actual flat hierarchy and true hierarchy stored in bolt db
package filesys

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	bolt "github.com/boltdb/bolt"
	"github.com/google/uuid"
)

func RandStringBytes(homeNode []byte) []byte {
	b := homeNode
	for bytes.Equal(b, homeNode) {
		uuid_string := uuid.New().String()
		b = []byte(uuid_string)
	}

	return b
}

func stringInSlice(list []string, a string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func nodeToString(node []byte) string {
	return string(node)
}

type downloadFunc func(string, [][]string) error
type process_child_func func(string) ([]interface{}, error)

type FileSystem struct {
	nodeLength      int
	homeNode        []byte
	homeDirLocation string
	nodesDB         *bolt.DB
}
type ContentInfoOnDevice struct {
	DownloadTime time.Time
	FolderPath   string
	CommandId    string
}

func MakeFileSystem(homeDirLocation string, boltdb_location string) (*FileSystem, error) {
	nodesDB, err := bolt.Open(boltdb_location, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("[Filesystem][MakeFileSystem]: %e", err)
	}
	nodeLength := 36
	fs := FileSystem{nodeLength, []byte(strings.Repeat("z", nodeLength)), homeDirLocation, nodesDB}

	return &fs, nil
}

func (fs *FileSystem) InitFileSystem() error {

	/* Flat indexing of the contents-
	   AssetPathMapping
	   SatelliteIdtoAssetIdMapping
	*/
	err := fs.CreateBucket("AssetPathMapping")
	if err != nil {
		return err
	}
	err = fs.CreateBucket("SatelliteIdtoAssetIdMapping")
	if err != nil {
		return err
	}

	err = fs.CreateBucket("PendingAPIRequestMapping")
	if err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) Close() {
	fs.nodesDB.Close()
}

func (fs *FileSystem) CreateBucket(bucket_name string) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket_name))
		if err != nil {
			return fmt.Errorf("[Filesystem][CreateBucket] %s", err)
		}
		return nil
	})

	return err
}

func (fs *FileSystem) CreateHomeFolder() error {
	err := os.MkdirAll(filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode)), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (fs *FileSystem) DoesContentIdExist(id string) (bool, error) {
	found := false
	err := fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("AssetPathMapping"))
		value := b.Get([]byte(id))
		if value != nil {
			found = true
		}
		return nil
	})
	return found, err
}
func (fs *FileSystem) GetAssetFolderPathFromDB(id string) ([]byte, error) {
	var info []byte
	err := fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("AssetPathMapping")) // asset id to {downloadTime, path}
		info = b.Get([]byte(id))
		if info == nil {
			return fmt.Errorf("folder info for %s does not exist", id)
		}
		return nil
	})
	//containerStorage := cfg.Section("DEVICE_INFO").Key("MSTORE_CONTAINER_STORAGE").String()
	//containerpathString := strings.ReplaceAll(string(path), "/mnt/hdd_1/mstore/QCAST.ipts", "/mstore")
	// [a,b,c,d] -- [a,b,e,f]
	return info, err
}
func (fs *FileSystem) GetBroadcastCommandId(contentId string) (string, error) {
	var infoByte []byte
	err := fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("AssetPathMapping")) // asset id to {downloadTime, path}
		infoByte = b.Get([]byte(contentId))
		if infoByte == nil {
			return fmt.Errorf("Error in getting commandId for contentID: ", contentId)
		}
		return nil
	})
	var info ContentInfoOnDevice
	err = json.Unmarshal(infoByte, &info)
	return info.CommandId, err
}
func (fs *FileSystem) GetContentIdFromSatelliteId(satelliteId string) (string, error) {
	var infoByte []byte
	err := fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("SatelliteIdtoAssetIdMapping")) // asset id to {downloadTime, path}
		infoByte = b.Get([]byte(satelliteId))
		if infoByte == nil {
			return fmt.Errorf("Error in getting contentID for sateliteid: ", satelliteId)
		}
		return nil
	})
	return string(infoByte), err
}

func (fs *FileSystem) CreateSatelliteIndexing(cid, assetId string, contentInfo []byte) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("SatelliteIdtoAssetIdMapping"))
		fmt.Println("Creating entry for Satellite Asset- CID : ", cid, " AssetID: ", assetId)
		if err := b.Put([]byte(cid), []byte(assetId)); err != nil {
			return fmt.Errorf("[Filesystem][CreateSatelliteIndexing] %s", err)
		}
		b = tx.Bucket([]byte("AssetPathMapping"))
		//fmt.Println("Mapping Satellite Asset- AssetID: ", assetId, " to folderpath :", pathToAsset)
		if err := b.Put([]byte(assetId), contentInfo); err != nil {
			return fmt.Errorf("[Filesystem][CreateSatelliteIndexing] %s", err)
		}
		return nil
	})
	return err
}
func (fs *FileSystem) DownloadAndCreateIndexing(assetId string, dfunc downloadFunc, downloadParams [][]string) error {
	node := RandStringBytes(fs.homeNode)
	path := filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(node))
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	err = dfunc(path, downloadParams)
	if err != nil {
		ferr := os.RemoveAll(path)
		if ferr != nil {
			return ferr
		}
		return err
	}
	// Index in the bucket
	err = fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("AssetPathMapping"))
		fmt.Println("Mapping Satellite Asset- AssetID: ", assetId, " to folderpath :", path)
		if err := b.Put([]byte(assetId), []byte(path)); err != nil {
			return fmt.Errorf("[Filesystem][DownloadAndCreateIndexing] %s", err)
		}
		return nil
	})
	return nil
}

func (fs *FileSystem) GetHomeFolder() string {
	return filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode))
}
func (fs *FileSystem) GetHomeNode() []byte {
	return fs.homeNode
}
func (fs *FileSystem) GetHomeDirLocation() string {
	return fs.homeDirLocation
}
func (fs *FileSystem) GetNodeLength() int {
	return fs.nodeLength
}

func (fs *FileSystem) PrintBuckets() {
	fs.nodesDB.View(func(tx *bolt.Tx) error {

		fmt.Println("--------------------")

		fmt.Println("----------- AssetID - FolderPath Mapping--------------")

		b := tx.Bucket([]byte("AssetPathMapping"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}

		fmt.Println()
		fmt.Println("----------- SatID - AssetID Mapping--------------")

		b = tx.Bucket([]byte("SatelliteIdtoAssetIdMapping"))

		c = b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}

		fmt.Println()
		fmt.Println("----------- Pending API calls bucket --------------")

		b = tx.Bucket([]byte("PendingAPIRequestMapping"))

		c = b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}

		return nil
	})
}

func (fs *FileSystem) GetSatelliteMappedItems() (map[string]string, error) {
	satAssetMap := make(map[string]string)

	fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("SatelliteIdtoAssetIdMapping"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			satAssetMap[string(k)] = string(v)
		}

		return nil
	})

	return satAssetMap, nil
}

func (fs *FileSystem) DeleteSatelliteIds(deleteIds []string) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("SatelliteIdtoAssetIdMapping"))
		fmt.Printf("Deleting entry for Satellite Asset- satellite ids : %v ", deleteIds)

		for _, item := range deleteIds {
			if err := b.Delete([]byte(item)); err != nil {
				return fmt.Errorf("[Filesystem][DeleteSatelliteIds] %s", err)
			}
		}
		return nil
	})

	return err
}

func (fs *FileSystem) DeleteAssetMapping(deleteIds []string) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("AssetPathMapping"))
		fmt.Printf("Deleting entry for asset path mapping- content ids : %v ", deleteIds)

		for _, item := range deleteIds {
			if err := b.Delete([]byte(item)); err != nil {
				return fmt.Errorf("[Filesystem][DeleteAssetMapping] %s", err)
			}
		}
		return nil
	})

	return err
}

func (fs *FileSystem) GetPendingApiRequests() [][]byte {

	var result [][]byte
	fs.nodesDB.View(func(tx *bolt.Tx) error {
		pendingAPIRequestBucket := tx.Bucket([]byte("PendingAPIRequestMapping"))

		if pendingAPIRequestBucket == nil {
			return nil
		}

		cursor := pendingAPIRequestBucket.Cursor()

		log.Println("===========Iterating over pending api request bucket ==============")
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			log.Printf("key=%s, value=%s\n", k, v)

			result = append(result, v)

		}
		log.Println("====================================")
		return nil
	})
	return result
}

func (fs *FileSystem) AddProvisionedStatus(deviceId string, data []byte) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("PendingAPIRequestMapping"))
		err := b.Put([]byte(deviceId), data)
		if err != nil {
			return fmt.Errorf("[AddProvisionedStatus] %s", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
func (fs *FileSystem) AddCommandStatus(commandId string, data []byte) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("PendingAPIRequestMapping"))
		err := b.Put([]byte(commandId), data)
		if err != nil {
			return fmt.Errorf("[AddCommandStatus] %s", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
func (fs *FileSystem) AddContents(contentIds []string, data [][]byte) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("PendingAPIRequestMapping"))

		for i := 0; i < len(contentIds); i++ {
			err := b.Put([]byte(contentIds[i]), data[i])
			if err != nil {
				log.Printf("[AddContents] Error in insertion of item %v with error %s", contentIds[i], err)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) AddContent(contentId string, data []byte) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("PendingAPIRequestMapping"))

		err := b.Put([]byte(contentId), data)
		if err != nil {
			log.Printf("[AddContents] Error in insertion of item %v with error %s", contentId, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) DeletePendingAPIRequestEntries(data []string) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("PendingAPIRequestMapping"))

		for _, value := range data {

			log.Printf("Deleting db entry with key %v", value)
			if err := b.Delete([]byte(value)); err != nil {
				return fmt.Errorf("[PendingAPIRequestMapping][DeleteRequest] %s", err)
			} else {
				log.Printf("[DeletePendingAPIRequestEntries] Deleted entry %v", value)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("[PendingAPIRequestMapping][DeleteRequest] Error in transaction %s", err)
		return err
	}

	return nil
}
func (fs *FileSystem) GetAssetInfoMapItems() (map[string][]byte, error) {
	// CHANGE
	AssetInfoMap := make(map[string][]byte)

	fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("AssetPathMapping"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			AssetInfoMap[string(k)] = v
		}

		return nil
	})

	return AssetInfoMap, nil
}

/***********Old Code- Methods pertaining to the older logic of maintaining the filesystem heirarchy on the device. Not in use now. *************************************************************/

func (fs *FileSystem) CreateHome() error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		if b.Get(fs.homeNode) == nil {
			fmt.Println("Home noe is empty in Tree, therefore creating new")
			if err := b.Put(fs.homeNode, fs.homeNode); err != nil {
				return fmt.Errorf("[Filesystem][CreateHome] %s", err)
			}
		}

		b = tx.Bucket([]byte("FolderNameMapping"))

		if b.Get(fs.homeNode) == nil {
			fmt.Println("Home node is nil, therefore creating new")
			if err := b.Put(fs.homeNode, []byte("home")); err != nil {
				return fmt.Errorf("[Filesystem][CreateHome] %s", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return err
}

func (fs *FileSystem) InsertNode(node []byte, parent []byte) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		if err := b.Put(node, parent); err != nil {
			return fmt.Errorf("[Filesystem][InsertNode] %s", err)
		}

		children := b.Get(parent)
		if children == nil {
			return fmt.Errorf("[Filesystem][InsertNode] parent node %s doesn't exist in Tree Bucket", nodeToString(parent))
		}
		if err := b.Put(parent, append(children, node...)); err != nil {
			return fmt.Errorf("[Filesystem][InsertNode] %s", err)
		}

		return nil
	})

	return err
}
func (fs *FileSystem) GetChildrenInfo(path string, pfunc process_child_func) ([][]interface{}, error) {
	hierarchy := strings.Split(strings.Trim(path, "/"), "/")
	fmt.Println("trying to find hierarchy", strings.Trim(path, "/"))
	node, err := fs.getNodeForPath(hierarchy)
	if err != nil {
		return nil, err
	}
	fmt.Println("String node: ", nodeToString(node))
	children, err := fs.GetChildrenForNode(node)
	if err != nil {
		return nil, err
	}

	children_info := make([][]interface{}, (len(children)/fs.nodeLength)-1)
	for i := 0; i < len(children_info); i += 1 {
		child := children[(i+1)*fs.nodeLength : (i+2)*fs.nodeLength]
		actual_path := filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(child))

		children_info[i] = make([]interface{}, 0)
		folder_name, err := fs.GetFolderNameForNode(child)
		children_info[i] = append(children_info[i], folder_name)

		child_ret, err := pfunc(actual_path)
		if err != nil {
			return nil, err
		}
		children_info[i] = append(children_info[i], child_ret...)
	}

	return children_info, nil
}

func (fs *FileSystem) IsLeaf(actual_path_folder string) (bool, error) {
	if _, err := os.Stat(actual_path_folder); os.IsNotExist(err) {
		return false, fmt.Errorf("[Filesystem][IsLeaf] %s", err)
	}

	hierarchy := strings.Split(strings.Trim(actual_path_folder, "/"), "/")
	children, err := fs.GetChildrenForNode([]byte(hierarchy[len(hierarchy)-1]))
	if err != nil {
		return false, err
	}

	return len(children) == fs.nodeLength, nil
}

func (fs *FileSystem) preOrderTraversal(root []byte, prefix string) []string {
	folder_name, _ := fs.GetFolderNameForNode(root)

	children, _ := fs.GetChildrenForNode(root)

	if len(children) == fs.nodeLength {
		return []string{prefix + "/" + folder_name}
	}

	ans := []string{}
	for i := 0; i < len(children); i += fs.nodeLength {
		if i == 0 {
			continue
		}
		ans = append(ans, fs.preOrderTraversal(children[i:i+fs.nodeLength], prefix+"/"+folder_name)...)
	}

	return ans
}

func (fs *FileSystem) GetLeavesList(actual_path string) ([]string, error) {
	var startingNode []byte
	var err error = nil
	if strings.Trim(actual_path, "/") == "" {
		startingNode = fs.homeNode
	} else {
		hierarchy := strings.Split(strings.Trim(actual_path, "/"), "/")
		startingNode, err = fs.getNodeForPath(hierarchy)
	}

	if err != nil {
		return nil, err
	}

	ans := fs.preOrderTraversal(startingNode, "")

	if strings.Trim(actual_path, "/") == "" { //when giving all leaf nodes, omit the home folder from path
		homeNodeName, _ := fs.GetFolderNameForNode(startingNode)
		for i, _ := range ans {
			ans[i] = ans[i][len(homeNodeName)+1:]
		}
	}

	ret := make([]string, 0)
	for _, x := range ans {
		if len(x) != 0 {
			ret = append(ret, x)
		}
	}

	return ret, nil
}
func (fs *FileSystem) CreateDownloadNewFolder(hierarchy []string, dfunc downloadFunc, downloadParams [][]string, isSatellite bool, satelliteFolderPath string) error {
	// check if folder creation is a valid operation
	folder_name := []byte(hierarchy[len(hierarchy)-1])
	node := RandStringBytes(fs.homeNode)
	parent, err := fs.getNodeForPath(hierarchy[0 : len(hierarchy)-1])
	if err != nil {
		return err
	}

	current_children, err := fs.getChildrenNamesForNode(parent)
	if err != nil {
		return err
	}
	if stringInSlice(current_children, nodeToString(folder_name)) {
		return fmt.Errorf("[Filesystem][CreateFolder] %s", "A folder with the same name at the requested level already exists")
	}

	// create the actual folder
	actual_path := filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(node))
	err = os.MkdirAll(actual_path, os.ModePerm)
	if err != nil {
		return err
	}

	err = dfunc(actual_path, downloadParams)

	if err != nil {
		f_err := os.RemoveAll(actual_path)
		if f_err != nil {
			return f_err
		}
		return err
	}

	// once the actual folder is created, create the folder in abstraction and mark folder satellie if true
	err = fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("FolderNameMapping"))
		if err := b.Put(node, folder_name); err != nil {
			return fmt.Errorf("[Filesystem][CreateFolder] %s", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = fs.InsertNode(node, parent)
	if err != nil {
		return err
	}
	if isSatellite {
		fmt.Println("Marking folder ", node, " as satellite with path pointing to ", satelliteFolderPath)
		err = fs.MarkFolderSatellite(node, satelliteFolderPath)
		if err != nil {
			fmt.Println("Error while marking folder ", node, " as satellite, err: ", err)
			return err
		}
	}

	return nil
}
func (fs *FileSystem) MoveFolder(source_folder string, destination_folder string) error {
	if _, err := os.Stat(source_folder); os.IsNotExist(err) {
		return fmt.Errorf("[Filesystem][MoveFolder] %s", "Source Folder path does not exist")
	}
	if _, err := os.Stat(destination_folder); os.IsNotExist(err) {
		return fmt.Errorf("[Filesystem][MoveFolder] %s", "Destination Folder path does not exist")
	}
	subdirs, err := ioutil.ReadDir(source_folder)
	if err != nil {
		return fmt.Errorf("[Filesystem][MoveFolder] %s", err)
	}
	for _, subdir := range subdirs {
		err := os.Rename(filepath.Join(source_folder, subdir.Name()), filepath.Join(destination_folder, subdir.Name()))
		if err != nil {
			return fmt.Errorf("[Filesystem][MoveFolder] %s", err)
		}
	}
	return nil
}
func (fs *FileSystem) Old_InitFileSystem() error {
	err := fs.CreateBucket("Tree")
	if err != nil {
		return err
	}
	err = fs.CreateBucket("FolderNameMapping")
	if err != nil {
		return err
	}

	err = fs.CreateBucket("SatelliteMapping")
	if err != nil {
		return err
	}

	err = fs.CreateHome()
	if err != nil {
		return err
	}
	err = fs.CreateHomeFolder()
	if err != nil {
		return err
	}
	return nil
}

func (fs *FileSystem) Old_PrintBuckets() {
	fs.nodesDB.View(func(tx *bolt.Tx) error {
		fmt.Println("---------Tree-----------")
		b := tx.Bucket([]byte("Tree"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}

		fmt.Println("----------- FolderName--------------")

		b = tx.Bucket([]byte("FolderNameMapping"))

		c = b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}

		fmt.Println()
		b = tx.Bucket([]byte("SatelliteMapping"))

		c = b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}
		fmt.Println("--------------------")

		fmt.Println()

		return nil
	})
}
func (fs *FileSystem) MoveFile(source_file_path string, destination_folder string, file_type string) error {
	hierarchy := strings.Split(strings.Trim(destination_folder, "/"), "/")
	node, err := fs.getNodeForPath(hierarchy)

	if err != nil {
		return err
	}

	new_file := file_type + "_" + filepath.Base(source_file_path)
	new_location := filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(node), new_file)
	err = os.Rename(source_file_path, new_location)
	if err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) GetFileType(file_name string) (string, error) {
	temp := strings.Split(file_name, "_")
	if len(temp) == 1 {
		return "", fmt.Errorf("[Filesystem][GetFileType] %s", "file does't have a type")
	}

	return temp[0], nil
}

func (fs *FileSystem) GetActualPathForAbstractedPath(path string) (string, error) {
	hierarchy := strings.Split(strings.Trim(path, "/"), "/")

	node, err := fs.getNodeForPath(hierarchy)
	if err != nil {
		return "", err
	}
	fmt.Println("Node path is ", nodeToString(node))
	// Check if node path is in SES map
	var actual_path string
	if fs.IsSatelliteFolder(nodeToString(node)) {
		actual_path, err = fs.GetSatelliteFolderPath(nodeToString(node))
		if err != nil {
			fmt.Println("Could not find satellite folder path for ", path)
			return "", err
		}
		fmt.Println("Satellite folder- Done getting satellite path")
	} else {
		// if yes, return that file path
		actual_path = filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(node))
	}
	fmt.Println("Actual path is ", actual_path)
	return actual_path, nil
}

func (fs *FileSystem) MarkFolderSatellite(node []byte, satelliteFolderPath string) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("SatelliteMapping"))
		fmt.Println("Marking node ", nodeToString(node), " as satellite")
		if err := b.Put(node, []byte(satelliteFolderPath)); err != nil {
			return fmt.Errorf("[Filesystem][MarkFolderSatellite] %s", err)
		}
		return nil
	})
	return err
}

func (fs *FileSystem) IsSatelliteFolder(node string) bool {
	_, err := fs.GetSatelliteFolderPath(node)
	return err == nil
}

func (fs *FileSystem) IsSatelliteLeaf(abstractPath string) bool {
	hierarchy := strings.Split(strings.Trim(abstractPath, "/"), "/")
	node, err := fs.getNodeForPath(hierarchy)
	if err != nil {
		return false
	}
	_, err = fs.GetSatelliteFolderPath(nodeToString(node))
	return err == nil
}

func (fs *FileSystem) GetSatelliteFolderPath(node string) (string, error) {
	var path []byte
	err := fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("SatelliteMapping"))
		path = b.Get([]byte(node))
		if path == nil {
			return fmt.Errorf("Satellite folder %s does not exist", node)
		}
		return nil
	})
	return string(path), err
}
func (fs *FileSystem) GetFolderNameForNode(node []byte) (string, error) {
	var folder_name string
	err := fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("FolderNameMapping"))

		_temp := b.Get(node)
		if _temp == nil {
			return fmt.Errorf("[Filesystem][GetFolderNameForNode] can't find node %s in FolderNameMapping Bucket", nodeToString(_temp))
		}
		folder_name = nodeToString(_temp)
		return nil
	})

	if err != nil {
		return "", err
	}

	return folder_name, nil
}

func (fs *FileSystem) GetChildrenForNode(root []byte) ([]byte, error) {
	var children []byte
	err := fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		_temp := b.Get(root)
		if _temp == nil {
			return fmt.Errorf("[Filesystem][GetChildrenForNode] can't find node %s in Tree Bucket. Empty node", nodeToString(_temp))
		}
		children = append([]byte{}, _temp...)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return children, nil
}

func (fs *FileSystem) getNodeForPath(hierarchy []string) ([]byte, error) {
	root := fs.homeNode
	for _, folder := range hierarchy {
		children, err := fs.GetChildrenForNode(root)
		if err != nil {
			return nil, err
		}

		found := false
		for i := 0; i < len(children); i += fs.nodeLength {
			if i == 0 {
				continue
			}

			_folder, err := fs.GetFolderNameForNode(children[i : i+fs.nodeLength])
			if err != nil {
				return nil, err
			}
			if _folder == folder {
				root = children[i : i+fs.nodeLength]
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("[Filesystem][GetFolderNameForNode] can't find the node for folder %s in the hierarchy", folder)
		}
	}

	return root, nil
}

func (fs *FileSystem) getChildrenNamesForNode(parent []byte) ([]string, error) {
	children, err := fs.GetChildrenForNode(parent)
	if err != nil {
		return nil, err
	}

	ans := make([]string, 0)
	for i := 0; i < len(children); i += fs.nodeLength {
		if i == 0 {
			continue
		}

		child, err := fs.GetFolderNameForNode(children[i : i+fs.nodeLength])
		if err != nil {
			return nil, err
		}

		ans = append(ans, child)
	}

	return ans, nil
}

func (fs *FileSystem) CreateFolder(hierarchy []string) (string, error) {
	folder_name := []byte(hierarchy[len(hierarchy)-1])
	node := RandStringBytes(fs.homeNode)
	parent, err := fs.getNodeForPath(hierarchy[0 : len(hierarchy)-1])
	if err != nil {
		return "", err
	}

	current_children, err := fs.getChildrenNamesForNode(parent)
	if err != nil {
		return "", err
	}
	if stringInSlice(current_children, nodeToString(folder_name)) {
		return "", fmt.Errorf("[Filesystem][CreateFolder] %s", "A folder with the same name at the requested level already exists")
	}

	err = fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("FolderNameMapping"))
		if err := b.Put(node, folder_name); err != nil {
			return fmt.Errorf("[Filesystem][CreateFolder] %s", err)
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	err = fs.InsertNode(node, parent)
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(node)), os.ModePerm)
	if err != nil {
		return "", err
	}

	return nodeToString(node), nil
}
func (fs *FileSystem) recursivelyPrintNode(root []byte, level int) {
	folder_name, _ := fs.GetFolderNameForNode(root)
	fmt.Println(strings.Repeat("\t", level) + folder_name)

	children, _ := fs.GetChildrenForNode(root)

	for i := 0; i < len(children); i += fs.nodeLength {
		if i == 0 {
			continue
		}
		fs.recursivelyPrintNode(children[i:i+fs.nodeLength], level+1)
	}
}

func (fs *FileSystem) PrintFileSystem() {
	fs.recursivelyPrintNode(fs.homeNode, 0)
}

func (fs *FileSystem) DeleteNodeSubtree(node []byte) error {
	children, err := fs.GetChildrenForNode(node)
	if err != nil {
		return err
	}
	for i := 0; i < len(children); i += fs.nodeLength {
		if i == 0 {
			continue
		} else {
			fs.DeleteNodeSubtree(children[i : i+fs.nodeLength])
		}
	}

	err = fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		if err := b.Delete(node); err != nil {
			return fmt.Errorf("[Filesystem][DeleteNodeSubtree] %s", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("FolderNameMapping"))
		if err := b.Delete(node); err != nil {
			return fmt.Errorf("[Filesystem][DeleteNodeSubtree] %s", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if fs.IsSatelliteFolder(nodeToString(node)) {
		satelliteFolderPath, _ := fs.GetSatelliteFolderPath(nodeToString(node))
		if err != nil {
			fmt.Println("Could not find satellite folder path to delete for ", nodeToString(node))
			return err
		}
		err = fs.nodesDB.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("SatelliteMapping"))
			if err := b.Delete(node); err != nil {
				return fmt.Errorf("[Filesystem][DeleteNodeSubtree] %s", err)
			}
			return nil
		})
		if err != nil {
			return err
		}
		fmt.Println("deleting satellite folder Path:", satelliteFolderPath)
		if _, err = os.Stat(satelliteFolderPath); os.IsNotExist(err) {
			fmt.Println("Satellite Folder location does not exist : ", satelliteFolderPath)
		} else {
			err = os.RemoveAll(satelliteFolderPath)
			if err != nil {
				return err
			}
		}

	}
	// delete from home folder

	fmt.Println("subtree path deleting....", filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(node)))
	err = os.RemoveAll(filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(node)))
	if err != nil {
		return err
	}
	return err
}

func (fs *FileSystem) DeleteFolder(hierarchy []string) error {
	node, err := fs.getNodeForPath(hierarchy)
	if err != nil {
		return err
	}
	fmt.Println("node to be deleted", nodeToString(node))

	// update the parent only for the top most node in the hierarchy
	err = fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		parent := b.Get(node)[0:fs.nodeLength]

		children := b.Get(parent)
		children_w_node_removed := []byte{}
		for i := 0; i < len(children); i += fs.nodeLength {
			if bytes.Equal(node, children[i:i+fs.nodeLength]) {
				continue
			} else {
				children_w_node_removed = append(children_w_node_removed, children[i:i+fs.nodeLength]...)
			}
		}
		if err := b.Put(parent, children_w_node_removed); err != nil {
			return fmt.Errorf("[Filesystem][DeleteNodeSubtree] %s", err)
		}

		return nil
	})
	if fs.IsSatelliteFolder(nodeToString(node)) {
		satelliteFolderPath, _ := fs.GetSatelliteFolderPath(nodeToString(node))
		if err != nil {
			fmt.Println("Could not find satellite folder path to delete for ", nodeToString(node))
			return err
		}
		err = os.RemoveAll(satelliteFolderPath)
		if err != nil {
			return err
		}
	}
	fmt.Println("path deleting....", filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(node)))
	err = os.RemoveAll(filepath.Join(fs.homeDirLocation, nodeToString(fs.homeNode), nodeToString(node)))
	if err != nil {
		fmt.Errorf(err.Error())
		return err
	}
	fmt.Println("Deleting from Tree")
	err = fs.DeleteNodeSubtree(node)
	if err != nil {
		return err
	}
	return nil
}

func (fs *FileSystem) RecursiveDeleteFolder(hierarchy []string) error {
	err := fs.DeleteFolder(hierarchy)
	if err != nil {
		return err
	}

	for i := len(hierarchy) - 2; i >= 0; i-- {
		node, err := fs.getNodeForPath(hierarchy[0 : i+1])
		if err != nil {
			return err
		}

		children, err := fs.GetChildrenForNode(node)
		if err != nil {
			return err
		}

		if len(children) == fs.nodeLength {
			if err := fs.DeleteFolder(hierarchy[0 : i+1]); err != nil {
				return err
			}
		} else {
			break
		}
	}

	return nil
}
