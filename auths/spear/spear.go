package spear

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"spear/auths"
	"spear/config"
)

type SpearAuth struct {
	Config *config.SpearConfig
}

func (auth SpearAuth) findContact(b64Key string) string {
	for name, contact := range auth.Config.Contacts {
		pubKey := contact.Auths.Spear
		if pubKey == nil {
			continue
		}
		if pubKey.Key == b64Key {
			return name
		}
	}
	return ""
}

func (auth SpearAuth) Root(w http.ResponseWriter, req *http.Request) {
	if len(req.TLS.PeerCertificates) == 0 {
		fmt.Fprintf(w, "Client auth required\n")
		return
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(req.TLS.PeerCertificates[0].PublicKey)
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	b64 := base64.StdEncoding.EncodeToString(pubKeyBytes)
	contactName := auth.findContact(b64)
	if contactName == "" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	redirect := req.URL.Query().Get("location")
	if redirect == "" {
		redirect = "/files/"
	}

	http.Redirect(w, req, redirect, http.StatusFound)

}

func (auth SpearAuth) Authenticate(w http.ResponseWriter, req *http.Request) auths.AuthResult {
	if len(req.TLS.PeerCertificates) == 0 {
		return auths.AuthResult{State: auths.Inapplicable}
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(req.TLS.PeerCertificates[0].PublicKey)
	if err != nil {
		return auths.AuthResult{State: auths.Inapplicable}
	}

	b64Key := base64.StdEncoding.EncodeToString(pubKeyBytes)
	fmt.Println(b64Key)
	contactName := auth.findContact(b64Key)
	if contactName == "" {
		return auths.AuthResult{State: auths.Inapplicable}
	}

	return auths.AuthResult{State: auths.Valid, ContactName: contactName}
}

func (auth SpearAuth) Register(mux *http.ServeMux) {
	mux.HandleFunc("/spear/", auth.Root)
	mux.HandleFunc("/spear", auth.Root)
}
