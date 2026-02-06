package contextforge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		bearerToken string
		wantAddress string
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:        "valid URL with trailing slash",
			address:     "https://api.example.com/v1/",
			bearerToken: "test-token",
			wantAddress: "https://api.example.com/v1/",
			wantErr:     false,
		},
		{
			name:        "valid URL without trailing slash",
			address:     "https://api.example.com/v1",
			bearerToken: "test-token",
			wantAddress: "https://api.example.com/v1/",
			wantErr:     false,
		},
		{
			name:        "localhost URL",
			address:     "http://localhost:9000/",
			bearerToken: "test-token",
			wantAddress: "http://localhost:9000/",
			wantErr:     false,
		},
		{
			name:        "localhost URL without trailing slash",
			address:     "http://localhost:9000",
			bearerToken: "test-token",
			wantAddress: "http://localhost:9000/",
			wantErr:     false,
		},
		{
			name:        "invalid URL",
			address:     "://invalid-url",
			bearerToken: "test-token",
			wantErr:     true,
			wantErrMsg:  "invalid address",
		},
		{
			name:        "URL with path",
			address:     "https://api.example.com/contextforge/api/",
			bearerToken: "test-token",
			wantAddress: "https://api.example.com/contextforge/api/",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(nil, tt.address, tt.bearerToken)

			if tt.wantErr {
				if err == nil {
					t.Error("NewClient() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("NewClient() error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
				return
			}

			if c == nil {
				t.Fatal("NewClient() returned nil client")
			}

			if c.Address.String() != tt.wantAddress {
				t.Errorf("NewClient() Address = %q, want %q", c.Address.String(), tt.wantAddress)
			}

			if c.BearerToken != tt.bearerToken {
				t.Errorf("NewClient() BearerToken = %q, want %q", c.BearerToken, tt.bearerToken)
			}

			if c.UserAgent != userAgent {
				t.Errorf("NewClient() UserAgent = %q, want %q", c.UserAgent, userAgent)
			}

			if c.Tools == nil {
				t.Error("NewClient() Tools service is nil")
			}

			if c.Resources == nil {
				t.Error("NewClient() Resources service is nil")
			}

			if c.Gateways == nil {
				t.Error("NewClient() Gateways service is nil")
			}

			if c.Cancel == nil {
				t.Error("NewClient() Cancel service is nil")
			}
		})
	}
}

func TestNewClient_CustomHTTPClient(t *testing.T) {
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}
	c, err := NewClient(httpClient, "https://api.example.com/", "test-token")

	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	if c.client != httpClient {
		t.Error("NewClient() did not use provided HTTP client")
	}
}

func TestNewRequest(t *testing.T) {
	c, err := NewClient(nil, "http://localhost:8000/", "test-token")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	tests := []struct {
		name        string
		address     string
		method      string
		urlStr      string
		body        any
		wantErr     bool
		wantErrMsg  string
		checkHeader bool
	}{
		{
			name:    "valid request without body",
			address: "http://localhost:8000/",
			method:  "GET",
			urlStr:  "tools",
			body:    nil,
			wantErr: false,
		},
		{
			name:    "valid request with body",
			address: "http://localhost:8000/",
			method:  "POST",
			urlStr:  "tools",
			body:    map[string]string{"name": "test"},
			wantErr: false,
		},
		{
			name:       "address without trailing slash",
			address:    "http://localhost:8000",
			method:     "GET",
			urlStr:     "tools",
			body:       nil,
			wantErr:    true,
			wantErrMsg: "Address must have a trailing slash",
		},
		{
			name:       "invalid URL path",
			address:    "http://localhost:8000/",
			method:     "GET",
			urlStr:     "://invalid",
			body:       nil,
			wantErr:    true,
			wantErrMsg: "parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address, _ := url.Parse(tt.address)
			c.Address = address

			req, err := c.NewRequest(tt.method, tt.urlStr, tt.body)

			if tt.wantErr {
				if err == nil {
					t.Error("NewRequest() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("NewRequest() error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewRequest() unexpected error: %v", err)
				return
			}

			if req == nil {
				t.Fatal("NewRequest() returned nil request")
			}

			if req.Method != tt.method {
				t.Errorf("NewRequest() method = %q, want %q", req.Method, tt.method)
			}

			if tt.body != nil {
				if req.Header.Get("Content-Type") != mediaTypeJSON {
					t.Errorf("NewRequest() Content-Type = %q, want %q", req.Header.Get("Content-Type"), mediaTypeJSON)
				}
			}

			if req.Header.Get("Accept") != mediaTypeJSON {
				t.Errorf("NewRequest() Accept = %q, want %q", req.Header.Get("Accept"), mediaTypeJSON)
			}

			if req.Header.Get("User-Agent") == "" {
				t.Error("NewRequest() User-Agent header not set")
			}

			if req.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("NewRequest() Authorization = %q, want %q", req.Header.Get("Authorization"), "Bearer test-token")
			}
		})
	}
}

