package httpbasic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"spear/config"
)

type BasicAuth struct {
	Config *config.SpearConfig
}

func sha256Sum(str string) string {
	hash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hash[:])
}

func (auth BasicAuth) Root(w http.ResponseWriter, req *http.Request) {

	u, p, err := req.BasicAuth()
	if !err {
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	pwHash := sha256Sum(p)
	for name, contact := range auth.Config.Contacts {
		httpAuth := contact.Auths.HTTP
		if httpAuth == nil {
			continue
		}
		if httpAuth.Username == u && httpAuth.Password == pwHash {
			fmt.Fprintf(w, "Hi from BasicAuth\n")
			fmt.Fprintf(w, "Welcome back, %s", name)
			return
		}

	}
	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	http.Error(w, "Access denied", http.StatusUnauthorized)
}

func (auth BasicAuth) Register(mux *http.ServeMux) {
	mux.HandleFunc("/basic/", auth.Root)
	mux.HandleFunc("/basic", auth.Root)
}
