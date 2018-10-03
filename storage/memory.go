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

type MemoryStorage struct {
	storageDir   string
	storageCache *os.File
	locks        map[string]*sync.Mutex
	data         map[string]entry
	ticker       *time.Ticker
	quit         chan bool
}

// Factory for memory storage
// saves db to `storageDir/memory.db`
func NewMemoryStorage(storageDir string) (*MemoryStorage, error) {
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

	storage := &MemoryStorage{
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

func (s *MemoryStorage) Type() string {
	return "memory"
}

func (s *MemoryStorage) IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return err == NotExistsError
}

func (s *MemoryStorage) lockAll() {
	for key := range s.locks {
		s.locks[key].Lock()
	}
}

func (s *MemoryStorage) unlockAll() {
	for key := range s.locks {
		s.locks[key].Unlock()
	}
}

func (s *MemoryStorage) lock(key string) {
	if _, ok := s.locks[key]; !ok {
		s.locks[key] = &sync.Mutex{}
	}

	s.locks[key].Lock()
}

func (s *MemoryStorage) unlock(key string) {
	s.locks[key].Unlock()
}

func (s *MemoryStorage) Get(key string) (io.Reader, error) {
	r := bytes.NewReader(nil)

	s.lock(key)
	defer s.unlock(key)

	if entry, ok := s.data[key]; !ok {
		return r, NotExistsError

	} else if !isExpired(entry.Expiration) {
		return bytes.NewReader(entry.Value), nil
	}

	return r, NotExistsError
}

func (s *MemoryStorage) GetPattern(pattern string) (io.Reader, error) {
	r := bytes.NewReader(nil)

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

	r = bytes.NewReader([]byte(fmt.Sprintf("[%s]", strings.Join(ret, ","))))

	return r, nil
}

func (s *MemoryStorage) Delete(key string) error {
	s.lock(key)
	defer s.unlock(key)

	if _, ok := s.data[key]; !ok {
		return NotExistsError

	}

	delete(s.data, key)

	return nil
}

func (s *MemoryStorage) DeleteAll() error {
	s.lockAll()
	defer s.unlockAll()

	for key, _ := range s.data {
		delete(s.data, key)
	}

	return nil
}

func (s *MemoryStorage) Put(key string, value string, expiration time.Duration) error {
	s.lock(key)
	defer s.unlock(key)

	var newExpiration int64
	if expiration != NoExpiration {
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

func (s *MemoryStorage) Flush() {
	err := s.dumpToFilesystem()
	if err != nil {
		logger.Debugf("error in memory storage cache: %s", err)
	}

	s.quit <- true
}

func (s *MemoryStorage) dumpToFilesystem() error {
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