func TestNewRequest_BadJSON(t *testing.T) {
	c, err := NewClient(nil, "http://localhost:8000/", "test-token")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	type InvalidJSON struct {
		BadField chan int
	}

	_, err = c.NewRequest("POST", "tools", &InvalidJSON{BadField: make(chan int)})
	if err == nil {
		t.Error("NewRequest() expected JSON encoding error, got nil")
	}
}

func TestParseRate(t *testing.T) {
	tests := []struct {
		name     string
		headers  http.Header
		wantRate Rate
	}{
		{
			name:     "no rate limit headers",
			headers:  http.Header{},
			wantRate: Rate{},
		},
		{
			name: "all rate limit headers present",
			headers: http.Header{
				"X-Ratelimit-Limit":     []string{"100"},
				"X-Ratelimit-Remaining": []string{"50"},
				"X-Ratelimit-Reset":     []string{"2024-01-01T12:00:00Z"},
			},
			wantRate: Rate{
				Limit:     100,
				Remaining: 50,
				Reset:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "only limit header",
			headers: http.Header{
				"X-Ratelimit-Limit": []string{"100"},
			},
			wantRate: Rate{
				Limit:     100,
				Remaining: 0,
				Reset:     time.Time{},
			},
		},
		{
			name: "invalid reset timestamp",
			headers: http.Header{
				"X-Ratelimit-Limit":     []string{"100"},
				"X-Ratelimit-Remaining": []string{"50"},
				"X-Ratelimit-Reset":     []string{"invalid"},
			},
			wantRate: Rate{
				Limit:     100,
				Remaining: 50,
				Reset:     time.Time{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: tt.headers,
			}

			rate := parseRate(resp)

			if rate.Limit != tt.wantRate.Limit {
				t.Errorf("parseRate() Limit = %d, want %d", rate.Limit, tt.wantRate.Limit)
			}
			if rate.Remaining != tt.wantRate.Remaining {
				t.Errorf("parseRate() Remaining = %d, want %d", rate.Remaining, tt.wantRate.Remaining)
			}
			if !rate.Reset.Equal(tt.wantRate.Reset) {
				t.Errorf("parseRate() Reset = %v, want %v", rate.Reset, tt.wantRate.Reset)
			}
		})
	}
}

func TestParseCursor(t *testing.T) {
	tests := []struct {
		name       string
		headers    http.Header
		wantCursor string
	}{
		{
			name:       "no cursor header",
			headers:    http.Header{},
			wantCursor: "",
		},
		{
			name: "cursor header present",
			headers: http.Header{
				"X-Next-Cursor": []string{"abc123"},
			},
			wantCursor: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: tt.headers,
			}

			cursor := parseCursor(resp)

			if cursor != tt.wantCursor {
				t.Errorf("parseCursor() = %q, want %q", cursor, tt.wantCursor)
			}
		})
	}
}

func TestDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"123","name":"test"}`))
	}))
	defer server.Close()

	c, err := NewClient(nil, server.URL+"/", "test-token")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	req, _ := c.NewRequest("GET", "tools", nil)

	var result map[string]string
	resp, err := c.Do(context.Background(), req, &result)

	if err != nil {
		t.Errorf("Do() unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("Do() returned nil response")
	}

	if result["id"] != "123" {
		t.Errorf("Do() result id = %q, want %q", result["id"], "123")
	}

	if result["name"] != "test" {
		t.Errorf("Do() result name = %q, want %q", result["name"], "test")
	}
}

func TestDo_NilContext(t *testing.T) {
	c, err := NewClient(nil, "http://localhost:8000/", "test-token")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	req, _ := c.NewRequest("GET", "tools", nil)

	_, err = c.Do(nil, req, nil)

	if err == nil {
		t.Error("Do() with nil context expected error, got nil")
	}

	if !strings.Contains(err.Error(), "context must be non-nil") {
		t.Errorf("Do() error = %q, want to contain %q", err.Error(), "context must be non-nil")
	}
}

func TestAddOptions(t *testing.T) {
	opts := &ListOptions{
		Limit:  10,
		Cursor: "abc123",
	}

	u, err := addOptions("tools", opts)
	if err != nil {
		t.Errorf("addOptions() unexpected error: %v", err)
	}

	if !strings.Contains(u, "limit=10") {
		t.Errorf("addOptions() url = %q, want to contain %q", u, "limit=10")
	}

	if !strings.Contains(u, "cursor=abc123") {
		t.Errorf("addOptions() url = %q, want to contain %q", u, "cursor=abc123")
	}
}
