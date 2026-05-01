package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	gc "gopkg.in/check.v1"
)

func Test(t *testing.T) { gc.TestingT(t) }

type HandlerSuite struct {
	srv     *httptest.Server
	handler *Handler
}

var _ = gc.Suite(&HandlerSuite{})

func (s *HandlerSuite) SetUpTest(c *gc.C) {
	r := httprouter.New()
	handler, err := NewHandler("hkps://api.protonmail.ch")
	c.Assert(err, gc.IsNil)
	s.handler = handler
	s.handler.Register(r)
	s.srv = httptest.NewServer(r)
}

func (s *HandlerSuite) TearDownTest(c *gc.C) {
	s.srv.Close()
}

func (s *HandlerSuite) TestSettings(c *gc.C) {
	c.Assert(s.handler.Keyserver, gc.Equals, "https://api.protonmail.ch")
}

func (s *HandlerSuite) TestPolicy(c *gc.C) {
	res, err := http.Get(s.srv.URL + "/.well-known/openpgpkey/example.com/policy")
	c.Assert(err, gc.IsNil)
	defer res.Body.Close()
	c.Assert(res.StatusCode, gc.Equals, 200)
	body, err := io.ReadAll(res.Body)
	c.Assert(err, gc.IsNil)
	c.Assert(len(body), gc.Equals, 0)
}

func (s *HandlerSuite) Test404(c *gc.C) {
	res, err := http.Get(s.srv.URL + "/.well-known/openpgpkey/example.com/hu/iy9q119eutrkn8s1mk4r39qejnbu3n5q?l=Joe.Doe")
	c.Assert(err, gc.IsNil)
	defer res.Body.Close()
	c.Assert(res.StatusCode, gc.Equals, 404)
}

func (s *HandlerSuite) TestReal(c *gc.C) {
	res, err := http.Get(s.srv.URL + "/.well-known/openpgpkey/protonmail.com/hu/iy9q119eutrkn8s1mk4r39qejnbu3n5q?l=postmaster")
	c.Assert(err, gc.IsNil)
	defer res.Body.Close()
	c.Assert(res.StatusCode, gc.Equals, 200)
	body, err := io.ReadAll(res.Body)
	c.Assert(len(body), gc.Not(gc.Equals), 0)
	c.Assert(body[:3], gc.DeepEquals, []byte{0xc6, 0x33, 0x04}) // raw OpenPGP packet header
}
