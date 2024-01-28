package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options

func selfSignedCert() error {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, privateKey.Public(), privateKey)
	if err != nil {
		return err
	}

	// Write private key to file
	privateKeyFile, err := os.Create("private.pem")
	if err != nil {
		return err
	}
	defer privateKeyFile.Close()
	der, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}
	privateKeyPEM := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	err = pem.Encode(privateKeyFile, privateKeyPEM)
	if err != nil {
		return err
	}

	// Write certificate to file
	certFile, err := os.Create("cert.pem")
	if err != nil {
		return err
	}
	defer certFile.Close()
	certPEM := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}
	err = pem.Encode(certFile, certPEM)
	if err != nil {
		return err
	}

	fmt.Println("Self-signed certificate and private key generated successfully")
	return nil
}

func WsNewListener(port int, callback func(c *websocket.Conn)) {
	first := false
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if first {
			w.WriteHeader(401)
			return
		}
		first = true
		c, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		callback(c)
	})
	err := selfSignedCert()
	if err != nil {
		fmt.Println("Error creating selfSignedCert: ", err)
		os.Exit(1)
	}

	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf("0.0.0.0:%d", port), "cert.pem", "private.pem", nil))
}
