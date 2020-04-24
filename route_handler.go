package main

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const lengthOfStringBytes int = 4
const lengthOfInt32Bytes int = 4
const lengthOfInt64Bytes int = 8

// Setup Routes and their handles which the hub exposes
// Route 1: "/list/files/:parent" returns the children and their metadata of :parent
func setupRoutes(ginEngine *gin.Engine) {
	ginEngine.Static("/static", "./static")
	ginEngine.GET("/metadata/:mediaHouse/:id", serveSingleMetadata)
	ginEngine.GET("/list/files/:mediaHouse/:parent", serveMetadata)
	ginEngine.GET("/download/files/:mediaHouse/:folderID/:fileName", serveFile)
}

func serveSingleMetadata(context *gin.Context) {
	mediaHouse := context.Param("mediaHouse")
	folderID := context.Param("id")
	writeMetadataFiles(context, getMetadataFiles(mediaHouse, folderID), mediaHouse, folderID)
}

func serveFile(context *gin.Context) {
	mediaHouse := context.Param("mediaHouse")
	folderID := context.Param("folderID")
	fileName := context.Param("fileName")
	// redirect to this path
	context.Redirect(http.StatusTemporaryRedirect, "/static/"+mediaHouse+"/"+folderID+"/"+fileName)
}

//Route handler for /list/files/:parent
//Returns metadata of the requested parent's children and the actual metadata files associated with each of the child
//Batching files and children this way in a single response reduces the latency as opposed to using different HTTP request for each file
func serveMetadata(context *gin.Context) {
	parent := context.Param("parent")
	mediaHouse := context.Param("mediaHouse")
	fmt.Println("Parent is: ", parent)

	// get List of children
	// need to get this and metadata file list from Database
	children := getChildren(mediaHouse, parent)

	// return number of children
	if writeInt32(context, len(children)) < 0 {
		fmt.Println("Could not write to response stream")
		return
	}

	// loop through children
	for i := 0; i < len(children); i++ {

		// return length of name - 4 bytes
		if writeString(context, children[i].ID) < 0 {
			fmt.Println("Could not write ID of child")
			return
		}

		// TODO: consider processing infoMetadataFile and send results as a part of this reply

		// return if this child has children
		// 1 - yes, 0 - no
		// TODO: Change this to a single byte
		hasChildren := 0
		if children[i].HasChildren {
			hasChildren = 1
		}
		if writeInt32(context, hasChildren) < 0 {
			fmt.Println("Could not write hasChildren of ID: ", children[i].ID)
			return
		}

		metadataFiles := getMetadataFiles(mediaHouse, children[i].ID)
		if writeMetadataFiles(context, metadataFiles, mediaHouse, children[i].ID) < 0 {
			fmt.Println("Could not write metadata files for ID: " + children[i].ID)
		}
	}
}

func writeMetadataFiles(context *gin.Context, metadataFiles []string, mediaHouse string, id string) int {
	// return number of metadata files
	if writeInt32(context, len(metadataFiles)) < 0 {
		return -1
	}

	// loop through metadata files
	for j := 0; j < len(metadataFiles); j++ {
		// write length of name
		if writeString(context, metadataFiles[j]) < 0 {
			fmt.Println("Could not write file name: for ID ", id, " name: ", metadataFiles[j])
			return -1
		}

		// write length of file
		filePath := getFilePath(mediaHouse, id, metadataFiles[j])
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			fmt.Println("Could not get file info ", id, " name: ", metadataFiles[j])
			return -1
		}
		if writeInt64(context, fileInfo.Size()) < 0 {
			fmt.Println("Could not write file size ", id, " name: ", metadataFiles[j])
			return -1
		}

		// write the actual file data
		fileHandle, err := os.Open(filePath)
		defer fileHandle.Close()
		if err != nil {
			fmt.Println("Could not open file ", filePath)
			return -1
		}
		fmt.Println("Writing file ", filePath)
		written := writeFile(context, fileHandle, fileInfo.Size())
		fmt.Println("Done writing file ", filePath, " total bytes written: ", written, "/", fileInfo.Size(), " FOR: ", id, " name: ", metadataFiles[j])
	}
	return len(metadataFiles)
}

// Writes bytes int the method parameter 'value' to the output stream associated with the current HTTP request
// Returns -1 if the write call fails at any point
// Returns the total number of bytes written on success
func writeBytes(context *gin.Context, value []byte) int {
	written := 0
	for written < len(value) {
		tempWritten, err := context.Writer.Write(value[written:])
		if err != nil {
			return -1
		}
		written += tempWritten
	}
	return written
}

// Writes the string in method parameter 'value' to the output stream associated with the current HTTP request
// Returns -1 if the write call fails at any point
// Returns the string length on success
func writeString(context *gin.Context, value string) int {
	var tempIntBytes []byte = make([]byte, lengthOfStringBytes)
	valueInBytes := []byte(value)
	binary.BigEndian.PutUint32(tempIntBytes, uint32(len(valueInBytes)))
	// write length of string
	if writeBytes(context, tempIntBytes) < 0 {
		return -1
	}
	// write actual string
	if writeBytes(context, valueInBytes) < 0 {
		return -1
	}
	return len(valueInBytes)
}

// Writes the passed value as 32 bit integer to the output stream
// Returns -1 on failure
// Returns number of bytes written on success
func writeInt32(context *gin.Context, value int) int {
	tempIntBytes := make([]byte, lengthOfInt32Bytes)
	binary.BigEndian.PutUint32(tempIntBytes, uint32(value))
	return writeBytes(context, tempIntBytes)
}

// Writes the passed value as 64 bit integer to the output stream
// Returns -1 on failure
// Returns number of bytes written on success
func writeInt64(context *gin.Context, value int64) int {
	tempIntBytes := make([]byte, lengthOfInt64Bytes)
	binary.BigEndian.PutUint64(tempIntBytes, uint64(value))
	return writeBytes(context, tempIntBytes)
}

// Writes the passed file data as bytes to the output stream
// Returns how many ever bytes were written on failure
// Returns number of bytes written on success
func writeFile(context *gin.Context, fileHandle *os.File, fileSize int64) int64 {
	buffer := make([]byte, 512*1024)
	var written int64 = 0
	for written < fileSize {
		fileBytesRead, err := fileHandle.Read(buffer)
		if err != nil {
			fmt.Println("Could not read from file")
			return written
		}
		if writeBytes(context, buffer[:fileBytesRead]) < 0 {
			fmt.Println("Could not write file to stream")
			return written
		}
		written += int64(fileBytesRead)
	}
	return written
}
