// file system implementation with actual flat hierarchy and true hierarchy stored in bolt db
package filesys

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	bolt "github.com/boltdb/bolt"
)

type FolderExistError struct {
	Err error
}

func (e *FolderExistError) Error() string {
	return e.Err.Error()
}

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int, homeNode []byte) []byte {
	b := homeNode
	for bytes.Equal(b, homeNode) {
		b = make([]byte, n)
		for i := range b {
			b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
		}
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

type downloadFunc func(string, [][]string) error

type FileSystem struct {
	nodeLength      int
	homeNode        []byte
	homeDirLocation string
	nodesDB         *bolt.DB
}

func MakeFileSystem(nodeLength int, homeDirLocation string, boltdb_location string) (*FileSystem, error) {
	nodesDB, err := bolt.Open(boltdb_location, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("[Filesystem][MakeFileSystem]: %e", err)
	}

	fs := FileSystem{nodeLength, []byte(strings.Repeat("z", nodeLength)), homeDirLocation, nodesDB}

	return &fs, nil
}

func (fs *FileSystem) InitFileSystem() error {
	err := fs.CreateBucket("Tree")
	if err != nil {
		return err
	}
	err = fs.CreateBucket("FolderNameMapping")
	if err != nil {
		return err
	}

	err = fs.CreateHome()
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

func (fs *FileSystem) CreateHome() error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		if err := b.Put(fs.homeNode, fs.homeNode); err != nil {
			return fmt.Errorf("[Filesystem][CreateHome] %s", err)
		}

		b = tx.Bucket([]byte("FolderNameMapping"))

		if err := b.Put(fs.homeNode, []byte("home")); err != nil {
			return fmt.Errorf("[Filesystem][CreateHome] %s", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(fs.homeDirLocation, string(fs.homeNode)), os.ModePerm)
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
			return fmt.Errorf("[Filesystem][InsertNode] parent node %s doesn't exist in Tree Bucket", string(parent))
		}
		if err := b.Put(parent, append(children, node...)); err != nil {
			return fmt.Errorf("[Filesystem][InsertNode] %s", err)
		}

		return nil
	})

	return err
}

func (fs *FileSystem) recursivelyPrintNode(root []byte, level int) {
	folder_name, _ := fs.getFolderNameForNode(root)
	fmt.Println(strings.Repeat("\t", level) + folder_name)

	children, _ := fs.getChildrenForNode(root)

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

func (fs *FileSystem) getFolderNameForNode(node []byte) (string, error) {
	var folder_name string
	err := fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("FolderNameMapping"))

		_temp := b.Get(node)
		if _temp == nil {
			return fmt.Errorf("[Filesystem][getFolderNameForNode] can't find node %s in FolderNameMapping Bucket", string(_temp))
		}
		folder_name = string(_temp)
		return nil
	})

	if err != nil {
		return "", err
	}

	return string(folder_name), nil
}

func (fs *FileSystem) getChildrenForNode(root []byte) ([]byte, error) {
	var children []byte
	err := fs.nodesDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		_temp := b.Get(root)
		if _temp == nil {
			return fmt.Errorf("[Filesystem][getChildrenForNode] can't find node %s in Tree Bucket", string(_temp))
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
		children, err := fs.getChildrenForNode(root)
		if err != nil {
			return nil, err
		}

		found := false
		for i := 0; i < len(children); i += fs.nodeLength {
			if i == 0 {
				continue
			}

			_folder, err := fs.getFolderNameForNode(children[i : i+fs.nodeLength])
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
			return nil, fmt.Errorf("[Filesystem][getFolderNameForNode] can't find the node for folder %s in the hierarchy", folder)
		}
	}

	return root, nil
}

func (fs *FileSystem) getChildrenNamesForNode(parent []byte) ([]string, error) {
	children, err := fs.getChildrenForNode(parent)
	if err != nil {
		return nil, err
	}

	ans := make([]string, 0)
	for i := 0; i < len(children); i += fs.nodeLength {
		if i == 0 {
			continue
		}

		child, err := fs.getFolderNameForNode(children[i : i+fs.nodeLength])
		if err != nil {
			return nil, err
		}

		ans = append(ans, child)
	}

	return ans, nil
}

