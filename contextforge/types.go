package contextforge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Timestamp represents a time that can be unmarshalled from the ContextForge API.
// The API returns timestamps without timezone information, so we need custom parsing.
type Timestamp struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler for Timestamp.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if s == "" {
		return nil
	}

	// Try parsing with timezone first (RFC3339)
	parsed, err := time.Parse(time.RFC3339, s)
	if err == nil {
		t.Time = parsed
		return nil
	}

	// Try parsing without timezone (ContextForge format)
	parsed, err = time.Parse("2006-01-02T15:04:05.999999", s)
	if err == nil {
		t.Time = parsed
		return nil
	}

	// Try parsing without microseconds
	parsed, err = time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		return err
	}

	t.Time = parsed
	return nil
}

// MarshalJSON implements json.Marshaler for Timestamp.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Format(time.RFC3339))
}

// FlexibleID represents an ID that can be either a string or integer from the API.
// The ContextForge API inconsistently returns IDs as integers in some responses (e.g., CREATE)
// and as strings in others (e.g., GET). This type handles both cases.
type FlexibleID string

// UnmarshalJSON handles both string and integer ID values from the API.
func (f *FlexibleID) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleID(s)
		return nil
	}

	// If that fails, try as integer
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*f = FlexibleID(fmt.Sprintf("%d", i))
		return nil
	}

	return fmt.Errorf("id must be string or integer")
}

// String returns the string representation of the ID.
func (f FlexibleID) String() string {
	return string(f)
}

// Client manages communication with the ContextForge MCP Gateway API.
type Client struct {
	clientMu sync.Mutex   // protects the client during calls
	client   *http.Client // HTTP client used to communicate with the API

	// Base URL for API requests.
	// Defaults to http://localhost:8000/, but can be
	// overridden to point to another ContextForge instance.
	BaseURL *url.URL

	// User agent used when communicating with the ContextForge API.
	UserAgent string

	// Bearer token (JWT) for API authentication
	BearerToken string

	common service // Reuse a single struct instead of allocating one for each service

	// Services used for talking to different parts of the ContextForge API
	Tools     *ToolsService
	Resources *ResourcesService
	Gateways  *GatewaysService
	Servers   *ServersService
	Prompts   *PromptsService

	// Rate limit tracking
	rateMu     sync.Mutex
	rateLimits map[string]Rate
}

// service provides a general service interface for the API.
type service struct {
	client *Client
}

// ToolsService handles communication with the tool related
// methods of the ContextForge API.
type ToolsService service

// ResourcesService handles communication with the resource related
// methods of the ContextForge API.
type ResourcesService service

// GatewaysService handles communication with the gateway related
// methods of the ContextForge API.
type GatewaysService service

// ServersService handles communication with the server related
// methods of the ContextForge API.
type ServersService service

// PromptsService handles communication with the prompt related
// methods of the ContextForge API.
type PromptsService service

// Response wraps the standard http.Response and provides convenient access to
// pagination and rate limit information.
type Response struct {
	*http.Response

	// Pagination cursor extracted from response
	NextCursor string

	// Rate limiting information
	Rate Rate
}

// Rate represents the rate limit information returned in API responses.
type Rate struct {
	// The maximum number of requests that can be made in the current window.
	Limit int

	// The number of requests remaining in the current window.
	Remaining int

	// The time at which the current rate limit window resets.
	Reset time.Time
}

// ListOptions specifies the optional parameters to various List methods that
// support pagination.
type ListOptions struct {
	// Limit specifies the maximum number of items to return.
	// The API may return fewer than this value.
	Limit int `url:"limit,omitempty"`

	// Cursor is an opaque string used for pagination.
	// To get the next page of results, pass the NextCursor from the
	// previous response.
	Cursor string `url:"cursor,omitempty"`
}

