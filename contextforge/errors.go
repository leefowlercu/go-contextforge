package contextforge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ErrorResponse represents an error response from the ContextForge API.
type ErrorResponse struct {
	Response *http.Response // HTTP response that caused this error
	Message  string         `json:"message,omitempty"`
	Errors   []Error        `json:"errors,omitempty"`
}

// Error represents a single error detail in an API response.
type Error struct {
	Resource string `json:"resource,omitempty"` // Resource on which the error occurred
	Field    string `json:"field,omitempty"`    // Field on which the error occurred
	Code     string `json:"code,omitempty"`     // Validation error code
	Message  string `json:"message,omitempty"`  // Message describing the error
}

func (r *ErrorResponse) Error() string {
	if r.Message != "" {
		return fmt.Sprintf("%v %v; %d %v",
			r.Response.Request.Method, sanitizeURL(r.Response.Request.URL),
			r.Response.StatusCode, r.Message)
	}

	if len(r.Errors) > 0 {
		return fmt.Sprintf("%v %v; %d %+v",
			r.Response.Request.Method, sanitizeURL(r.Response.Request.URL),
			r.Response.StatusCode, r.Errors)
	}

	return fmt.Sprintf("%v %v; %d",
		r.Response.Request.Method, sanitizeURL(r.Response.Request.URL),
		r.Response.StatusCode)
}

// RateLimitError occurs when the API rate limit is exceeded.
type RateLimitError struct {
	Rate     Rate           // Rate specifies the current rate limit information
	Response *http.Response // HTTP response that caused this error
	Message  string         `json:"message,omitempty"`
}

func (r *RateLimitError) Error() string {
	return fmt.Sprintf("%v %v; %d %v (rate limit; %d/%d, reset at %v)",
		r.Response.Request.Method, sanitizeURL(r.Response.Request.URL),
		r.Response.StatusCode, r.Message,
		r.Rate.Remaining, r.Rate.Limit, r.Rate.Reset)
}

// Is returns whether the provided error equals this error.
func (r *RateLimitError) Is(target error) bool {
	v, ok := target.(*RateLimitError)
	if !ok {
		return false
	}

	return r.Rate == v.Rate &&
		r.Message == v.Message &&
		r.Response.StatusCode == v.Response.StatusCode &&
		r.Response.Request.Method == v.Response.Request.Method &&
		sanitizeURL(r.Response.Request.URL) == sanitizeURL(v.Response.Request.URL)
}

// CheckResponse checks the API response for errors, and returns them if present.
// A response is considered an error if it has a status code outside the 200 range.
// API error responses are expected to have either no response body, or a JSON
// response body that maps to ErrorResponse.
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := io.ReadAll(r.Body)
	if err == nil && data != nil && len(data) > 0 {
		if err := json.Unmarshal(data, errorResponse); err != nil {
			// If we can't unmarshal the error, include the raw response
			errorResponse.Message = string(data)
		}
	}

	// Check for rate limit error
	if r.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{
			Rate:     parseRate(r),
			Response: r,
			Message:  errorResponse.Message,
		}
	}

	return errorResponse
}

// sanitizeURL redacts any authentication tokens from the URL.
func sanitizeURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	u2 := *u
	if u2.User != nil {
		u2.User = url.UserPassword("REDACTED", "REDACTED")
	}
	return &u2
}
