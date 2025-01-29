package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/go-logr/logr"
)

// NewProxy returns a proxy.
func NewProxy(
	baseURL string,
	authToken string,
	logger logr.Logger,
) *Proxy {
	return &Proxy{
		baseURL:   baseURL,
		authToken: authToken,
		logger:    logger,
	}
}

// Proxy is a proxy.
type Proxy struct {
	name      string
	baseURL   string
	authToken string
	logger    logr.Logger
}

// forward forwards the request.
//
// TODO(kenji): Support params.
func (p *Proxy) forward(
	w http.ResponseWriter,
	r *http.Request,
	httpMethod string,
	path string,
) (*http.Response, error) {
	p.logger.Info("Forwarding request", "method", httpMethod, "path", path)

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	hreq, err := http.NewRequest(httpMethod, p.baseURL+path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	hreq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.authToken))

	hresp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return nil, err
	}

	return hresp, nil
}
