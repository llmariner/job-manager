package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// NewProxy returns a proxy.
func NewProxy(
	baseURL string,
	authToken string,
) *Proxy {
	return &Proxy{
		baseURL:   baseURL,
		authToken: authToken,
	}
}

// Proxy is a proxy.
type Proxy struct {
	baseURL   string
	authToken string
}

// forward forwards the request.
//
// TODO(kenji): Support params.
func (p *Proxy) forward(
	w http.ResponseWriter,
	r *http.Request,
	httpMethod string,
	path string,
) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hreq, err := http.NewRequest(httpMethod, p.baseURL+path, bytes.NewReader(reqBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hreq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.authToken))

	hresp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(hresp.StatusCode)

	respBody, err := io.ReadAll(hresp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(w, bytes.NewBuffer(respBody)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
