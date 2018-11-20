package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type fileSystemStorage struct {
	storageDir string
	locks      map[string]*sync.Mutex
}

// NewFileSystemStorage Factory for fs storage
// saves db to `storageDir/*`
func NewFileSystemStorage(storageDir string) (*fileSystemStorage, error) {
	return &fileSystemStorage{
		storageDir: storageDir,
		locks:      map[string]*sync.Mutex{},
	}, nil
}

// fileSystemStorage.Type Returns type of the storage
func (s *fileSystemStorage) Type() string {
	return "fs"
}

// fileSystemStorage.IsNotExist Returns if err is for not existing file
func (s *fileSystemStorage) IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return err == errNotExists
}

func (s *fileSystemStorage) lockAll() {
	for key := range s.locks {
		s.locks[key].Lock()
	}
}

func (s *fileSystemStorage) unlockAll() {
	for key := range s.locks {
		s.locks[key].Unlock()
	}
}

func (s *fileSystemStorage) lock(key string) {
	if _, ok := s.locks[key]; !ok {
		s.locks[key] = &sync.Mutex{}
	}

	s.locks[key].Lock()
}

func (s *fileSystemStorage) unlock(key string) {
	s.locks[key].Unlock()
}

// fileSystemStorage.Get Returns io.Reader for a key or error if it fails
func (s *fileSystemStorage) Get(key string) (io.Reader, error) {
	r := bytes.NewReader(nil)

	s.lock(key)
	defer s.unlock(key)

	key = md5Hash(key)

	b, err := s.getStorageData(key)
	if err != nil {
		return r, err
	}

	if len(b) == 0 {
		return r, errNotExists
	}

	var entry entry
	err = json.Unmarshal(b, &entry)
	if err != nil {
		return r, err
	}

	if !isExpired(entry.Expiration) {
		return bytes.NewReader(entry.Value), nil
	}

	return r, errNotExists
}

// fileSystemStorage.Get Returns io.Reader for a pattern or error if it fails
func (s *fileSystemStorage) GetPattern(pattern string) (io.Reader, error) {
	r := bytes.NewReader(nil)

	s.lockAll()
	defer s.unlockAll()

	keys, err := s.getAllStorageKeys()
	if err != nil {
		return r, err
	}

	ret := make([]string, 0, len(keys))
	for _, key := range keys {
		b, err := s.getStorageData(key)
		if err != nil {
			continue
		}

		if len(b) == 0 {
			continue
		}

		var entry entry
		err = json.Unmarshal(b, &entry)
		if err != nil {
			continue
		}

		if ok, err := filepath.Match(pattern, entry.Key); !ok || err != nil {
			continue
		}

		if !isExpired(entry.Expiration) {
			ret = append(ret, fmt.Sprintf(`{"%s":"%s"}`, entry.Key, entry.Value))
		}
	}

	r = bytes.NewReader([]byte(fmt.Sprintf("[%s]", strings.Join(ret, ","))))

	return r, nil
}

// fileSystemStorage.Delete Deletes an entry by key, returns error if it fails
func (s *fileSystemStorage) Delete(key string) error {
	s.lock(key)
	defer s.unlock(key)

	key = md5Hash(key)

	return s.deleteStorage(key)
}

// fileSystemStorage.DeleteAll Deletes all entries, returns error if it fails
func (s *fileSystemStorage) DeleteAll() error {
	s.lockAll()
	defer s.unlockAll()

	keys, err := s.getAllStorageKeys()
	if err != nil {
		return err
	}

	for _, key := range keys {
		s.deleteStorage(key)
	}

	return nil
}

// fileSystemStorage.Put Saves an entry by key with timeout, returns error if it fails
func (s *fileSystemStorage) Put(key string, value string, expiration time.Duration) error {
	s.lock(key)
	defer s.unlock(key)

	var newExpiration int64
	if expiration != noExpiration {
		newExpiration = time.Now().Add(expiration).UnixNano()
	}

	newEntry := entry{
		Key:        key,
		Value:      []byte(value),
		Expiration: newExpiration,
	}

	dumped, err := json.Marshal(newEntry)
	if err != nil {
		return err
	}

	return s.dumpToStorage(key, dumped)
}

// fileSystemStorage.Flush Flushes storage
func (s *fileSystemStorage) Flush() {

}

func (s *fileSystemStorage) getAllStorageKeys() ([]string, error) {
	if err := os.Mkdir(s.storageDir, 0700); err != nil && !os.IsExist(err) {
		return []string{}, err
	}

	files, err := ioutil.ReadDir(s.storageDir)
	if err != nil {
		return []string{}, err
	}

	r := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() {
			r = append(r, file.Name())
		}
	}

	return r, nil
}

func (s *fileSystemStorage) getStorageData(key string) ([]byte, error) {
	f, err := getReader(s.storageDir, key)
	if err != nil {
		return nil, err
	}

	err = f.Sync()
	if err != nil {
		return []byte(""), err
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return []byte(""), err
	}

	return ioutil.ReadAll(f)

}

func (s *fileSystemStorage) deleteStorage(key string) error {
	if err := os.Mkdir(s.storageDir, 0700); err != nil && !os.IsExist(err) {
		return err
	}

	storagePath := filepath.Join(s.storageDir, key)

	if err := os.Remove(storagePath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		return errNotExists
	}

	return nil
}

func (s *fileSystemStorage) dumpToStorage(key string, data []byte) error {
	key = md5Hash(key)

	f, err := getWriter(s.storageDir, key)
	if err != nil {
		return err
	}

	err = f.Truncate(0)
	if err != nil {
		return err
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	err = f.Sync()

	return err
}
