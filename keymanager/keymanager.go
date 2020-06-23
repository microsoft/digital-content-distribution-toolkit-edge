// Package to maintain a fixed size FIFO cache of public keys
package keymanager

import (
	"sync"
	"crypto/rsa"
	"io/ioutil"

	jwt "github.com/dgrijalva/jwt-go"
)

type KeyManager struct {
	keys []*rsa.PublicKey
	maxLength int
	currentLength int
	mutex sync.Mutex
}

func MakeKeyManager(maxLength int) (*KeyManager, error) {
	var mutex sync.Mutex
	km := KeyManager{make([]*rsa.PublicKey, maxLength), maxLength, 0, mutex}

	return &km, nil
}

func (km *KeyManager) AddKey(publicKeyPath string) error {
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
		km.keys = km.keys[1:]
		km.keys = append(km.keys, key)
	} else {
		km.keys[km.currentLength] = key;
		km.currentLength += 1;
	}

	return nil
}

func (km *KeyManager) GetKeyList() []*rsa.PublicKey {
	return km.keys[:km.currentLength]
}