// Tool represents a ContextForge tool.
type Tool struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	Enabled     bool           `json:"enabled,omitempty"`
	TeamID      *string        `json:"teamId,omitempty"`
	Visibility  string         `json:"visibility,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	CreatedAt   *Timestamp     `json:"createdAt,omitempty"`
	UpdatedAt   *Timestamp     `json:"updatedAt,omitempty"`
}

// ToolListOptions specifies the optional parameters to the
// ToolsService.List method.
type ToolListOptions struct {
	ListOptions

	// IncludeInactive includes inactive tools in the results
	IncludeInactive bool `url:"include_inactive,omitempty"`

	// Tags filters tools by tags (comma-separated)
	Tags string `url:"tags,omitempty"`

	// TeamID filters tools by team ID
	TeamID string `url:"team_id,omitempty"`

	// Visibility filters tools by visibility (public, private, etc.)
	Visibility string `url:"visibility,omitempty"`
}

// ToolCreateOptions specifies additional options for creating a tool.
// These fields are placed at the top level of the request wrapper.
type ToolCreateOptions struct {
	TeamID     *string
	Visibility *string
}

// Resource represents a ContextForge resource (read response).
type Resource struct {
	// Core fields
	ID          *FlexibleID      `json:"id,omitempty"`
	URI         string           `json:"uri"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	MimeType    *string          `json:"mimeType,omitempty"`
	Size        *int             `json:"size,omitempty"`
	IsActive    bool             `json:"isActive"`
	Metrics     *ResourceMetrics `json:"metrics,omitempty"`

	// Organizational fields
	Tags       []string `json:"tags,omitempty"`
	TeamID     *string  `json:"teamId,omitempty"`
	Team       *string  `json:"team,omitempty"`
	OwnerEmail *string  `json:"ownerEmail,omitempty"`
	Visibility *string  `json:"visibility,omitempty"`

	// Timestamps
	CreatedAt *Timestamp `json:"createdAt,omitempty"`
	UpdatedAt *Timestamp `json:"updatedAt,omitempty"`

	// Metadata fields (read-only)
	CreatedBy         *string `json:"createdBy,omitempty"`
	CreatedFromIP     *string `json:"createdFromIp,omitempty"`
	CreatedVia        *string `json:"createdVia,omitempty"`
	CreatedUserAgent  *string `json:"createdUserAgent,omitempty"`
	ModifiedBy        *string `json:"modifiedBy,omitempty"`
	ModifiedFromIP    *string `json:"modifiedFromIp,omitempty"`
	ModifiedVia       *string `json:"modifiedVia,omitempty"`
	ModifiedUserAgent *string `json:"modifiedUserAgent,omitempty"`
	ImportBatchID     *string `json:"importBatchId,omitempty"`
	FederationSource  *string `json:"federationSource,omitempty"`
	Version           *int    `json:"version,omitempty"`
}

// ResourceMetrics represents performance statistics for a resource.
type ResourceMetrics struct {
	TotalExecutions      int        `json:"totalExecutions,omitempty"`
	SuccessfulExecutions int        `json:"successfulExecutions,omitempty"`
	FailedExecutions     int        `json:"failedExecutions,omitempty"`
	FailureRate          float64    `json:"failureRate,omitempty"`
	MinResponseTime      *float64   `json:"minResponseTime,omitempty"`
	MaxResponseTime      *float64   `json:"maxResponseTime,omitempty"`
	AvgResponseTime      *float64   `json:"avgResponseTime,omitempty"`
	LastExecutionTime    *Timestamp `json:"lastExecutionTime,omitempty"`
}

