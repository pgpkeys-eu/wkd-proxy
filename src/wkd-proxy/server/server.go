package server

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/carbocation/interpose"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"gopkg.in/tomb.v2"
)

type Server struct {
	settings  *Settings
	middle    *interpose.Middleware
	r         *httprouter.Router
	logWriter io.WriteCloser
	t         tomb.Tomb
	httpAddr  string
	httpsAddr string
}

func NewServer(settings *Settings) (s *Server, err error) {
	s = &Server{}
	return s, nil
}

func (s *Server) Start() error {
	s.openLog()
	log.Debugf("Starting server with settings: %#v", s.settings)

	s.t.Go(s.listenAndServeHTTP)
	if s.settings.HTTPS != nil {
		s.t.Go(s.listenAndServeHTTPS)
	}

	return nil
}

func (s *Server) Wait() error {
	return s.t.Wait()
}

// ErrStopping is the error indicates that server is stopping normally
var ErrStopping = fmt.Errorf("stopping server")

func (s *Server) Stop() {
	defer s.closeLog()

	s.t.Kill(ErrStopping)
	s.t.Wait()
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

func (s *Server) openLog() {
	defer func() {
		level, err := log.ParseLevel(strings.ToLower(s.settings.LogLevel))
		if err != nil {
			log.Warningf("invalid LogLevel=%q: %v", s.settings.LogLevel, err)
			return
		}
		log.SetLevel(level)
	}()

	s.logWriter = nopCloser{os.Stderr}
	if s.settings.LogFile != "" {
		f, err := os.OpenFile(s.settings.LogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			log.Errorf("failed to open LogFile=%q: %v", s.settings.LogFile, err)
		}
		s.logWriter = f
	}
	log.SetOutput(s.logWriter)
	log.Debug("log opened")
}

func (s *Server) closeLog() {
	log.SetOutput(os.Stderr)
	s.logWriter.Close()
}

func (s *Server) LogRotate() {
	w := s.logWriter
	s.openLog()
	w.Close()
}
