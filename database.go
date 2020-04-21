package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
)

const databaseFileName string = "bine.db"

// Create Map 1:
// first level key media house -> child is another map
// second level key is parent ID -> child is another MAP
// third level key is child ID and value is corresponding FolderEntry
const folderStructureBucketName string = "folderStructure"

// Create Map 2: Metadata files
// first level key media house -> child is another map
// second level key is ID -> child is list of FileEntry
const metadataFilesBucketName string = "metadataFiles"

// Create Map 3: Download bulk files
// first level key media house -> child is another map
// second level key is ID -> child is list of FileEntry
const bulkFilesBucketName string = "bulkFiles"

// Create Map 4: Child to parent map
// first level key media house -> child is another map
// second level key is ID -> child is parent ID
const childToParentBucketName string = "childToParent"

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

func safeCreateIfNotExistsInBucket(bucket *bolt.Bucket, name string) {
	if bucket != nil {
		bucket.CreateBucketIfNotExists(getByteString(name))
	}
}

func safeCreateIfNotExistsInTran(tx *bolt.Tx, name string) {
	if tx != nil {
		tx.CreateBucketIfNotExists(getByteString(name))
	}
}

// Returns the map associated with the current media house in the root map reffered by rootBucketName
func getMediaHouseBucket(rootBucketName string, mediaHouse string, tx *bolt.Tx) *bolt.Bucket {
	safeCreateIfNotExistsInTran(tx, rootBucketName)
	folderStructureBucket := tx.Bucket(getByteString(rootBucketName))
	if folderStructureBucket == nil {
		return nil
	}
	safeCreateIfNotExistsInBucket(folderStructureBucket, mediaHouse)
	mediaHouseBucket := folderStructureBucket.Bucket(getByteString(mediaHouse))
	return mediaHouseBucket
}

// Puts the passed struct int to the map `bucket` after serializing it to JSON
func putEncodedJSON(bucket *bolt.Bucket, v interface{}, key string) error {
	if encoded, err := json.Marshal(v); err != nil {
		return errors.New("Could not encode into json")
	} else if err := bucket.Put(getByteString(key), encoded); err == nil {
		fmt.Println("Successfully added JSON: ", string(encoded))
	} else {
		return errors.New("Could not add encoded entry: " + string(encoded))
	}
	return nil
}

// Adds file entries into the map specified by `bucketName` and `mediaHouse`
func putFileEntries(bucketName string, tx *bolt.Tx, mediaHouse string, id string, fileEntries []FileEntry) error {
	mediaHouseBucket := getMediaHouseBucket(bucketName, mediaHouse, tx)
	if mediaHouseBucket == nil {
		return errors.New("Media house bucket was null while adding file entries to " + bucketName)
	}
	return putEncodedJSON(mediaHouseBucket, fileEntries, id)
}

// Adds folder entry into the mediaHouse and as a child of the parent
func addFolder(mediaHouse string, parent string, folderStructureEntry FolderStructureEntry) error {
	return db.Update(func(tx *bolt.Tx) error {
		mediaHouseBucket := getMediaHouseBucket(folderStructureBucketName, mediaHouse, tx)
		if mediaHouseBucket == nil {
			return errors.New("Media house bucket associated with mediahouse value: " + mediaHouse + " in parent bucket: " + folderStructureBucketName + " was nil")
		}
		safeCreateIfNotExistsInBucket(mediaHouseBucket, parent)
		parentBucket := mediaHouseBucket.Bucket(getByteString(parent))
		errIn := putEncodedJSON(parentBucket, folderStructureEntry, folderStructureEntry.ID)
		if errIn != nil {
			return errIn
		}
		mediaHouseBucket = getMediaHouseBucket(childToParentBucketName, mediaHouse, tx)
		if mediaHouseBucket == nil {
			return errors.New("Media house bucket associated with mediahouse value: " + mediaHouse + " in parent bucket: " + childToParentBucketName + " was nil")
		}
		mediaHouseBucket.Put(getByteString(folderStructureEntry.ID), getByteString(parent))
		return nil
	})
}

// Adds the given list of files as metadata files of the folder `id` in mediahouse
func addMetadataFiles(mediaHouse string, id string, fileEntries []FileEntry) error {
	return db.Update(func(tx *bolt.Tx) error {
		return putFileEntries(metadataFilesBucketName, tx, mediaHouse, id, fileEntries)
	})
}

// Adds the given list of files as bulk files (e.g. video and audio) of the folder `id` in mediahouse
func addBulkFiles(mediaHouse string, id string, fileEntries []FileEntry) error {
	return db.Update(func(tx *bolt.Tx) error {
		return putFileEntries(bulkFilesBucketName, tx, mediaHouse, id, fileEntries)
	})
}