// ResourceCreate represents the request body for creating a resource.
// Note: Uses snake_case field names as required by the API.
type ResourceCreate struct {
	// Required fields
	URI     string `json:"uri"`
	Name    string `json:"name"`
	Content any    `json:"content"` // Can be string or binary data

	// Optional fields (snake_case per API spec)
	Description *string  `json:"description,omitempty"`
	MimeType    *string  `json:"mime_type,omitempty"`
	Template    *string  `json:"template,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// ResourceUpdate represents the request body for updating a resource.
// Note: Uses camelCase field names as required by the API.
type ResourceUpdate struct {
	// All fields optional (camelCase per API spec)
	URI         *string  `json:"uri,omitempty"`
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	MimeType    *string  `json:"mimeType,omitempty"`
	Template    *string  `json:"template,omitempty"`
	Content     any      `json:"content,omitempty"` // Can be string or binary data
	Tags        []string `json:"tags,omitempty"`
}

// ResourceCreateOptions specifies additional options for creating a resource.
// These fields are placed at the top level of the request wrapper.
type ResourceCreateOptions struct {
	TeamID     *string
	Visibility *string
}

// ResourceListOptions specifies the optional parameters to the
// ResourcesService.List method.
type ResourceListOptions struct {
	ListOptions

	// IncludeInactive includes inactive resources in the results
	IncludeInactive bool `url:"include_inactive,omitempty"`

	// Tags filters resources by tags (comma-separated)
	Tags string `url:"tags,omitempty"`

	// TeamID filters resources by team ID
	TeamID string `url:"team_id,omitempty"`

	// Visibility filters resources by visibility (public, private, etc.)
	Visibility string `url:"visibility,omitempty"`
}

// ListResourceTemplatesResult represents the response from listing resource templates.
type ListResourceTemplatesResult struct {
	Templates []ResourceTemplate `json:"templates"`
}

// ResourceTemplate represents a template for creating resources.
type ResourceTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URI         string `json:"uri"`
	MimeType    string `json:"mime_type"`
}

// Gateway represents a ContextForge gateway.
type Gateway struct {
	// Core fields
	ID          *string    `json:"id,omitempty"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Description *string    `json:"description,omitempty"`
	Transport   string     `json:"transport,omitempty"`
	Enabled     bool       `json:"enabled,omitempty"`
	Reachable   bool       `json:"reachable,omitempty"`
	Capabilities map[string]any `json:"capabilities,omitempty"`

	// Authentication fields
	PassthroughHeaders []string           `json:"passthroughHeaders,omitempty"`
	AuthType           *string            `json:"authType,omitempty"`
	AuthUsername       *string            `json:"authUsername,omitempty"`
	AuthPassword       *string            `json:"authPassword,omitempty"`
	AuthToken          *string            `json:"authToken,omitempty"`
	AuthHeaderKey      *string            `json:"authHeaderKey,omitempty"`
	AuthHeaderValue    *string            `json:"authHeaderValue,omitempty"`
	AuthHeaders        []map[string]string `json:"authHeaders,omitempty"`
	AuthValue          *string            `json:"authValue,omitempty"`
	OAuthConfig        map[string]any     `json:"oauthConfig,omitempty"`

	// Organizational fields
	Tags       []string `json:"tags,omitempty"`
	TeamID     *string  `json:"teamId,omitempty"`
	Team       *string  `json:"team,omitempty"`
	OwnerEmail *string  `json:"ownerEmail,omitempty"`
	Visibility *string  `json:"visibility,omitempty"`

	// Timestamps
	CreatedAt *Timestamp `json:"createdAt,omitempty"`
	UpdatedAt *Timestamp `json:"updatedAt,omitempty"`
	LastSeen  *Timestamp `json:"lastSeen,omitempty"`

	// Metadata fields (read-only)
	CreatedBy         *string `json:"createdBy,omitempty"`
	CreatedFromIP     *string `json:"createdFromIp,omitempty"`
	CreatedVia        *string `json:"createdVia,omitempty"`
	CreatedUserAgent  *string `json:"createdUserAgent,omitempty"`
	ModifiedBy        *string `json:"modifiedBy,omitempty"`
	ModifiedFromIP    *string `json:"modifiedFromIp,omitempty"`
	ModifiedVia       *string `json:"modifiedVia,omitempty"`
	ModifiedUserAgent *string `json:"modifiedUserAgent,omitempty"`
	ImportBatchID     *string `json:"importBatchId,omitempty"`
	FederationSource  *string `json:"federationSource,omitempty"`
	Version           *int    `json:"version,omitempty"`
	Slug              *string `json:"slug,omitempty"`
}

// GatewayListOptions specifies the optional parameters to the
// GatewaysService.List method.
type GatewayListOptions struct {
	ListOptions

	// IncludeInactive includes inactive gateways in the results
	IncludeInactive bool `url:"include_inactive,omitempty"`
}

// GatewayCreateOptions specifies additional options for creating a gateway.
// These fields are placed at the top level of the request wrapper.
type GatewayCreateOptions struct {
	TeamID     *string
	Visibility *string
}

