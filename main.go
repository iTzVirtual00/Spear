package main

import (
	"crypto/tls"
	"log"
	"main/auths"
	"main/auths/httpbasic"
	"main/auths/spear"
	"main/config"
	"main/fileserver"
	"main/utils"
	"net/http"
)

func main() {

	spearConfig, err := config.LoadConfig("spear.yml")

	paths := []*utils.Path{
		utils.NewPath("./config"),
	}
	spearParameters := config.SpearParameters{
		Paths: paths,
	}
	if err != nil {
		log.Fatal(err)
		return
	}
	mux := http.NewServeMux()

	var authMethods = []auths.AuthMethod{
		spear.SpearAuth{
			Config: spearConfig,
		},
		httpbasic.BasicAuth{
			Config: spearConfig,
		},
	}
	for _, auth := range authMethods {
		auth.Register(mux)
	}
	fb := fileserver.FileserverBackend{
		Config:     spearConfig,
		Parameters: spearParameters,
	}
	fb.RegisterFileserver(mux)

	addr := "0.0.0.0:8000"
	certFile := "cert.pem"
	keyFile := "key.pem"

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			GetConfigForClient: func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
				return nil, nil
			},
			ClientAuth: tls.RequestClientCert,

			//InsecureSkipVerify:    true,
			//VerifyPeerCertificate: customCertVerify,
		},
	}

	log.Printf("Starting server on %s", addr)

	err = srv.ListenAndServeTLS(certFile, keyFile)
	log.Fatal(err)

}
