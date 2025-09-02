package spear

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"spear/auths"
	"spear/config"
	"spear/utils"

	"github.com/rs/zerolog/log"
)

type SpearAuth struct {
	Config *config.SpearConfig
}

func (auth SpearAuth) findContact(fingerprint []byte) string {
	stringFingerprint := base64.StdEncoding.EncodeToString(fingerprint)
	for name, contact := range auth.Config.Contacts {
		pubKey := contact.Auths.Spear
		if pubKey == nil {
			continue
		}
		if pubKey.Fingerprint == stringFingerprint {
			return name
		}
	}
	return ""
}

func (auth SpearAuth) Authenticate(w http.ResponseWriter, req *http.Request) auths.AuthResult {

	if req.Header.Get("User-Agent") != "Spear" {
		return auths.AuthResult{State: auths.Inapplicable}
	}
	if len(req.TLS.PeerCertificates) == 0 {
		return auths.AuthResult{State: auths.Denied}
	}

	contactName := auth.contactFromReq(req)
	if contactName == "" {
		log.Debug().Msg("SpearAuth: contact not found in config")
		return auths.AuthResult{State: auths.Denied}
	}

	return auths.AuthResult{State: auths.Valid, ContactName: contactName}
}

func (auth SpearAuth) contactFromReq(req *http.Request) string {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(req.TLS.PeerCertificates[0].PublicKey)
	if err != nil {
		return ""
	}
	fingerprint := utils.GetCertFingerprint(pubKeyBytes)
	auth.findContact(fingerprint[:])
	b64Key := base64.StdEncoding.EncodeToString(fingerprint[:])
	return b64Key
}

type SpearAuthResponse struct {
	ContactName string `json:"contact"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

func (auth SpearAuth) AuthRoute(w http.ResponseWriter, req *http.Request) {
	var authResult auths.AuthResult
	authResult = auth.Authenticate(w, req)
	if authResult.State == auths.Valid {
		response := SpearAuthResponse{
			ContactName: authResult.ContactName,
			Status:      "ok",
			Message:     "",
		}
		log.Info().Msgf("SpearAuth: authenticated contact %s", authResult.ContactName)
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal SpearAuth response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseJSON)
	} else if authResult.State == auths.Denied {
		log.Warn().Msg("SpearAuth: authentication denied")
		response := SpearAuthResponse{
			ContactName: "",
			Status:      "denied",
			Message:     "",
		}
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal SpearAuth response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(responseJSON)
	} else {
		log.Debug().Msg("SpearAuth: inapplicable authentication method")
		response := SpearAuthResponse{
			ContactName: "",
			Status:      "inapplicable",
			Message:     "",
		}
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal SpearAuth response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(responseJSON)
	}
}

func (auth SpearAuth) Register(mux *http.ServeMux) {

	mux.HandleFunc("/auth/spear/", auth.AuthRoute)
	mux.HandleFunc("/auth/spear", auth.AuthRoute)
}
