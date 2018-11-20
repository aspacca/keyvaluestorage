package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const memoryCacheFile = "memory.db"

var logger *logrus.Logger

type memoryStorage struct {
	storageDir   string
	storageCache *os.File
	locks        map[string]*sync.Mutex
	data         map[string]entry
	ticker       *time.Ticker
	quit         chan bool
}

// NewMemoryStorage Factory for memory storage
// saves db to `storageDir/memory.db`
func NewMemoryStorage(storageDir string) (*memoryStorage, error) {
	logger = logrus.New()
	logger.Out = os.Stdout

	storageCache, err := getWriter(storageDir, memoryCacheFile)
	if err != nil {
		return nil, err
	}

	cache, err := ioutil.ReadAll(storageCache)
	if err != nil {
		return nil, err
	}

	if len(cache) == 0 {
		cache = []byte("{}")
	}

	var data map[string]entry
	err = json.Unmarshal(cache, &data)
	if err != nil {
		return nil, err
	}

	storage := &memoryStorage{
		storageDir:   storageDir,
		storageCache: storageCache,
		data:         data,
		locks:        map[string]*sync.Mutex{},
		ticker:       time.NewTicker(15 * time.Second),
		quit:         make(chan bool),
	}

	go func() {
		for {
			select {
			case <-storage.ticker.C:
				err := storage.dumpToFilesystem()
				if err != nil {
					logger.Debugf("error in memory storage cache: %s", err)
				}
			case <-storage.quit:
				storage.ticker.Stop()
				return
			}
		}
	}()

	return storage, nil
}

// memoryStorage.Type Returns type of the storage
func (s *memoryStorage) Type() string {
	return "memory"
}

// memoryStorage.IsNotExist Returns if err is for not existing file
func (s *memoryStorage) IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return err == errNotExists
}

func (s *memoryStorage) lockAll() {
	for key := range s.locks {
		s.locks[key].Lock()
	}
}

func (s *memoryStorage) unlockAll() {
	for key := range s.locks {
		s.locks[key].Unlock()
	}
}

func (s *memoryStorage) lock(key string) {
	if _, ok := s.locks[key]; !ok {
		s.locks[key] = &sync.Mutex{}
	}

	s.locks[key].Lock()
}

func (s *memoryStorage) unlock(key string) {
	s.locks[key].Unlock()
}

// memoryStorage.Get Returns io.Reader for a key or error if it fails
func (s *memoryStorage) Get(key string) (io.Reader, error) {
	r := bytes.NewReader(nil)

	s.lock(key)
	defer s.unlock(key)

	if entry, ok := s.data[key]; !ok {
		return r, errNotExists

	} else if !isExpired(entry.Expiration) {
		return bytes.NewReader(entry.Value), nil
	}

	return r, errNotExists
}

// memoryStorage.Get Returns io.Reader for a pattern or error if it fails
func (s *memoryStorage) GetPattern(pattern string) (io.Reader, error) {
	s.lockAll()
	defer s.unlockAll()

	ret := make([]string, 0, len(s.data))
	for _, entry := range s.data {
		if ok, err := filepath.Match(pattern, entry.Key); !ok || err != nil {
			continue
		}

		if !isExpired(entry.Expiration) {
			ret = append(ret, fmt.Sprintf(`{"%s":"%s"}`, entry.Key, entry.Value))
		}
	}

	r := bytes.NewReader([]byte(fmt.Sprintf("[%s]", strings.Join(ret, ","))))

	return r, nil
}

// memoryStorage.Delete Deletes an entry by key, returns error if it fails
func (s *memoryStorage) Delete(key string) error {
	s.lock(key)
	defer s.unlock(key)

	if _, ok := s.data[key]; !ok {
		return errNotExists

	}

	delete(s.data, key)

	return nil
}

// memoryStorage.DeleteAll Deletes all entries, returns error if it fails
func (s *memoryStorage) DeleteAll() error {
	s.lockAll()
	defer s.unlockAll()

	for key := range s.data {
		delete(s.data, key)
	}

	return nil
}

// memoryStorage.Put Saves an entry by key with timeout, returns error if it fails
func (s *memoryStorage) Put(key string, value string, expiration time.Duration) error {
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

	s.data[key] = newEntry

	return nil
}

// memoryStorage.Flush Flushes storage
func (s *memoryStorage) Flush() {
	err := s.dumpToFilesystem()
	if err != nil {
		logger.Debugf("error in memory storage cache: %s", err)
	}

	s.quit <- true
}

func (s *memoryStorage) dumpToFilesystem() error {
	s.lockAll()
	defer s.unlockAll()

	f, err := getWriter(s.storageDir, memoryCacheFile)
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

	data, err := json.Marshal(s.data)
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
