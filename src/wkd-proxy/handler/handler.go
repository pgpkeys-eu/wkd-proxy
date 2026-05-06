package handler

import (
	"io"
	"net/http"
	"regexp"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"

	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

type Handler struct {
	Keyserver string
}

func NewHandler(keyserver string) (h *Handler, err error) {
	// convert "hkp(s)" to "http(s)" in keyserver URL
	r := regexp.MustCompile(`^hkp`)
	k := r.ReplaceAll([]byte(keyserver), []byte("http"))
	h = &Handler{
		Keyserver: string(k),
	}
	return h, nil
}

func (h *Handler) Register(r *httprouter.Router) {
	r.OPTIONS("/.well-known/openpgpkey/:domain/policy", h.WkdGetHeadOptions)
	r.GET("/.well-known/openpgpkey/:domain/policy", h.WkdPolicy)

	r.OPTIONS("/.well-known/openpgpkey/:domain/hu/:hash", h.WkdGetHeadOptions)
	r.GET("/.well-known/openpgpkey/:domain/hu/:hash", h.WkdGet)
}

func (h *Handler) WkdGetHeadOptions(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Infof("OPTIONS")
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) WkdPolicy(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Infof("GET policy")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) WkdGet(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	log.Infof("GET lookup")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/octet-stream")

	r.ParseForm()
	localpart := r.Form.Get("l")
	domain := p.ByName("domain")
	if localpart == "" || domain == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Infof("lookup request for %s@%s", localpart, domain)

	res, err := http.Get(h.Keyserver + "/pks/lookup?op=get&exact=on&search=" + localpart + "@" + domain)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("lookup error: %v", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		w.WriteHeader(res.StatusCode)
		log.Errorf("got non-200 status from HKP: %d", res.StatusCode)
		return
	}

	a, err := armor.Decode(res.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("decode error: %v", err)
		return
	}

	b, err := io.ReadAll(a.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("read error: %v", err)
		return
	}

	_, err = w.Write(b)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("write error: %v", err)
		return
	}
}