func (fs *FileSystem) CreateFolder(hierarchy []string) (string, error) {
	folder_name := []byte(hierarchy[len(hierarchy)-1])
	node := RandStringBytes(fs.nodeLength, fs.homeNode)
	parent, err := fs.getNodeForPath(hierarchy[0 : len(hierarchy)-1])
	if err != nil {
		return "", err
	}

	current_children, err := fs.getChildrenNamesForNode(parent)
	if err != nil {
		return "", err
	}
	if stringInSlice(current_children, string(folder_name)) {
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

	err = os.MkdirAll(filepath.Join(fs.homeDirLocation, string(fs.homeNode), string(node)), os.ModePerm)
	if err != nil {
		return "", err
	}

	return string(node), nil
}

func (fs *FileSystem) DeleteNodeSubtree(node []byte) error {
	children, err := fs.getChildrenForNode(node)
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

	return err
}

func (fs *FileSystem) DeleteFolder(hierarchy []string) error {
	node, err := fs.getNodeForPath(hierarchy)
	if err != nil {
		return err
	}
	fmt.Println("node to be deleted", string(node))

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

	fmt.Println("Deleting from Tree")
	err = fs.DeleteNodeSubtree(node)
	if err != nil {
		return err
	}

	err = os.RemoveAll(filepath.Join(fs.homeDirLocation, string(fs.homeNode), string(node)))
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

		children, err := fs.getChildrenForNode(node)
		if err != nil {
			return err
		}

		if len(children) == 4 {
			if err := fs.DeleteFolder(hierarchy[0 : i+1]); err != nil {
				return err
			}
		} else {
			break
		}
	}

	return nil
}

func (fs *FileSystem) MoveFile(source_file_path string, destination_folder string, file_type string) error {
	hierarchy := strings.Split(strings.Trim(destination_folder, "/"), "/")
	node, err := fs.getNodeForPath(hierarchy)

	if err != nil {
		return err
	}

	new_file := file_type + "_" + filepath.Base(source_file_path)
	new_location := filepath.Join(fs.homeDirLocation, string(fs.homeNode), string(node), new_file)
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
	actual_path := filepath.Join(fs.homeDirLocation, string(fs.homeNode), string(node))

	return actual_path, nil
}

func (fs *FileSystem) CreateDownloadNewFolder(hierarchy []string, dfunc downloadFunc, downloadParams [][]string) error {
	// check if folder creation is a valid operation
	folder_name := []byte(hierarchy[len(hierarchy)-1])
	node := RandStringBytes(fs.nodeLength, fs.homeNode)
	parent, err := fs.getNodeForPath(hierarchy[0 : len(hierarchy)-1])
	if err != nil {
		return err
	}

	current_children, err := fs.getChildrenNamesForNode(parent)
	if err != nil {
		return err
	}
	if stringInSlice(current_children, string(folder_name)) {
		return fmt.Errorf("[Filesystem][CreateFolder] %s", "A folder with the same name at the requested level already exists")
		//return &FolderExistError{fmt.Errorf("[Filesystem][CreateFolder] %s", "A folder with the same name at the requested level already exists")}
	}

	// create the actual folder
	actual_path := filepath.Join(fs.homeDirLocation, string(fs.homeNode), string(node))
	err = os.MkdirAll(actual_path, os.ModePerm)
	if err != nil {
		return err
	}

	// downlaod the files, check hashsum is done in dfunc
	err = dfunc(actual_path, downloadParams)
	if err != nil {
		f_err := os.RemoveAll(actual_path)
		if f_err != nil {
			return f_err
		}
		return err
	}

	// once the actual folder is created, create the folder in abstraction
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

	return nil
}

func (fs *FileSystem) GetHomeFolder() string {
	return filepath.Join(fs.homeDirLocation, string(fs.homeNode))
}
func (fs *FileSystem) GetHomeDirLocation() string {
	return fs.homeDirLocation
}
func (fs *FileSystem) PrintBuckets() {
	fs.nodesDB.View(func(tx *bolt.Tx) error {
		fmt.Println("--------------------")
		b := tx.Bucket([]byte("Tree"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}

		fmt.Println()

		b = tx.Bucket([]byte("FolderNameMapping"))

		c = b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}
		fmt.Println("--------------------")

		return nil
	})
}
