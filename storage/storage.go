package storage

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const noExpiration time.Duration = -1

var errNotExists = fmt.Errorf("entry does not exists")

type entry struct {
	Key        string `json:"key"`
	Value      []byte `json:"value"`
	Expiration int64  `json:"expiration"`
}

// Storage Interface for storage operations
type Storage interface {
	Put(key string, value string, expiration time.Duration) error
	Get(key string) (io.Reader, error)
	GetPattern(pattern string) (io.Reader, error)
	Delete(key string) error
	DeleteAll() error

	Type() string
	IsNotExist(err error) bool

	Flush()
}

func getWriter(storageDir string, fileName string) (*os.File, error) {
	if err := os.Mkdir(storageDir, 0700); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("cannot access storageDir (%s): %s", storageDir, err)
	}

	storagePath := filepath.Join(storageDir, fileName)

	f, err := os.OpenFile(storagePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot access storagePath (%s): %s", storagePath, err)
	}

	return f, nil
}

func getReader(storageDir string, fileName string) (*os.File, error) {
	if err := os.Mkdir(storageDir, 0700); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("cannot access storageDir (%s): %s", storageDir, err)
	}

	storagePath := filepath.Join(storageDir, fileName)

	f, err := os.OpenFile(storagePath, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errNotExists
		}

		return nil, fmt.Errorf("cannot access storagePath (%s): %s", storagePath, err)
	}

	return f, nil
}

func md5Hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func isExpired(expirationTime int64) bool {
	return expirationTime > 0 && time.Now().UnixNano() > expirationTime
}
