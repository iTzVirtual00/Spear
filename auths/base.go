package auths

import "net/http"

type AuthMethod interface {
	Register(mux *http.ServeMux)
	Authenticate(w http.ResponseWriter, r *http.Request) AuthResult
}

type AuthResult struct {
	State       AuthState
	ContactName string
}

type AuthState int

const (
	Inapplicable AuthState = iota
	Denied
	Valid
)
