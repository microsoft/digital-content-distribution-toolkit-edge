// file system implementation with actual flat hierarchy and true hierarchy stored in bolt db
package filesys

import (
	"fmt"
	"math/rand"
	"strings"
	"bytes"
	"os"
	"path/filepath"

	bolt "github.com/boltdb/bolt"
)

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ";

func RandStringBytes(n int, homeNode []byte) []byte {
	b := homeNode
	for bytes.Equal(b, homeNode) {
	    b = make([]byte, n)
	    for i := range b {
	        b[i] = letterBytes[rand.Int63() % int64(len(letterBytes))]
	    }
	}

    return b;
}


type FileSystem struct {
	nodeLength int
	homeNode []byte
	homeDirLocation string
	nodesDB *bolt.DB
}

func MakeFileSystem(nodeLength	int, homeDirLocation string) (*FileSystem, error) {
	nodesDB, err := bolt.Open("test.db", 0600, nil);
	if(err != nil) {
		return nil, fmt.Errorf("[Database][MakeFileSystem]: %e", err)
	}

	fs := FileSystem{nodeLength, []byte(strings.Repeat("z", nodeLength)), homeDirLocation, nodesDB}

	return &fs, nil
}

func (fs *FileSystem) Close() {
	fs.nodesDB.Close()
}

func (fs *FileSystem) CreateBucket(bucket_name string) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket_name))
		if err != nil {
			return fmt.Errorf("[Database][CreateBucket] %s", err)
		}
		return nil
	})

	return err
}

func (fs *FileSystem) CreateHome() error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		if err := b.Put(fs.homeNode, fs.homeNode); err != nil {
			return fmt.Errorf("[Database][CreateHome] %s", err)
		}

		b = tx.Bucket([]byte("FolderNameMapping"))

		if err := b.Put(fs.homeNode, []byte("home")); err != nil {
			return fmt.Errorf("[Database][CreateHome] %s", err)
		}

		return nil
	})
	if(err != nil) {
		return err
	}

	err = os.MkdirAll(filepath.Join(fs.homeDirLocation, string(fs.homeNode)), os.ModePerm)
	if(err != nil) {
		return err
	}

	return err;
}

func (fs *FileSystem) InsertNode(node []byte, parent []byte) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		if err := b.Put(node, parent); err != nil {
			return fmt.Errorf("[Database][InsertNode] %s", err)
		}

		children := b.Get(parent)
		if children == nil {
			return fmt.Errorf("[Database][InsertNode] parent node %s doesn't exist in Tree Bucket", string(parent));
		}
		if err := b.Put(parent, append(children, node...)); err != nil {
			return fmt.Errorf("[Database][InsertNode] %s", err)
		}

		return nil;
	});

	return err;
}

func (fs *FileSystem) DeleteNodeSubtree(node []byte) error {
	err := fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tree"))

		parent := b.Get(node)[0: fs.nodeLength]

		if err := b.Delete(node); err != nil {
			return fmt.Errorf("[Database][DeleteNodeSubtree] %s", err)
		}

		children := b.Get(parent)
		children_w_node_removed := []byte{}
		for i := 0; i < len(children); i += fs.nodeLength {
			if(bytes.Equal(node, children[i: i + fs.nodeLength])) {
				continue;				
			} else {
				children_w_node_removed = append(children_w_node_removed, children[i: i + fs.nodeLength]...)
			}
		}
		if err := b.Put(parent, children_w_node_removed); err != nil {
			return fmt.Errorf("[Database][DeleteNodeSubtree] %s", err)
		}

		return nil;
	});

	return err;
}

func (fs *FileSystem) recursivelyPrintNode(root []byte, level int) {
	folder_name, _ := fs.getFolderNameForNode(root)
	fmt.Println(strings.Repeat("\t", level) + folder_name)

	children, _ := fs.getChildrenForNode(root)

	for i := 0; i < len(children); i += fs.nodeLength {
		if(i == 0) {
			continue
		}
		fs.recursivelyPrintNode(children[i: i + fs.nodeLength], level + 1)
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
		if(_temp == nil) {
			return fmt.Errorf("[Database][getFolderNameForNode] can't find node %s in FolderNameMapping Bucket", string(_temp))
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
		if(_temp == nil) {
			return fmt.Errorf("[Database][getChildrenForNode] can't find node %s in Tree Bucket", string(_temp))
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
		if(err != nil) {
			return nil, err
		}

		found := false
		for i := 0; i < len(children); i += fs.nodeLength {
			if(i == 0) {
				continue
			}
			
			_folder, err := fs.getFolderNameForNode(children[i: i + fs.nodeLength])
			if(err != nil) {
				return nil, err
			}
			if(_folder == folder) {
				root = children[i: i + fs.nodeLength]
				found = true
				break
			}
		}

		if(!found) {
			return nil, fmt.Errorf("[Database][getFolderNameForNode] can't find the node for folder %s in the hierarchy", folder)
		}
	}

	return root, nil
}

func (fs *FileSystem) CreateFolder(hierarchy []string) (string, error) {
	folder_name := []byte(hierarchy[len(hierarchy) - 1]);
	node := RandStringBytes(fs.nodeLength, fs.homeNode)
	parent, err := fs.getNodeForPath(hierarchy[0: len(hierarchy) - 1])
	if(err != nil) {
		return "", err
	}

	err = fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("FolderNameMapping"))
		if err := b.Put(node, folder_name); err != nil {
			return fmt.Errorf("[Database][CreateFolder] %s", err)
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	err = fs.InsertNode(node, parent)
	if(err != nil) {
		return "", err
	}

	err = os.MkdirAll(filepath.Join(fs.homeDirLocation, string(fs.homeNode), string(node)), os.ModePerm)
	if(err != nil) {
		return "", err
	}

	return string(node), nil
}

func (fs *FileSystem) DeleteFolder(hierarchy []string) error {
	node, err := fs.getNodeForPath(hierarchy)
	if(err != nil) {
		return err
	}
	fmt.Println("node to be deleted", string(node))

	fmt.Println("Removing Folder mapping")
	err = fs.nodesDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("FolderNameMapping"))
		if err := b.Delete(node); err != nil {
			return fmt.Errorf("[Database][DeleteFolder] %s", err)
		}
		return nil
	})
	if(err != nil) {
		return err
	}

	fmt.Println("Deleting from Tree")
	err = fs.DeleteNodeSubtree(node)
	if(err != nil) {
		return err
	}

	err = os.RemoveAll(filepath.Join(fs.homeDirLocation, string(fs.homeNode), string(node)))
	if(err != nil) {
		return err
	}

	return nil
}

func (fs *FileSystem) RecursiveDeleteFolder(hierarchy []string) error {
	err := fs.DeleteFolder(hierarchy)
	if(err != nil) {
		return err
	}

	for i := len(hierarchy) - 2; i >= 0; i-- {
		node, err := fs.getNodeForPath(hierarchy[0: i + 1])
		if(err != nil) {
			return err
		}

		children, err := fs.getChildrenForNode(node)
		if(err != nil) {
			return err
		}

		if(len(children) == 4) {
			if err := fs.DeleteFolder(hierarchy[0: i + 1]); err != nil {
				return err
			}
		} else {
			break
		}
	}

	return nil
}

func (fs *FileSystem) GetHomeFolder() string {
	return filepath.Join(fs.homeDirLocation, string(fs.homeNode))
}
func (fs *FileSystem) PrintBuckets() {
	fs.nodesDB.View(func(tx *bolt.Tx) error {
		fmt.Println()
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
		fmt.Println()

		return nil
	})
}