package handlers

import (
	"log"
	"net/http"
	"spear/auths"
	"spear/config"
)

func TryAuth(w http.ResponseWriter, r *http.Request, spearConfig *config.SpearConfig, authMethods []auths.AuthMethod) (auths.AuthResult, auths.AuthMethod) {
	//return auths.AuthResult{State: auths.Valid}, nil
	var authResult auths.AuthResult = auths.AuthResult{State: auths.Inapplicable}
	for _, authMethod := range authMethods {
		authResult = authMethod.Authenticate(w, r)
		switch authResult.State {
		case auths.Valid:
			// valid auth, continue to next handler
			log.Printf("Authenticated contact: %s", authResult.ContactName)
			return authResult, authMethod
		case auths.Denied:
			// denied auth, return forbidden
			http.Error(w, "Permission denied", http.StatusForbidden)
			return authResult, authMethod
		case auths.Inapplicable:
			// inapplicable auth, continue to next method
			continue
		}
	}
	return authResult, nil
}
