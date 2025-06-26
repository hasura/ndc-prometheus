package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type apiResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType v1.ErrorType    `json:"errorType"` //nolint:tagliatelle
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings,omitempty"`
}

func apiError(code int) bool {
	// These are the codes that Prometheus sends when it returns an error.
	return code == http.StatusUnprocessableEntity || code == http.StatusBadRequest
}

func errorTypeAndMsgFor(resp *http.Response) (v1.ErrorType, string) {
	switch resp.StatusCode / 100 {
	case 4:
		return v1.ErrClient, fmt.Sprintf("client error: %d", resp.StatusCode)
	case 5:
		return v1.ErrServer, fmt.Sprintf("server error: %d", resp.StatusCode)
	}

	return v1.ErrBadResponse, fmt.Sprintf("bad response code %d", resp.StatusCode)
}
