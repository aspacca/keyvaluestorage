package http

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func healthHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "OK")
}

func (s *Server) notFoundHandler(w http.ResponseWriter, req *http.Request) {
	s.logger.WithField("Component", "HTTP").Debugf("Requested URL not found: %s", req.RequestURI)
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func (s *Server) putHandler(w http.ResponseWriter, req *http.Request) {
	var expiration time.Duration

	vars := mux.Vars(req)
	key := vars["id"]
	value, err := ioutil.ReadAll(req.Body)
	if err != nil {
		s.logger.WithField("Component", "HTTP").Debugf("Error in body content: %s", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	expireIn := req.FormValue("expire_in")

	if len(expireIn) > 0 {
		expirationDuration, err := strconv.Atoi(expireIn)

		if err != nil {
			s.logger.WithField("Component", "HTTP").Debugf("Error in expiration (%s): %s", expireIn, err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		expiration = time.Duration(time.Duration(expirationDuration) * time.Second)
	} else {
		expiration = time.Duration(-1)
	}

	if err := s.storage.Put(key, string(value), expiration); err != nil {
		s.logger.WithField("Component", "HTTP").Errorf("Error putting new key (%s): %s", key, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteHandler(w http.ResponseWriter, req *http.Request) {
	var err error

	vars := mux.Vars(req)
	key := vars["id"]

	if len(key) == 0 {
		err = s.storage.DeleteAll()
	} else {
		err = s.storage.Delete(key)
	}

	if s.storage.IsNotExist(err) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if err != nil {
		s.logger.WithField("Component", "HTTP").Errorf("Error deleting key (%s): %s", key, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) headHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	key := vars["id"]

	_, err := s.storage.Get(key)
	if s.storage.IsNotExist(err) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else if err != nil {
		s.logger.WithField("Component", "HTTP").Errorf("Error hitting key (%s): %s", key, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (s *Server) getHandler(w http.ResponseWriter, req *http.Request) {
	var r io.Reader
	var value []byte
	var err error

	vars := mux.Vars(req)
	key := vars["id"]
	filter := req.FormValue("filter")

	if len(key) == 0 {
		if len(filter) == 0 {
			filter = "*"
		}

		r, err = s.storage.GetPattern(filter)
	} else {
		r, err = s.storage.Get(key)
	}

	if s.storage.IsNotExist(err) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	} else {
		value, err = ioutil.ReadAll(r)
	}

	if err != nil {
		s.logger.WithField("Component", "HTTP").Errorf("Error getting key (%s): %s", key, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	s.streamToWriter(value, w)
}

func (s *Server) streamToWriter(value []byte, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.FormatUint(uint64(len(value)), 10))

	reader := bytes.NewReader(value)
	if _, err := io.Copy(w, reader); err != nil {
		s.logger.WithField("Component", "HTTP").Errorf("Error dumping value, err: %s", err)
		s.logger.WithField("Component", "HTTP").Debugf("Error dumping value, value: %s", value)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
