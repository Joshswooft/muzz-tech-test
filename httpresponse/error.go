package httpresponse

// ErrorResponse represents an error response for a http request
type ErrorResponse struct {
	Error string `json:"error"`
}
