package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/carbocation/interpose"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/tomb.v2"

	"wkd-proxy/handler"
)

type statusCodeResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewStatusCodeResponseWriter(w http.ResponseWriter) *statusCodeResponseWriter {
	// WriteHeader is not called if our response implicitly
	// returns 200 OK, so we default to that status code.
	return &statusCodeResponseWriter{w, http.StatusOK}
}

func (scrw *statusCodeResponseWriter) WriteHeader(code int) {
	scrw.statusCode = code
	scrw.ResponseWriter.WriteHeader(code)
}

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
	handler, err := handler.NewHandler(settings.Keyserver)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	s = &Server{
		settings: settings,
		r:        httprouter.New(),
	}

	s.middle = interpose.New()
	s.middle.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			start := time.Now()
			rw.Header().Set("Server", fmt.Sprintf("%s/%s", s.settings.Software, s.settings.Version))
			scrw := NewStatusCodeResponseWriter(rw)
			next.ServeHTTP(scrw, req)
			duration := time.Since(start)

			fields := log.Fields{
				req.Method:    req.URL.String(),
				"duration":    duration.String(),
				"host":        req.Host,
				"status-code": scrw.statusCode,
			}

			if s.settings.HTTP.LogRequestDetails {
				fields["from"] = req.RemoteAddr
				fields["user-agent"] = req.UserAgent()

				proxyHeaders := []string{
					"x-forwarded-for",
					"x-forwarded-host",
					"x-forwarded-server",
				}
				for _, ph := range proxyHeaders {
					if v := req.Header.Get(ph); v != "" {
						fields[ph] = v
					}
				}
			}

			log.WithFields(fields).Info()
		})
	})
	s.middle.UseHandler(s.r)

	handler.Register(s.r)
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
