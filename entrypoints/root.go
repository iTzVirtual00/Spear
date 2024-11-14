package entrypoints

import (
	"crypto/tls"
	"log"
	"net/http"
	"spear/auths"
	"spear/auths/httpbasic"
	"spear/auths/spear"
	"spear/config"
	"spear/fileserver"
	"spear/utils"
)

type ReceiveCommand struct {
	Address         utils.TCPAddr
	Listen          utils.TCPAddr
	Files           []string
	AllowedContacts []string
}
type SendCommand struct {
	Address         utils.TCPAddr
	Listen          utils.TCPAddr
	Files           []string
	AllowedContacts []string
}

// SendClient spear send -a Address[:port] -f <outfile>
func SendClient(config *config.SpearConfig, cmd *SendCommand) {
	log.Println("Called send with client mode!", cmd.Address)
}

// SendAsServer spear send -l Port -f <outfile>
func SendAsServer(spearConfig *config.SpearConfig, cmd *SendCommand) {
	// convert paths to spearParameters
	paths := make([]*utils.Path, len(cmd.Files))
	for i, file := range cmd.Files {
		paths[i] = utils.NewPath(file)
	}

	spearParameters := config.SpearParameters{
		Paths: paths,
	}

	// spin https server up
	mux := http.NewServeMux()

	var authMethods = []auths.AuthMethod{
		httpbasic.BasicAuth{
			Config: spearConfig,
		},
		spear.SpearAuth{
			Config: spearConfig,
		},
	}

	for _, auth := range authMethods {
		auth.Register(mux)
	}

	fb := fileserver.FileserverBackend{
		Config:      spearConfig,
		Parameters:  spearParameters,
		AuthMethods: authMethods,
	}
	fb.RegisterFileserver(mux)

	addr := cmd.Listen
	certFile := "cert.pem"
	keyFile := "key.pem"

	srv := &http.Server{
		Addr:    addr.String(),
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			ClientAuth: tls.RequestClientCert,

			//GetConfigForClient: func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
			//	return nil, nil
			//},
			//InsecureSkipVerify:    true,
			//VerifyPeerCertificate: customCertVerify,
		},
	}

	log.Printf("Starting server on %s", addr)

	err := srv.ListenAndServeTLS(certFile, keyFile)
	log.Fatal(err)
}

// ReceiveClient spear receive -a Address[:port] -f <outfile>
func ReceiveClient(config *config.SpearConfig, cmd *ReceiveCommand) {
	println("Called receive with client mode!")
}

// ReceiveServer spear receive -l Port -f <outfile>
func ReceiveServer(config *config.SpearConfig, cmd *ReceiveCommand) {
	println("Called receive with server mode!")
}
