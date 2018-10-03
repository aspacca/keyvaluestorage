package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func boostrapFilesystem(t *testing.T) string {
	tmpDir := filepath.Join(os.TempDir(), "keyvaluestorage")
	files, err := ioutil.ReadDir(tmpDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("err in boostrap: %s", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(tmpDir, file.Name())
			err := os.Remove(filePath)
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("err in boostrap: %s", err)
			}
		}
	}

	return tmpDir
}

func TestNewFileSystemStorage(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	_, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}
}

func TestFileSystemStorage_IsNotExist(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	b := storage.IsNotExist(NotExistsError)
	if !b {
		t.Fatalf("expected: %t, found : %t", true, b)
	}

	b = storage.IsNotExist(nil)
	if b {
		t.Fatalf("expected: %t, found : %t", false, b)
	}

	b = storage.IsNotExist(fmt.Errorf("some error"))
	if b {
		t.Fatalf("expected: %t, found : %t", false, b)
	}
}

func TestFileSystemStorage_Type(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	chk := storage.Type()
	if chk != "fs" {
		t.Fatalf("expected: %s, found : %s", "fs", chk)
	}
}

func TestFileSystemStorage_PutWithExpiration(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Put("a key", "a value", time.Duration(2*time.Second))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	time.Sleep(time.Duration(2 * time.Second))

	r, err := storage.Get("a key")
	if err != NotExistsError {
		t.Fatalf("err not expected: %s", err)
	}

	chk, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	if len(chk) != 0 {
		t.Fatalf("expected empty, found : %s", chk)
	}
}

func TestFileSystemStorage_DeleteEmpty(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Delete("a key")
	if err != NotExistsError {
		t.Fatalf("err not expected: %s", err)
	}
}

func TestFileSystemStorage_Delete(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Put("a key", "a value", time.Duration(-1))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Put("another key", "another value", time.Duration(-1))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Delete("a key")
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	r, err := storage.Get("a key")
	if err != NotExistsError {
		t.Fatalf("err not expected: %s", err)
	}

	chk, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	if len(chk) != 0 {
		t.Fatalf("expected empty, found : %s", chk)
	}

	r, err = storage.Get("another key")
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	chk, err = ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	if string(chk) != "another value" {
		t.Fatalf("expected: %s, found : %s", "[]", chk)
	}
}

func TestFileSystemStorage_DeleteAll(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Put("a key", "a value", time.Duration(-1))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Put("another key", "another value", time.Duration(-1))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.DeleteAll()
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	r, err := storage.Get("a key")
	if err != NotExistsError {
		t.Fatalf("err not expected: %s", err)
	}

	chk, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	if len(chk) != 0 {
		t.Fatalf("expected empty, found : %s", chk)
	}

	r, err = storage.Get("another key")
	if err != NotExistsError {
		t.Fatalf("err not expected: %s", err)
	}

	chk, err = ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	if len(chk) != 0 {
		t.Fatalf("expected empty, found : %s", chk)
	}
}

func TestFileSystemStorage_Get(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Put("a key", "a value", time.Duration(-1))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	r, err := storage.Get("a key")
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	chk, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	if string(chk) != "a value" {
		t.Fatalf("expected: %s, found : %s", "[]", chk)
	}
}

func TestFileSystemStorage_GetPattern(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Put("a key", "a value", time.Duration(-1))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Put("another key", "another value", time.Duration(-1))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	r, err := storage.GetPattern("another*")
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	chk, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	if string(chk) != `[{"another key":"another value"}]` {
		t.Fatalf("expected: %s, found : %s", `[{"another key":"another value"}]`, chk)
	}
}

func TestFileSystemStorage_GetEmpty(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	r, err := storage.Get("a key")
	if err != NotExistsError {
		t.Fatalf("err not expected: %s", err)
	}

	chk, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	if len(chk) != 0 {
		t.Fatalf("expected empty, found : %s", chk)
	}
}

func TestFileSystemStorage_GetPatternEmpty(t *testing.T) {
	tmpDir := boostrapFilesystem(t)

	storage, err := NewFileSystemStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	r, err := storage.GetPattern("a*glob?")
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	chk, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	if string(chk) != "[]" {
		t.Fatalf("expected: %s, found : %s", "[]", chk)
	}
}
