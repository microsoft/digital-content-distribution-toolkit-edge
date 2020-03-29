package main

import (
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
)

const databaseFileName string = "bine.db"

// Create table 1:
// [ID|MediaHouse]|Parent|hasChildren|InfoMetadataFileName
const folderStructureBucketName string = "folderStructure"

// Create table 2: Metadata files
// [ID|MediaHouse]|FileName
const metadataFilesBucketName string = "metadataFiles"

// Create table 3: Download bulk files
// [ID|MediaHouse]|FileName
const bulkFilesBucketName string = "bulkFiles"

var db *bolt.DB

// Some notes on bolt DB
// Bolt is a pure Go key/value store which is simple, fast and reliable.
// it just needs a go environment and file system to run
// More on it here - https://github.com/boltdb/bolt
// db.Update is serialized (i.e only one of these can happen at a time)
// db.View is concurrent (i.e multiple of these will be executed at a given time)

func getByteString(value string) []byte {
	return []byte(value)
}

// Returns the map associated with the current media house in the root map reffered by rootBucketName
func getMediaHouseBucket(rootBucketName string, mediaHouse string, tx *bolt.Tx) *bolt.Bucket {
	tx.CreateBucketIfNotExists(getByteString(rootBucketName))
	folderStructureBucket := tx.Bucket(getByteString(rootBucketName))

	folderStructureBucket.CreateBucketIfNotExists(getByteString(mediaHouse))
	mediaHouseBucket := folderStructureBucket.Bucket(getByteString(mediaHouse))
	return mediaHouseBucket
}

// Puts the passed struct int to the map `bucket` after serializing it to JSON
func putEncodedJSON(bucket *bolt.Bucket, v interface{}, key string) {
	if encoded, err := json.Marshal(v); err != nil {
		fmt.Println("Could not encode into json")
		return
	} else if err := bucket.Put(getByteString(key), encoded); err == nil {
		fmt.Println("Successfully added JSON: ", string(encoded))
	} else {
		fmt.Println("Could not add encoded entry: ", encoded)
	}
}

// Adds file entries into the map specified by `bucketName` and `mediaHouse`
func putFileEntries(bucketName string, tx *bolt.Tx, mediaHouse string, id string, fileEntries []FileEntry) {
	mediaHouseBucket := getMediaHouseBucket(bucketName, mediaHouse, tx)
	putEncodedJSON(mediaHouseBucket, fileEntries, id)
}

// Adds folder entry into the mediaHouse and as a child of the parent
func addFolder(tx *bolt.Tx, mediaHouse string, parent string, folderStructureEntry FolderStructureEntry) {
	mediaHouseBucket := getMediaHouseBucket(folderStructureBucketName, mediaHouse, tx)
	mediaHouseBucket.CreateBucketIfNotExists(getByteString(parent))
	parentBucket := mediaHouseBucket.Bucket(getByteString(parent))
	putEncodedJSON(parentBucket, folderStructureEntry, folderStructureEntry.ID)
}

// Adds the given list of files as metadata files of the folder `id` in mediahouse
func addMetadataFiles(tx *bolt.Tx, mediaHouse string, id string, fileEntries []FileEntry) {
	putFileEntries(metadataFilesBucketName, tx, mediaHouse, id, fileEntries)
}

// Adds the given list of files as bulk files (e.g. video and audio) of the folder `id` in mediahouse
func addBulkFiles(tx *bolt.Tx, mediaHouse string, id string, fileEntries []FileEntry) {
	putFileEntries(bulkFilesBucketName, tx, mediaHouse, id, fileEntries)
}

// get children of folder specified at `parent`
func getChildrenEntries(mediaHouse string, parent string) []FolderStructureEntry {
	var result []FolderStructureEntry
	db.View(func(tx *bolt.Tx) error {
		mediaHouseBucket := getMediaHouseBucket(folderStructureBucketName, mediaHouse, tx)
		parentBucket := mediaHouseBucket.Bucket(getByteString(parent))
		if parentBucket == nil {
			return nil
		}
		cursor := parentBucket.Cursor()
		// fmt.Println("==============Children==============")
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			// fmt.Printf("key=%s, value=%s\n", k, v)
			target := new(FolderStructureEntry)
			json.Unmarshal(v, target)
			result = append(result, *target)
		}
		// fmt.Println("====================================")
		return nil
	})
	return result
}

