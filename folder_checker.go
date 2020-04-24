package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"
)

const checkingInterval time.Duration = 1000

func computeSHA256(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Could not open the file")
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		fmt.Println("Could not compute hash")
	}
	return hex.EncodeToString(h.Sum(nil))
}

// start navigating from root till leaves of the content tree structure in a DFS manner
// this happens every `checkingInterval` seconds
func check() {
	for true {
		time.Sleep(checkingInterval * time.Second)
		fmt.Println("Checking files' integrity from background thread")
		navigate("root")
	}
}

// Does the actual DFS
func navigate(node string) {
	// TODO: Call this for each media house on the box not just MSR
	filesToCheck := getFilesToCheck("MSR", node)
	fmt.Println("==========Checking ", node, " total files len: ", len(filesToCheck), "==========")
	for i := 0; i < len(filesToCheck); i++ {
		calculatedSHA256 := computeSHA256(filesToCheck[i].path)
		fmt.Println("Calculated SHA256: ", calculatedSHA256)
		fmt.Println("Actual SHA256: ", filesToCheck[i].sha256)

		if calculatedSHA256 == filesToCheck[i].sha256 {
			fmt.Println("No issues found in file tampering")
		} else {
			// TODO: Implement deletion process.
			fmt.Println("Get rid of this folder from the database and delete file contents")
		}
	}
	fmt.Println("==========DONE Checking ", node, " ==========")
	children := getChildren("MSR", node)
	for i := 0; i < len(children); i++ {
		fmt.Println("CALLING CHECK ON CHILD: ", children[i].ID)
		navigate(children[i].ID)
	}
	// Get children of this node and then navigate them as well
}
