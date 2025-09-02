package share

import (
	"crypto/tls"
	"html/template"
	"log"
	"net/http"
	"spear/auths"
	"spear/auths/httpbasic"
	"spear/auths/spear"
	"spear/config"
	"spear/handlers"
	"spear/serve"
	"spear/utils"
	"strings"
)

// SpearShare spear send -l Port -f <outfile>
func SpearShare(spearConfig *config.SpearConfig, shareParams *config.ShareParams) {
	log.Printf("Share called with params: %+v", shareParams)
	mux := http.NewServeMux()

	var authMethods = []auths.AuthMethod{
		spear.SpearAuth{
			Config: spearConfig,
		},
		httpbasic.BasicAuth{
			Config: spearConfig,
		},
	}

	for _, authMethod := range authMethods {
		authMethod.Register(mux)
	}

	// default route
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		var authResult, _ = handlers.TryAuth(w, r, spearConfig, authMethods)
		// default auth is basic
		if authResult.State == auths.Inapplicable {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"restricted\"")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}

		http.Redirect(w, r, "/files/", http.StatusFound)
	})

	/*mux.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Serving file %s", r.URL.Path)
		p := strings.TrimPrefix(r.URL.Path, "/files/")
		//f, _ := shareParams.MountFS.Open("asd")
		//s, _ := f.Stat()
		//log.Printf("Opened asd: %v", s)
		//
		//f, _ = shareParams.MountFS.Open("asd2")
		//s, _ = f.Stat()
		//log.Printf("Opened asd2: %v", s)
		http.ServeFileFS(w, r, shareParams.MountFS, p)
	})*/
	// serve files logic
	templates, _ := template.New("").ParseFiles("templates/files.gohtml")
	debug := false
	mux.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		if debug {
			log.Printf("Serving file %s", r.URL.Path)
			p := strings.TrimPrefix(r.URL.Path, "/files/")
			http.ServeFileFS(w, r, shareParams.MountFS, p)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		var authResult, _ = handlers.TryAuth(w, r, spearConfig, authMethods)
		contactName := authResult.ContactName

		// default auth is basic
		if authResult.State == auths.Inapplicable || authResult.State == auths.Denied {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"restricted\"")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		relativePathStr := strings.TrimPrefix(r.URL.Path, "/files/")
		if utils.PathContainsDotDot(relativePathStr) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		relativePath := utils.NewPathFS(relativePathStr, shareParams.MountFS)
		if !relativePath.Exists() {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		isFile := relativePath.IsRegular()

		if isFile {
			serve.ServeFile(w, r, relativePath, contactName)
		} else {
			serve.ServeDirectory(w, r, templates, relativePath, contactName)
		}
	})

	addr := shareParams.ListenAddress
	identity, ok := spearConfig.Identities["default"]
	if !ok {
		log.Fatal("default identity not found")
	}
	certPEM := []byte(identity.Cert)
	keyPEM := []byte(identity.Key)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalf("failed to parse TLS certificate: %v", err)
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequestClientCert,
		},
	}
	log.Printf("Starting server on https://%s", addr)
	//srv.ListenAndServe()
	err = srv.ListenAndServeTLS("", "")
	log.Fatal(err)
}