// Server represents a ContextForge server (read response).
type Server struct {
	// Core fields
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Icon        *string        `json:"icon,omitempty"`
	IsActive    bool           `json:"isActive,omitempty"`
	Metrics     *ServerMetrics `json:"metrics,omitempty"`

	// Association fields
	AssociatedTools     []string `json:"associatedTools,omitempty"`
	AssociatedResources []int    `json:"associatedResources,omitempty"`
	AssociatedPrompts   []int    `json:"associatedPrompts,omitempty"`
	AssociatedA2aAgents []string `json:"associatedA2aAgents,omitempty"`

	// Organizational fields
	Tags       []string `json:"tags,omitempty"`
	TeamID     *string  `json:"teamId,omitempty"`
	Team       *string  `json:"team,omitempty"`
	OwnerEmail *string  `json:"ownerEmail,omitempty"`
	Visibility *string  `json:"visibility,omitempty"`

	// Timestamps
	CreatedAt *Timestamp `json:"createdAt,omitempty"`
	UpdatedAt *Timestamp `json:"updatedAt,omitempty"`

	// Metadata fields (read-only)
	CreatedBy         *string `json:"createdBy,omitempty"`
	CreatedFromIP     *string `json:"createdFromIp,omitempty"`
	CreatedVia        *string `json:"createdVia,omitempty"`
	CreatedUserAgent  *string `json:"createdUserAgent,omitempty"`
	ModifiedBy        *string `json:"modifiedBy,omitempty"`
	ModifiedFromIP    *string `json:"modifiedFromIp,omitempty"`
	ModifiedVia       *string `json:"modifiedVia,omitempty"`
	ModifiedUserAgent *string `json:"modifiedUserAgent,omitempty"`
	ImportBatchID     *string `json:"importBatchId,omitempty"`
	FederationSource  *string `json:"federationSource,omitempty"`
	Version           *int    `json:"version,omitempty"`
}

// ServerMetrics represents performance statistics for a server.
type ServerMetrics struct {
	TotalExecutions      int        `json:"totalExecutions"`
	SuccessfulExecutions int        `json:"successfulExecutions"`
	FailedExecutions     int        `json:"failedExecutions"`
	FailureRate          float64    `json:"failureRate"`
	MinResponseTime      *float64   `json:"minResponseTime,omitempty"`
	MaxResponseTime      *float64   `json:"maxResponseTime,omitempty"`
	AvgResponseTime      *float64   `json:"avgResponseTime,omitempty"`
	LastExecutionTime    *Timestamp `json:"lastExecutionTime,omitempty"`
}

// ServerCreate represents the request body for creating a server.
// Note: Uses snake_case field names as required by the API.
type ServerCreate struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Icon        *string  `json:"icon,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Association fields (snake_case per API spec)
	AssociatedTools     []string `json:"associated_tools,omitempty"`
	AssociatedResources []string `json:"associated_resources,omitempty"`
	AssociatedPrompts   []string `json:"associated_prompts,omitempty"`
	AssociatedA2aAgents []string `json:"associated_a2a_agents,omitempty"`

	// Organizational fields (snake_case per API spec)
	TeamID     *string `json:"team_id,omitempty"`
	OwnerEmail *string `json:"owner_email,omitempty"`
	Visibility *string `json:"visibility,omitempty"`
}

// ServerUpdate represents the request body for updating a server.
// Note: Uses camelCase field names as required by the API.
type ServerUpdate struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Icon        *string  `json:"icon,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Association fields (camelCase per API spec)
	AssociatedTools     []string `json:"associatedTools,omitempty"`
	AssociatedResources []string `json:"associatedResources,omitempty"`
	AssociatedPrompts   []string `json:"associatedPrompts,omitempty"`
	AssociatedA2aAgents []string `json:"associatedA2aAgents,omitempty"`

	// Organizational fields (camelCase per API spec)
	TeamID     *string `json:"teamId,omitempty"`
	OwnerEmail *string `json:"ownerEmail,omitempty"`
	Visibility *string `json:"visibility,omitempty"`
}

// ServerListOptions specifies the optional parameters to the
// ServersService.List method.
type ServerListOptions struct {
	ListOptions

	// IncludeInactive includes inactive servers in the results
	IncludeInactive bool `url:"include_inactive,omitempty"`

	// Tags filters servers by tags (comma-separated)
	Tags string `url:"tags,omitempty"`

	// TeamID filters servers by team ID
	TeamID string `url:"team_id,omitempty"`

	// Visibility filters servers by visibility (public, private, etc.)
	Visibility string `url:"visibility,omitempty"`
}

// ServerCreateOptions specifies additional options for creating a server.
// These fields are placed at the top level of the request wrapper.
type ServerCreateOptions struct {
	TeamID     *string
	Visibility *string
}

