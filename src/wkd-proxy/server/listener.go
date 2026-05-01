package server

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by listenAndServe and listenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept implements net.Listener.
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func (s *Server) newListener(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	s.t.Go(func() error {
		<-s.t.Dying()
		return ln.Close()
	})
	return tcpKeepAliveListener{ln.(*net.TCPListener)}, nil
}

var newListener = (*Server).newListener

func (s *Server) listenAndServeHTTP() error {
	ln, err := newListener(s, s.settings.HTTP.Bind)
	if err != nil {
		return errors.WithStack(err)
	}
	s.httpAddr = ln.Addr().String()
	return http.Serve(ln, s.middle)
}

func (s *Server) listenAndServeHTTPS() error {
	config := &tls.Config{
		NextProtos: []string{"http/1.1"},
	}
	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(s.settings.HTTPS.Cert, s.settings.HTTPS.Key)
	if err != nil {
		return errors.Wrapf(err, "failed to load HKPS certificate=%q key=%q", s.settings.HTTPS.Cert, s.settings.HTTPS.Key)
	}

	ln, err := newListener(s, s.settings.HTTPS.Bind)
	if err != nil {
		return errors.WithStack(err)
	}
	s.httpsAddr = ln.Addr().String()
	ln = tls.NewListener(ln, config)
	return http.Serve(ln, s.middle)
}
