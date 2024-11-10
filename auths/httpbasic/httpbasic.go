package httpbasic

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"spear/auths"
	"spear/config"
)

type BasicAuth struct {
	Config *config.SpearConfig
}

func sha256Sum(str string) string {
	hash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hash[:])
}

// remember that basicauth's username may be different from contact's name
func (auth BasicAuth) findContact(username string, password string) string {
	pwSum := sha256Sum(password)
	for name, contact := range auth.Config.Contacts {
		creds := contact.Auths.HTTP
		if creds == nil {
			continue
		}
		if creds.Username == username && creds.Password == pwSum {
			return name
		}
	}
	return ""
}

func (auth BasicAuth) Root(w http.ResponseWriter, req *http.Request) {
	u, p, ok := req.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	contactName := auth.findContact(u, p)

	if contactName == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	redirect := req.URL.Query().Get("location")
	if redirect == "" {
		redirect = "/files/"
	}

	http.Redirect(w, req, redirect, http.StatusFound)
}

func (auth BasicAuth) Authenticate(w http.ResponseWriter, req *http.Request) auths.AuthResult {
	u, p, ok := req.BasicAuth()
	if !ok {
		return auths.AuthResult{State: auths.Inapplicable}
	}
	contactName := auth.findContact(u, p)
	if contactName == "" {
		return auths.AuthResult{State: auths.Denied}
	}
	return auths.AuthResult{State: auths.Valid, ContactName: contactName}
}

func (auth BasicAuth) Register(mux *http.ServeMux) {
	mux.HandleFunc("/basic/", auth.Root)
	mux.HandleFunc("/basic", auth.Root)
}