// ServerAssociationOptions specifies the optional parameters for listing
// server associations (tools, resources, prompts).
type ServerAssociationOptions struct {
	// IncludeInactive includes inactive items in the results
	IncludeInactive bool `url:"include_inactive,omitempty"`
}

// Prompt represents a ContextForge prompt (read response).
// Note: These types are shared between ServersService and the future PromptsService.
type Prompt struct {
	// Core fields
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Template    string            `json:"template"`
	Arguments   []PromptArgument  `json:"arguments"`
	CreatedAt   *Timestamp        `json:"createdAt,omitempty"`
	UpdatedAt   *Timestamp        `json:"updatedAt,omitempty"`
	IsActive    bool              `json:"isActive"`
	Tags        []string          `json:"tags,omitempty"`
	Metrics     *PromptMetrics    `json:"metrics,omitempty"`

	// Organizational fields
	TeamID     *string `json:"teamId,omitempty"`
	Team       *string `json:"team,omitempty"`
	OwnerEmail *string `json:"ownerEmail,omitempty"`
	Visibility *string `json:"visibility,omitempty"`

	// Metadata fields (read-only)
	CreatedBy         *string `json:"createdBy,omitempty"`
	CreatedFromIP     *string `json:"createdFromIp,omitempty"`
	CreatedVia        *string `json:"createdVia,omitempty"`
	CreatedUserAgent  *string `json:"createdUserAgent,omitempty"`
	ModifiedBy        *string `json:"modifiedBy,omitempty"`
	ModifiedFromIP    *string `json:"modifiedFromIp,omitempty"`
	ModifiedVia       *string `json:"modifiedVia,omitempty"`
	ModifiedUserAgent *string `json:"modifiedUserAgent,omitempty"`
	ImportBatchID     *string `json:"importBatchId,omitempty"`
	FederationSource  *string `json:"federationSource,omitempty"`
	Version           *int    `json:"version,omitempty"`
}

// PromptArgument represents a parameter definition for a prompt.
type PromptArgument struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
}

// PromptMetrics represents performance statistics for a prompt.
type PromptMetrics struct {
	TotalExecutions      int        `json:"totalExecutions"`
	SuccessfulExecutions int        `json:"successfulExecutions"`
	FailedExecutions     int        `json:"failedExecutions"`
	FailureRate          float64    `json:"failureRate"`
	MinResponseTime      *float64   `json:"minResponseTime,omitempty"`
	MaxResponseTime      *float64   `json:"maxResponseTime,omitempty"`
	AvgResponseTime      *float64   `json:"avgResponseTime,omitempty"`
	LastExecutionTime    *Timestamp `json:"lastExecutionTime,omitempty"`
}

// PromptCreate represents the request body for creating a prompt.
// Note: Uses snake_case field names as required by the API.
type PromptCreate struct {
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Template    string           `json:"template"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
	Tags        []string         `json:"tags,omitempty"`

	// Organizational fields (snake_case per API spec)
	TeamID     *string `json:"team_id,omitempty"`
	OwnerEmail *string `json:"owner_email,omitempty"`
	Visibility *string `json:"visibility,omitempty"`
}

// PromptUpdate represents the request body for updating a prompt.
// Note: Uses camelCase field names as required by the API.
type PromptUpdate struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Template    *string          `json:"template,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
	Tags        []string         `json:"tags,omitempty"`

	// Organizational fields (camelCase per API spec)
	TeamID     *string `json:"teamId,omitempty"`
	OwnerEmail *string `json:"ownerEmail,omitempty"`
	Visibility *string `json:"visibility,omitempty"`
}

// PromptListOptions specifies the optional parameters to the
// PromptsService.List method.
type PromptListOptions struct {
	ListOptions

	// IncludeInactive includes inactive prompts in the results
	IncludeInactive bool `url:"include_inactive,omitempty"`

	// Tags filters prompts by tags (comma-separated)
	Tags string `url:"tags,omitempty"`

	// TeamID filters prompts by team ID
	TeamID string `url:"team_id,omitempty"`

	// Visibility filters prompts by visibility (public, private, etc.)
	Visibility string `url:"visibility,omitempty"`
}

// PromptCreateOptions specifies additional options for creating a prompt.
// These fields are placed at the top level of the request wrapper.
type PromptCreateOptions struct {
	TeamID     *string
	Visibility *string
}