// get children of folder specified at `parent`
func getChildrenEntries(mediaHouse string, parent string) []FolderStructureEntry {
	var result []FolderStructureEntry
	db.View(func(tx *bolt.Tx) error {
		mediaHouseBucket := getMediaHouseBucket(folderStructureBucketName, mediaHouse, tx)
		if mediaHouseBucket == nil {
			return nil
		}
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

func getFileEntries(mediaHouse string, id string, bucketName string) []FileEntry {
	var result []FileEntry
	err := db.View(func(tx *bolt.Tx) error {
		mediaHouseBucket := getMediaHouseBucket(bucketName, mediaHouse, tx)
		if mediaHouseBucket == nil {
			return errors.New("Media house bucket : " + bucketName + " for mediahouse " + mediaHouse + " was null")
		}
		files := mediaHouseBucket.Get(getByteString(id))
		if files != nil {
			json.Unmarshal(files, &result)
		} else {
			return errors.New("could not marshall json while getting file entries for media house: " + mediaHouse + " and ID: " + id)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Could not get file entries: " + err.Error())
	}
	return result
}

// get metadata files' information of the folder with id in mediaHouse
func getMetadataFileEntries(mediaHouse string, id string) []FileEntry {
	return getFileEntries(mediaHouse, id, metadataFilesBucketName)
}

// get bulk files' information of the folder with id in mediaHouse
func getBulkFileEntries(mediaHouse string, id string) []FileEntry {
	return getFileEntries(mediaHouse, id, bulkFilesBucketName)
}

func eraseKey(bucketName string, mediaHouse string, tx *bolt.Tx, id string) error {
	mediaHouseBucket := getMediaHouseBucket(bucketName, mediaHouse, tx)
	return mediaHouseBucket.Delete(getByteString(id))
}

func doErase(id string, mediaHouse string, tx *bolt.Tx) error {
	// Remove associated file entries
	errIn := eraseKey(metadataFilesBucketName, mediaHouse, tx, id)
	if errIn != nil {
		return errIn
	}
	errIn = eraseKey(bulkFilesBucketName, mediaHouse, tx, id)
	if errIn != nil {
		return errIn
	}

	fmt.Println("Done removing files for ID: " + id)

	// remove Id from it's parents children list
	mediaHouseBucket := getMediaHouseBucket(childToParentBucketName, mediaHouse, tx)
	if mediaHouseBucket == nil {
		// return error
		return errors.New("Media house bucket as null while trying to get parent of child")
	}
	parentBytes := mediaHouseBucket.Get(getByteString(id))
	println("Parent of current ID is: " + string(parentBytes))
	if parentBytes == nil {
		return errors.New("Parent was nil for the passed ID")
	}
	mediaHouseBucket = getMediaHouseBucket(folderStructureBucketName, mediaHouse, tx)
	parentBucket := mediaHouseBucket.Bucket(parentBytes)
	errIn = parentBucket.Delete(getByteString(id))
	if errIn != nil {
		return errIn
	}
	fmt.Println("Done removing files and entry in parent's children bucket for ID: " + id)

	// remove the subtree that starts at the node of this ID of the folder structure
	folderChildren := getChildrenEntries(mediaHouse, id)
	for _, child := range folderChildren {
		errIn = doErase(child.ID, mediaHouse, tx)
		if errIn != nil {
			return errIn
		}
	}
	fmt.Println("Done removing all the children nodes of ID: " + id)
	// finally remove the node from folder structure
	mediaHouseBucket.DeleteBucket(getByteString(id))
	return nil
}

// delete metadatafiles
// delete bulk files
// delete from parents map
// delete from children list
func eraseFolder(id string, mediaHouse string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		return doErase(id, mediaHouse, tx)
	})
	return err
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

	// create entries for testing
	// Create directory strucutre
	addFolder("MSR", "root", FolderStructureEntry{"FRIENDS", true, "metadata.xml"})
	addFolder("MSR", "FRIENDS", FolderStructureEntry{"FS01", true, "metadata.xml"})
	addFolder("MSR", "FRIENDS", FolderStructureEntry{"FS02", true, "metadata.xml"})
	addFolder("MSR", "FS01", FolderStructureEntry{"FS01E01", false, "metadata.xml"})
	addFolder("MSR", "FS02", FolderStructureEntry{"FS02E01", false, "metadata.xml"})

	// Create MetadataFile entries
	addMetadataFiles("MSR", "FRIENDS", []FileEntry{FileEntry{Name: "cover.jpg", HashSum: "7440c1695f713cfc40e02c1f7f246cad718a0377d759fd298f7aed351edd4b3a"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})
	addMetadataFiles("MSR", "FS01", []FileEntry{FileEntry{Name: "cover.jpg", HashSum: "e6ad15ad2d6fc3bd26024111656b06c438faa90015c55a9a1709419ab4711c43"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})

	addMetadataFiles("MSR", "FS02", []FileEntry{FileEntry{Name: "cover.jpg", HashSum: "87aaf4791fb76c812bca62caebcf66568150541b2c323e173b3a49be3c4cb857"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})

	addMetadataFiles("MSR", "FS01E01", []FileEntry{FileEntry{Name: "cover.png", HashSum: "e02d7821816870ee0e58918e35d7f8916144d5359c4b9c02ca5e1b756f87899f"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})

	addMetadataFiles("MSR", "FS02E01", []FileEntry{FileEntry{Name: "cover.jpg", HashSum: "b1f2a066e355ba7be9c01a3afe1843148f50c1868e650ac5290664a580c344cd"}, FileEntry{Name: "metadata.xml", HashSum: "7b4bb8b01c333e7fa8145bcec20244197c7763786ef47c9de962effb017d01de"}})

	// Create Bulk file entries
	addBulkFiles("MSR", "FS01E01", []FileEntry{FileEntry{Name: "vod.mp4", HashSum: "e754ddcf735de06ac797915b814c933771276831b1d802022036036e0edd4294"}})

	addBulkFiles("MSR", "FS02E01", []FileEntry{FileEntry{Name: "vod.mp4", HashSum: "e754ddcf735de06ac797915b814c933771276831b1d802022036036e0edd4294"}})
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
