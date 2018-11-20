package http

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/PuerkitoBio/ghost/handlers"
	"github.com/gorilla/mux"

	"github.com/aspacca/keyvaluestorage/storage"
	"github.com/sirupsen/logrus"
)

// parse request with maximum memory of _24Kilobits
const _24K = (1 << 10) * 24

// OptionFn Functional option type
type OptionFn func(*Server)

// Listener Set bind address
func Listener(s string) OptionFn {
	return func(srvr *Server) {
		srvr.ListenerString = s
	}

}

// UseStorage Set storage type
func UseStorage(s storage.Storage) OptionFn {
	return func(srvr *Server) {
		srvr.storage = s
	}

}

// Server HTTP Server struct
type Server struct {
	logger  *logrus.Logger
	router  *mux.Router
	storage storage.Storage

	ListenerString string
}

// New HTTP Server factory
func New(options ...OptionFn) (*Server, error) {
	logger := logrus.New()
	logger.Out = os.Stdout

	s := &Server{
		logger: logger,
	}

	for _, optionFn := range options {
		optionFn(s)
	}

	return s, nil
}

func (s *Server) setupRouter() {
	s.router = mux.NewRouter()

	s.router.HandleFunc("/health", healthHandler).Methods("GET")

	s.router.HandleFunc("/keys/{id}", s.getHandler).Methods("GET")
	s.router.HandleFunc("/keys", s.getHandler).Methods("GET")
	s.router.Path("/keys").Queries("filter", "{filter=.*}").HandlerFunc(s.getHandler).Methods("GET")
	s.router.HandleFunc("/keys/{id}", s.putHandler).Methods("PUT")
	s.router.Path("/keys/{id}").Queries("expire_in", "{expire_in=[0-9]+}").HandlerFunc(s.putHandler).Methods("PUT")
	s.router.HandleFunc("/keys/{id}", s.headHandler).Methods("HEAD")
	s.router.HandleFunc("/keys/{id}", s.deleteHandler).Methods("DELETE")
	s.router.HandleFunc("/keys", s.deleteHandler).Methods("DELETE")

	s.router.NotFoundHandler = http.HandlerFunc(s.notFoundHandler)
}

// Server.Run Start the server
func (s *Server) Run() {
	s.logger.Infof("starting Key Value Storage HTTP Backend using storage provider: %s", s.storage.Type())

	s.setupRouter()

	go func() {
		listener := &http.Server{
			Addr:    s.ListenerString,
			Handler: handlers.PanicHandler(s.router, nil),
		}
		listener.ListenAndServe()
	}()

	s.logger.Infof("listening on port: %v\n", s.ListenerString)
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt)
	signal.Notify(term, syscall.SIGTERM)

	<-term

	s.storage.Flush()

	s.logger.Info("server stopped.")
}
