package contextforge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	defaultBaseURL = "http://localhost:8000/"
	userAgent      = "go-contextforge/v" + Version
	mediaTypeJSON  = "application/json"
)

// NewClient returns a new ContextForge API client. If a nil httpClient is
// provided, a new http.Client will be used. The bearerToken parameter is required
// for API authentication and should be a valid JWT token.
func NewClient(httpClient *http.Client, bearerToken string) *Client {
	baseURL, _ := url.Parse(defaultBaseURL)
	return newClient(httpClient, baseURL, bearerToken)
}

// NewClientWithBaseURL returns a new ContextForge API client with a custom base URL.
// If a nil httpClient is provided, a new http.Client will be used. The bearerToken
// parameter is required for API authentication and should be a valid JWT token.
// The baseURL must have a trailing slash.
func NewClientWithBaseURL(httpClient *http.Client, baseURL string, bearerToken string) (*Client, error) {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL; %w", err)
	}

	return newClient(httpClient, parsedURL, bearerToken), nil
}

// newClient initializes a new client with the given parameters.
func newClient(httpClient *http.Client, baseURL *url.URL, bearerToken string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	c := &Client{
		client:      httpClient,
		BaseURL:     baseURL,
		UserAgent:   userAgent,
		BearerToken: bearerToken,
		rateLimits:  make(map[string]Rate),
	}

	c.common.client = c
	c.Tools = (*ToolsService)(&c.common)
	c.Resources = (*ResourcesService)(&c.common)
	c.Gateways = (*GatewaysService)(&c.common)
	c.Servers = (*ServersService)(&c.common)
	c.Prompts = (*PromptsService)(&c.common)
	c.Agents = (*AgentsService)(&c.common)

	return c
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client.
// Relative URLs should always be specified without a preceding slash. If
// specified, the value pointed to by body is JSON encoded and included as the
// request body.
func (c *Client) NewRequest(method, urlStr string, body any) (*http.Request, error) {
	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, fmt.Errorf("BaseURL must have a trailing slash, but %q does not", c.BaseURL)
	}

	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", mediaTypeJSON)
	}
	req.Header.Set("Accept", mediaTypeJSON)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	}

	return req, nil
}

// newResponse creates a new Response for the provided http.Response.
// The Response is returned along with any error encountered while
// parsing rate limit headers and pagination cursor.
func newResponse(r *http.Response) *Response {
	response := &Response{Response: r}
	response.Rate = parseRate(r)
	response.NextCursor = parseCursor(r)
	return response
}

// parseRate parses rate limit headers from the response.
func parseRate(r *http.Response) Rate {
	var rate Rate
	if limit := r.Header.Get("X-RateLimit-Limit"); limit != "" {
		fmt.Sscanf(limit, "%d", &rate.Limit)
	}
	if remaining := r.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &rate.Remaining)
	}
	if reset := r.Header.Get("X-RateLimit-Reset"); reset != "" {
		if v, _ := time.Parse(time.RFC3339, reset); !v.IsZero() {
			rate.Reset = v
		}
	}
	return rate
}

// parseCursor parses pagination cursor from the response.
func parseCursor(r *http.Response) string {
	return r.Header.Get("X-Next-Cursor")
}

// Do sends an API request and returns the API response. The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred. If v implements the io.Writer interface,
// the raw response body will be written to v, without attempting to first
// decode it.
//
// The provided ctx must be non-nil. If it is canceled or times out,
// ctx.Err() will be returned.
func (c *Client) Do(ctx context.Context, req *http.Request, v any) (*Response, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context must be non-nil")
	}

	req = req.WithContext(ctx)

	c.clientMu.Lock()
	resp, err := c.client.Do(req)
	c.clientMu.Unlock()
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		return nil, err
	}
	defer resp.Body.Close()

	response := newResponse(resp)

	// Store rate limit information
	c.rateMu.Lock()
	c.rateLimits[req.URL.Path] = response.Rate
	c.rateMu.Unlock()

	err = CheckResponse(resp)
	if err != nil {
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			decErr := json.NewDecoder(resp.Body).Decode(v)
			if decErr == io.EOF {
				decErr = nil // ignore EOF errors caused by empty response body
			}
			if decErr != nil {
				err = decErr
			}
		}
	}

	return response, err
}

// addOptions adds the parameters in opts as URL query parameters to s.
// opts must be a struct whose fields may contain "url" tags.
func addOptions(s string, opts any) (string, error) {
	v, err := query.Values(opts)
	if err != nil {
		return s, err
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	if q := v.Encode(); q != "" {
		if u.RawQuery != "" {
			u.RawQuery = u.RawQuery + "&" + q
		} else {
			u.RawQuery = q
		}
	}

	return u.String(), nil
}
