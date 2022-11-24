// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

// Package to maintain a fixed size FIFO cache of public keys
package keymanager

import (
	"sync"
	"crypto/rsa"
	"io/ioutil"
	"os"
	"path/filepath"

	jwt "github.com/dgrijalva/jwt-go"
)

type KeyEntry struct {
	key *rsa.PublicKey
	fileName string
}
type KeyManager struct {
	keys []KeyEntry
	maxLength int
	currentLength int
	PubkeysDir string
	mutex sync.Mutex
}

func MakeKeyManager(maxLength int, PubkeysDir string) (*KeyManager, error) {
	var mutex sync.Mutex
	km := KeyManager{make([]KeyEntry, maxLength), maxLength, 0, PubkeysDir, mutex}

	return &km, nil
}

func (km *KeyManager) AddKey(publicKeyName string) error {
	publicKeyPath := filepath.Join(km.PubkeysDir, publicKeyName)
	keyData, err := ioutil.ReadFile(publicKeyPath)
    if err != nil {
        return err
    }

    key, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
    if err != nil {
        return err
	}
	
	km.mutex.Lock()
	defer km.mutex.Unlock()

	if(km.currentLength == km.maxLength) {
		os.Remove(filepath.Join(km.PubkeysDir, km.keys[0].fileName))
		km.keys = km.keys[1:]

		km.keys = append(km.keys, KeyEntry{key, publicKeyName})
	} else {
		km.keys[km.currentLength] = KeyEntry{key, publicKeyName};
		km.currentLength += 1;
	}

	return nil
}

func (km *KeyManager) GetKeyList() []*rsa.PublicKey {
	temp := make([]*rsa.PublicKey, 0)
	for i := 0; i < km.currentLength; i++ {
		temp = append(temp, km.keys[i].key)
	}

	return temp
}
