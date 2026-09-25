package balancer

import (
	"bytes"
	"net/http"
)

// bufferedResponse is a minimal http.ResponseWriter that captures a
// backend's response in memory instead of writing it straight to the client
// It is a wrapper around http.ResponseWriter
type bufferedResponse struct {
	statusCode int
	header     http.Header
	body       *bytes.Buffer
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{
		statusCode: http.StatusOK,
		header:     make(http.Header),
		body:       &bytes.Buffer{},
	}
}

func (br *bufferedResponse) Header() http.Header { return br.header }

func (br *bufferedResponse) Write(p []byte) (int, error) { return br.body.Write(p) }

func (br *bufferedResponse) WriteHeader(statusCode int) { br.statusCode = statusCode }
