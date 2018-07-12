package storage

import (
	"testing"
	"os"
	"fmt"
	"time"
	"io/ioutil"
	"path/filepath"
)

func boostrap(t *testing.T) string {
	tmpDir := os.TempDir() + "/" + "keyvaluestorage"
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

func TestNewLocalStorage(t *testing.T) {
	tmpDir := boostrap(t)

	_, err := NewLocalStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}
}

func TestLocalStorage_IsNotExist(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

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

func TestLocalStorage_Type(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	chk := storage.Type()
	if chk != "local" {
		t.Fatalf("expected: %s, found : %s", "local", chk)
	}
}

func TestLocalStorage_PutWithExpiration(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Put("a key", "a value", time.Duration(2 * time.Second))
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

func TestLocalStorage_DeleteEmpty(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	err = storage.Delete("a key")
	if err != NotExistsError {
		t.Fatalf("err not expected: %s", err)
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

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

func TestLocalStorage_DeleteAll(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

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

func TestLocalStorage_Get(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

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

func TestLocalStorage_GetPattern(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

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


func TestLocalStorage_GetEmpty(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

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

func TestLocalStorage_GetPatternEmpty(t *testing.T) {
	tmpDir := boostrap(t)

	storage, err := NewLocalStorage(tmpDir)

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