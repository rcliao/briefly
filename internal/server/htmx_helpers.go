package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HTMX request/response helpers

// isHTMXRequest checks if the request was made by HTMX
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// getHTMXTarget returns the ID of the target element
func getHTMXTarget(r *http.Request) string {
	return r.Header.Get("HX-Target")
}

// getHTMXTrigger returns the ID of the element that triggered the request
func getHTMXTrigger(r *http.Request) string {
	return r.Header.Get("HX-Trigger")
}

// getHTMXCurrentURL returns the current URL of the browser
func getHTMXCurrentURL(r *http.Request) string {
	return r.Header.Get("HX-Current-URL")
}

// setHTMXTrigger sets a client-side event to trigger after the response
func setHTMXTrigger(w http.ResponseWriter, event string) {
	w.Header().Set("HX-Trigger", event)
}

// setHTMXTriggerWithData sets a client-side event with JSON data
func setHTMXTriggerWithData(w http.ResponseWriter, event string, data interface{}) error {
	payload := map[string]interface{}{
		event: data,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal trigger data: %w", err)
	}
	w.Header().Set("HX-Trigger", string(jsonData))
	return nil
}

// setHTMXRedirect redirects the client to a new URL (client-side redirect)
func setHTMXRedirect(w http.ResponseWriter, url string) {
	w.Header().Set("HX-Redirect", url)
}

// setHTMXRefresh triggers a full page refresh
func setHTMXRefresh(w http.ResponseWriter) {
	w.Header().Set("HX-Refresh", "true")
}

// setHTMXPushURL pushes a URL into the browser history
func setHTMXPushURL(w http.ResponseWriter, url string) {
	w.Header().Set("HX-Push-Url", url)
}

// setHTMXReplaceURL replaces the current URL in the browser history
func setHTMXReplaceURL(w http.ResponseWriter, url string) {
	w.Header().Set("HX-Replace-Url", url)
}

// setHTMXReswap changes the swap method for this response
func setHTMXReswap(w http.ResponseWriter, method string) {
	// Valid methods: innerHTML, outerHTML, beforebegin, afterbegin, beforeend, afterend, delete, none
	w.Header().Set("HX-Reswap", method)
}

// setHTMXRetarget changes the target element for this response
func setHTMXRetarget(w http.ResponseWriter, selector string) {
	w.Header().Set("HX-Retarget", selector)
}

// showToast is a helper to show a toast notification via HTMX trigger
func showToast(w http.ResponseWriter, message, level string) error {
	return setHTMXTriggerWithData(w, "showToast", map[string]string{
		"message": message,
		"level":   level, // success, info, warning, error
	})
}

// ToastLevel represents the toast notification level
type ToastLevel string

const (
	ToastSuccess ToastLevel = "success"
	ToastInfo    ToastLevel = "info"
	ToastWarning ToastLevel = "warning"
	ToastError   ToastLevel = "error"
)

// ShowSuccessToast shows a success toast notification
func ShowSuccessToast(w http.ResponseWriter, message string) error {
	return showToast(w, message, string(ToastSuccess))
}

// ShowErrorToast shows an error toast notification
func ShowErrorToast(w http.ResponseWriter, message string) error {
	return showToast(w, message, string(ToastError))
}

// ShowInfoToast shows an info toast notification
func ShowInfoToast(w http.ResponseWriter, message string) error {
	return showToast(w, message, string(ToastInfo))
}

// ShowWarningToast shows a warning toast notification
func ShowWarningToast(w http.ResponseWriter, message string) error {
	return showToast(w, message, string(ToastWarning))
}
