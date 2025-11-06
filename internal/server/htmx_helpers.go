package server

import (
	"net/http"
)

// HTMX request/response helpers

// isHTMXRequest checks if the request was made by HTMX
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
