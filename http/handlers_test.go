package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"os"
	"time"
	"bytes"
	"github.com/aspacca/keyvaluestorage/storage"
	"io/ioutil"
	"path/filepath"
)

func boostrap(t *testing.T) *Server {
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

	strg, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	s, err := New(UseStorage(strg))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	s.setupRouter()

	return s
}

func assertBody(rr *httptest.ResponseRecorder, expected string, t *testing.T) {
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %s want %s",
			rr.Body.String(), expected)
	}
}

func assertStatus(rr *httptest.ResponseRecorder, expected int, t *testing.T) {
	if status := rr.Code; status != expected {
		t.Errorf("handler returned wrong status code: got %d want %d",
			status, expected)
	}
}

func executeRequest(req *http.Request, s *Server) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	return rr
}

func TestServer_NotFound(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("GET", "/not exists", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNotFound, t)
}

func TestServer_Health(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusOK, t)
	assertBody(rr, `OK`, t)
}

func TestServer_PutWithExpiration(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("PUT", "/keys/a key?expire_in=1", bytes.NewReader([]byte("a value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	time.Sleep(time.Duration(2 * time.Second))

	req, err = http.NewRequest("GET", "/keys/a key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNotFound, t)
}

func TestServer_DeleteNotFound(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("DELETE", "/keys/a key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNotFound, t)
}

func TestServer_Delete(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("PUT", "/keys/a key", bytes.NewReader([]byte("a value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("PUT", "/keys/another key", bytes.NewReader([]byte("another value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("DELETE", "/keys/a key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("GET", "/keys/a key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNotFound, t)

	req, err = http.NewRequest("GET", "/keys/another key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusOK, t)
	assertBody(rr, `another value`, t)
}

func TestServer_DeleteAll(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("PUT", "/keys/a key", bytes.NewReader([]byte("a value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("PUT", "/keys/another key", bytes.NewReader([]byte("another value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("DELETE", "/keys", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("GET", "/keys/a key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNotFound, t)

	req, err = http.NewRequest("GET", "/keys/another key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNotFound, t)
}

func TestServer_Get(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("PUT", "/keys/a key", bytes.NewReader([]byte("a value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("GET", "/keys/a key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusOK, t)
	assertBody(rr, `a value`, t)
}

func TestServer_Head(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("PUT", "/keys/a key", bytes.NewReader([]byte("a value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("HEAD", "/keys/a key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusOK, t)
	assertBody(rr, ``, t)
}

func TestServer_GetWithFilter(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("PUT", "/keys/a key", bytes.NewReader([]byte("a value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("PUT", "/keys/another key", bytes.NewReader([]byte("another value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("GET", "/keys?filter=another*", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusOK, t)
	assertBody(rr, `[{"another key":"another value"}]`, t)
}


func TestServer_GetNotFound(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("GET", "/keys/a key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNotFound, t)
}

func TestServer_HeadNotFound(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("HEAD", "/keys/a key", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusNotFound, t)
}

func TestServer_GetWithFilterEmpty(t *testing.T) {
	s := boostrap(t)

	req, err := http.NewRequest("GET", "/keys?filter=*", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr := executeRequest(req, s)

	assertStatus(rr, http.StatusOK, t)
	assertBody(rr, `[]`, t)

	req, err = http.NewRequest("PUT", "/keys/a key", bytes.NewReader([]byte("a value")))
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusNoContent, t)

	req, err = http.NewRequest("GET", "/keys?filter=filter*", nil)
	if err != nil {
		t.Fatalf("err not expected: %s", err)
	}

	rr = executeRequest(req, s)

	assertStatus(rr, http.StatusOK, t)
	assertBody(rr, `[]`, t)
}