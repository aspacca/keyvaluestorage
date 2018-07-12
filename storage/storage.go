package storage

import (
	"time"
	"os"
	"sync"
	"path/filepath"
	"encoding/json"
	"fmt"
	"strings"
	"io/ioutil"
	"io"
	"bytes"
	"crypto/md5"
)

const NoExpiration time.Duration = -1

var NotExistsError = fmt.Errorf("entry does not exists")

type entry struct {
	Key string `json:"key"`
	Value []byte `json:"value"`
	Expiration int64 `json:"expiration"`
}

type Storage interface {
	Put(key string, value string, expiration time.Duration) error
	Get(key string) (io.Reader, error)
	GetPattern(pattern string) (io.Reader, error)
	Delete(key string) error
	DeleteAll() error

	Type() string
	IsNotExist(err error) bool
}

type LocalStorage struct {
	storageDir string
	locks map[string]*sync.Mutex
}

// Factory for local storage
// saves db to `storageDir/storage.db`
// track metrics on nullable prometehus.HistogramVec
func NewLocalStorage(storageDir string) (*LocalStorage, error) {
	return &LocalStorage{
		storageDir: storageDir,
		locks: map[string]*sync.Mutex{},
	}, nil
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
			return nil, NotExistsError
		} else {
			return nil, fmt.Errorf("cannot access storagePath (%s): %s", storagePath, err)
		}
	}

	return f, nil
}

func (s *LocalStorage) Type() string {
	return "local"
}

func (s *LocalStorage) IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return err == NotExistsError
}

func (s *LocalStorage) lockAll() {
	for key := range s.locks {
		s.locks[key].Lock()
	}
}

func (s *LocalStorage) unlockAll() {
	for key := range s.locks {
		s.locks[key].Unlock()
	}
}

func (s *LocalStorage) lock(key string) {
	if _, ok := s.locks[key]; !ok {
		s.locks[key] = &sync.Mutex{}
	}

	s.locks[key].Lock()
}

func (s *LocalStorage) unlock(key string) {
	s.locks[key].Unlock()
}

func (s *LocalStorage) Get(key string) (io.Reader, error) {
	r := bytes.NewReader(nil)

	s.lock(key)
	defer s.unlock(key)

	key = md5Hash(key)

	b, err := s.getStorageData(key)
	if err != nil {
		return r, err
	}

	if len(b) == 0 {
		return r, NotExistsError
	}

	var entry entry
	err = json.Unmarshal(b, &entry)
	if err != nil {
		return r, err
	}

	if !isExpired(entry.Expiration) {
		return bytes.NewReader(entry.Value), nil
	}

	return r, NotExistsError
}

func (s *LocalStorage) GetPattern(pattern string) (io.Reader, error) {
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

func (s *LocalStorage) Delete(key string) error {
	s.lock(key)
	defer s.unlock(key)

	key = md5Hash(key)

	return s.deleteStorage(key)
}

func (s *LocalStorage) DeleteAll() error {
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

func (s *LocalStorage) Put(key string, value string, expiration time.Duration) error {
	s.lock(key)
	defer s.unlock(key)

	var newExpiration int64
	if expiration != NoExpiration {
		newExpiration = time.Now().Add(expiration).UnixNano()
	}

	newEntry := entry{
		Key:key,
		Value:[]byte(value),
		Expiration:newExpiration,
	}

	dumped, err := json.Marshal(newEntry)
	if err != nil {
		return err
	}

	return s.dumpToStorage(key, dumped)
}

func (s *LocalStorage) getAllStorageKeys() ([]string, error) {
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

func (s *LocalStorage) getStorageData(key string) ([]byte, error) {
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

func (s *LocalStorage) deleteStorage(key string) error {
	if err := os.Mkdir(s.storageDir, 0700); err != nil && !os.IsExist(err) {
		return err
	}

	storagePath := filepath.Join(s.storageDir, key)

	if err := os.Remove(storagePath); err != nil {
		if !os.IsNotExist(err) {
			return err
		} else {
			return NotExistsError
		}
	}

	return nil
}

func (s *LocalStorage) dumpToStorage(key string, data []byte) error {
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

func md5Hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func isExpired(expirationTime int64) bool {
	return expirationTime > 0 && time.Now().UnixNano() > expirationTime
}