// get metadata files' information of the folder with id in mediaHouse
func getMetadataFileEntries(mediaHouse string, id string) []FileEntry {
	var result []FileEntry
	db.View(func(tx *bolt.Tx) error {
		mediaHouseBucket := getMediaHouseBucket(metadataFilesBucketName, mediaHouse, tx)
		files := mediaHouseBucket.Get(getByteString(id))
		json.Unmarshal(files, &result)
		return nil
	})
	return result
}

// get bulk files' information of the folder with id in mediaHouse
func getBulkFileEntries(mediaHouse string, id string) []FileEntry {
	var result []FileEntry
	db.View(func(tx *bolt.Tx) error {
		mediaHouseBucket := getMediaHouseBucket(bulkFilesBucketName, mediaHouse, tx)
		files := mediaHouseBucket.Get(getByteString(id))
		json.Unmarshal(files, &result)
		return nil
	})
	return result
}

// Creates a database connection and stores it in a variable to be used throught the lifetime of this server
func createDatabaseConnection() bool {
	tempDb, err := bolt.Open(databaseFileName, 0600, nil)
	if err != nil {
		fmt.Println("Could not open database connection")
		return false
	}
	db = tempDb
	return true
}

func setupDbForTesting() {
	// Video under test folders are taken from - https://www.pexels.com/video/view-of-the-city-at-dusk-1826904/
	// Copyright free video

	db, err := bolt.Open(databaseFileName, 0600, nil)
	if err != nil {
		fmt.Println("Could not open DB")
		panic("Could not open database, no point running the hub :/")
	}
	defer db.Close()

	// create entries for testing
	db.Update(func(tx *bolt.Tx) error {
		// Create directory strucutre
		addFolder(tx, "MSR", "root", FolderStructureEntry{"FRIENDS", true, "metadata.xml"})
		addFolder(tx, "MSR", "FRIENDS", FolderStructureEntry{"FS01", true, "metadata.xml"})
		addFolder(tx, "MSR", "FRIENDS", FolderStructureEntry{"FS02", true, "metadata.xml"})
		addFolder(tx, "MSR", "FS01", FolderStructureEntry{"FS01E01", false, "metadata.xml"})
		addFolder(tx, "MSR", "FS02", FolderStructureEntry{"FS02E01", false, "metadata.xml"})

		// Create MetadataFile entries
		addMetadataFiles(tx, "MSR", "FRIENDS", []FileEntry{FileEntry{Name: "cover.jpg", HashSum: "7440c1695f713cfc40e02c1f7f246cad718a0377d759fd298f7aed351edd4b3a"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})
		addMetadataFiles(tx, "MSR", "FS01", []FileEntry{FileEntry{Name: "cover.jpg", HashSum: "e6ad15ad2d6fc3bd26024111656b06c438faa90015c55a9a1709419ab4711c43"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})

		addMetadataFiles(tx, "MSR", "FS02", []FileEntry{FileEntry{Name: "cover.jpg", HashSum: "87aaf4791fb76c812bca62caebcf66568150541b2c323e173b3a49be3c4cb857"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})

		addMetadataFiles(tx, "MSR", "FS01E01", []FileEntry{FileEntry{Name: "cover.png", HashSum: "e02d7821816870ee0e58918e35d7f8916144d5359c4b9c02ca5e1b756f87899f"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})

		addMetadataFiles(tx, "MSR", "FS02E01", []FileEntry{FileEntry{Name: "cover.jpg", HashSum: "b1f2a066e355ba7be9c01a3afe1843148f50c1868e650ac5290664a580c344cd"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})

		// Create Bulk file entries
		addBulkFiles(tx, "MSR", "FS01E01", []FileEntry{FileEntry{Name: "vod.mp4", HashSum: "e754ddcf735de06ac797915b814c933771276831b1d802022036036e0edd4294"}})

		addBulkFiles(tx, "MSR", "FS02E01", []FileEntry{FileEntry{Name: "vod.mp4", HashSum: "e754ddcf735de06ac797915b814c933771276831b1d802022036036e0edd4294"}})

		return nil
	})
}

// Directory structure
// static/MediaHouse/ID/file

//FolderStructureEntry Folder information
type FolderStructureEntry struct {
	ID                   string
	HasChildren          bool
	InfoMetadataFileName string
}

//FileEntry File Information
type FileEntry struct {
	Name    string
	HashSum string
}
