package cas

import (
	"log/slog"
	"net/http"
)

// restClientHandler handles CAS REST Protocol over HTTP Basic Authentication
type restClientHandler struct {
	c *RestClient
	h http.Handler
}

// ServeHTTP handles HTTP requests, processes HTTP Basic Authentication over CAS Rest api
// and passes requests up to its child http.Handler.
func (ch *restClientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("cas: handling request", slog.Any("method", r.Method), slog.Any("url", r.URL))

	username, password, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"CAS Protected Area\"")
		w.WriteHeader(401)
		return
	}

	// TODO we should implement a short cache to avoid hitting cas server on every request
	// the cache could use the authorization header as key and the authenticationResponse as value

	success, err := ch.authenticate(username, password)
	if err != nil {
		slog.Info("cas: rest authentication failed", slog.Any("error", err))
		w.Header().Set("WWW-Authenticate", "Basic realm=\"CAS Protected Area\"")
		w.WriteHeader(401)
		return
	}

	setAuthenticationResponse(r, success)
	ch.h.ServeHTTP(w, r)
	return
}

func (ch *restClientHandler) authenticate(username string, password string) (*AuthenticationResponse, error) {
	tgt, err := ch.c.RequestGrantingTicket(username, password)
	if err != nil {
		return nil, err
	}

	st, err := ch.c.RequestServiceTicket(tgt)
	if err != nil {
		return nil, err
	}

	return ch.c.ValidateServiceTicket(st)
}
