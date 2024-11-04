package spear

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"main/config"
	"net/http"
)

type SpearAuth struct {
	Config *config.SpearConfig
}

func (auth SpearAuth) Root(w http.ResponseWriter, req *http.Request) {
	if len(req.TLS.PeerCertificates) == 0 {
		fmt.Fprintf(w, "Client auth required\n")
		return
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(req.TLS.PeerCertificates[0].PublicKey)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintf(w, "SpearAuth failed 2\n")
		return
	}
	fmt.Fprintf(w, "Hi from SpearAuth\n")
	b64 := base64.StdEncoding.EncodeToString(pubKeyBytes)
	fmt.Fprintf(w, "%s\n", b64)
	for name, contact := range auth.Config.Contacts {
		spear := contact.Auths.Spear
		if spear == nil {
			continue
		}
		if spear.Key == b64 {
			fmt.Fprintf(w, "Welcome back, %s", name)
			return
		}

	}
	fmt.Fprintf(w, "Invalid credentials")
}

func (auth SpearAuth) Register(mux *http.ServeMux) {
	mux.HandleFunc("/spear/", auth.Root)
	mux.HandleFunc("/spear", auth.Root)
}
