package auths

import "net/http"

type AuthMethod interface {
	Register(mux *http.ServeMux)
}
