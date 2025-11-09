package contextforge

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestErrorResponse_Error(t *testing.T) {
	tests := []struct {
		name     string
		response *ErrorResponse
		want     string
	}{
		{
			name: "error with message field",
			response: &ErrorResponse{
				Response: &http.Response{
					StatusCode: 404,
					Request: &http.Request{
						Method: "GET",
						URL:    mustParseURL("http://localhost:8000/tools"),
					},
				},
				Message: "Tool not found",
			},
			want: "GET http://localhost:8000/tools; 404 Tool not found",
		},
		{
			name: "error with errors array",
			response: &ErrorResponse{
				Response: &http.Response{
					StatusCode: 422,
					Request: &http.Request{
						Method: "POST",
						URL:    mustParseURL("http://localhost:8000/tools"),
					},
				},
				Errors: []Error{
					{
						Resource: "Tool",
						Field:    "name",
						Code:     "invalid",
						Message:  "name is invalid",
					},
				},
			},
			want: "POST http://localhost:8000/tools; 422 [{Resource:Tool Field:name Code:invalid Message:name is invalid}]",
		},
		{
			name: "error with only status code",
			response: &ErrorResponse{
				Response: &http.Response{
					StatusCode: 500,
					Request: &http.Request{
						Method: "GET",
						URL:    mustParseURL("http://localhost:8000/tools"),
					},
				},
			},
			want: "GET http://localhost:8000/tools; 500",
		},
		{
			name: "error with credentials in URL",
			response: &ErrorResponse{
				Response: &http.Response{
					StatusCode: 401,
					Request: &http.Request{
						Method: "GET",
						URL:    mustParseURLWithUser("http://user:pass@localhost:8000/tools"),
					},
				},
				Message: "Unauthorized",
			},
			want: "GET http://REDACTED:REDACTED@localhost:8000/tools; 401 Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.response.Error()
			if got != tt.want {
				t.Errorf("ErrorResponse.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRateLimitError_Error(t *testing.T) {
	resetTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	err := &RateLimitError{
		Rate: Rate{
			Limit:     100,
			Remaining: 0,
			Reset:     resetTime,
		},
		Response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Request: &http.Request{
				Method: "GET",
				URL:    mustParseURL("http://localhost:8000/tools"),
			},
		},
		Message: "API rate limit exceeded",
	}

	want := "GET http://localhost:8000/tools; 429 API rate limit exceeded (rate limit; 0/100, reset at 2024-01-01 12:00:00 +0000 UTC)"
	got := err.Error()

	if got != want {
		t.Errorf("RateLimitError.Error() = %q, want %q", got, want)
	}
}

func TestRateLimitError_Is(t *testing.T) {
	resetTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	sharedURL := mustParseURL("http://localhost:8000/tools")
	sharedRequest := &http.Request{
		Method: "GET",
		URL:    sharedURL,
	}
	sharedResponse := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Request:    sharedRequest,
	}

	baseErr := &RateLimitError{
		Rate: Rate{
			Limit:     100,
			Remaining: 0,
			Reset:     resetTime,
		},
		Response: sharedResponse,
		Message:  "API rate limit exceeded",
	}

	tests := []struct {
		name   string
		target error
		want   bool
	}{
		{
			name: "identical values but different instances",
			target: &RateLimitError{
				Rate: Rate{
					Limit:     100,
					Remaining: 0,
					Reset:     resetTime,
				},
				Response: sharedResponse,
				Message:  "API rate limit exceeded",
			},
			want: false,
		},
		{
			name: "different rate",
			target: &RateLimitError{
				Rate: Rate{
					Limit:     50,
					Remaining: 0,
					Reset:     resetTime,
				},
				Response: sharedResponse,
				Message:  "API rate limit exceeded",
			},
			want: false,
		},
		{
			name: "different message",
			target: &RateLimitError{
				Rate: Rate{
					Limit:     100,
					Remaining: 0,
					Reset:     resetTime,
				},
				Response: sharedResponse,
				Message:  "Different message",
			},
			want: false,
		},
		{
			name:   "not a RateLimitError",
			target: &ErrorResponse{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := baseErr.Is(tt.target)
			if got != tt.want {
				t.Errorf("RateLimitError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name  string
		input *url.URL
		want  string
	}{
		{
			name:  "nil URL",
			input: nil,
			want:  "<nil>",
		},
		{
			name:  "URL without credentials",
			input: mustParseURL("http://localhost:8000/tools"),
			want:  "http://localhost:8000/tools",
		},
		{
			name:  "URL with credentials",
			input: mustParseURLWithUser("http://user:password@localhost:8000/tools"),
			want:  "http://REDACTED:REDACTED@localhost:8000/tools",
		},
		{
			name:  "URL with username only",
			input: mustParseURLWithUser("http://user@localhost:8000/tools"),
			want:  "http://REDACTED:REDACTED@localhost:8000/tools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeURL(tt.input)
			var gotStr string
			if got == nil {
				gotStr = "<nil>"
			} else {
				gotStr = got.String()
			}
			if gotStr != tt.want {
				t.Errorf("sanitizeURL() = %q, want %q", gotStr, tt.want)
			}
		})
	}
}

func TestCheckResponse(t *testing.T) {
	tests := []struct {
		name           string
		response       *http.Response
		wantErr        bool
		wantErrType    string
		wantErrMessage string
	}{
		{
			name: "success response",
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			},
			wantErr: false,
		},
		{
			name: "error with JSON response",
			response: &http.Response{
				StatusCode: 404,
				Request: &http.Request{
					Method: "GET",
					URL:    mustParseURL("http://localhost:8000/tools/test"),
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"message": "Tool not found"}`)),
			},
			wantErr:        true,
			wantErrType:    "ErrorResponse",
			wantErrMessage: "GET http://localhost:8000/tools/test; 404 Tool not found",
		},
		{
			name: "error with malformed JSON",
			response: &http.Response{
				StatusCode: 500,
				Request: &http.Request{
					Method: "GET",
					URL:    mustParseURL("http://localhost:8000/tools"),
				},
				Body: io.NopCloser(bytes.NewBufferString(`not valid json`)),
			},
			wantErr:        true,
			wantErrType:    "ErrorResponse",
			wantErrMessage: "GET http://localhost:8000/tools; 500 not valid json",
		},
		{
			name: "rate limit error (429)",
			response: &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Request: &http.Request{
					Method: "GET",
					URL:    mustParseURL("http://localhost:8000/tools"),
				},
				Header: http.Header{
					"X-Ratelimit-Limit":     []string{"100"},
					"X-Ratelimit-Remaining": []string{"0"},
					"X-Ratelimit-Reset":     []string{"2024-01-01T12:00:00Z"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"message": "API rate limit exceeded"}`)),
			},
			wantErr:     true,
			wantErrType: "RateLimitError",
		},
		{
			name: "error with empty body",
			response: &http.Response{
				StatusCode: 404,
				Request: &http.Request{
					Method: "GET",
					URL:    mustParseURL("http://localhost:8000/tools"),
				},
				Body: io.NopCloser(bytes.NewBufferString("")),
			},
			wantErr:        true,
			wantErrType:    "ErrorResponse",
			wantErrMessage: "GET http://localhost:8000/tools; 404",
		},
		{
			name: "error with errors array",
			response: &http.Response{
				StatusCode: 422,
				Request: &http.Request{
					Method: "POST",
					URL:    mustParseURL("http://localhost:8000/tools"),
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"errors": [{"resource": "Tool", "field": "name", "code": "invalid"}]}`)),
			},
			wantErr:     true,
			wantErrType: "ErrorResponse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckResponse(tt.response)

			if tt.wantErr {
				if err == nil {
					t.Error("CheckResponse() expected error, got nil")
					return
				}

				switch tt.wantErrType {
				case "ErrorResponse":
					if _, ok := err.(*ErrorResponse); !ok {
						t.Errorf("CheckResponse() error type = %T, want *ErrorResponse", err)
					}
				case "RateLimitError":
					if _, ok := err.(*RateLimitError); !ok {
						t.Errorf("CheckResponse() error type = %T, want *RateLimitError", err)
					}
				}

				if tt.wantErrMessage != "" && err.Error() != tt.wantErrMessage {
					t.Errorf("CheckResponse() error message = %q, want %q", err.Error(), tt.wantErrMessage)
				}
			} else {
				if err != nil {
					t.Errorf("CheckResponse() unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper functions

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}

func mustParseURLWithUser(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
