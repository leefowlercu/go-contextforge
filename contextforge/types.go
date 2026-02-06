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

	// Address for API requests.
	// Defaults to http://localhost:8000/, but can be
	// overridden to point to another ContextForge instance.
	Address *url.URL

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
	Agents    *AgentsService
	Teams     *TeamsService
	Cancel    *CancellationService

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

// AgentsService handles communication with the A2A agent related
// methods of the ContextForge API.
type AgentsService service

// TeamsService handles communication with the team related
// methods of the ContextForge API.
type TeamsService service

// CancellationService handles communication with the cancellation related
// methods of the ContextForge API.
type CancellationService service

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

	// IncludePagination requests body-based pagination metadata in API responses.
	// When true, list endpoints return an object with items and nextCursor fields.
	IncludePagination bool `url:"include_pagination,omitempty"`
}

// Tag represents a tag that can be unmarshaled from either a string or an object.
// In v1.0.0+, the API returns tags as objects with id and label fields,
// but accepts simple strings as input. This type handles both formats.
//
// Example input (create/update): ["tag1", "tag2"]
// Example output (read): [{"id":"tag1","label":"tag1"}, {"id":"tag2","label":"tag2"}]
type Tag struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// String returns the tag name (ID).
func (t Tag) String() string {
	return t.ID
}

// UnmarshalJSON handles both string and object formats for tags.
// This allows seamless parsing of both old string format and new object format.
func (t *Tag) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.ID = s
		t.Label = s
		return nil
	}

	// Fall back to object format
	type tagAlias Tag // prevent recursion
	var obj tagAlias
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*t = Tag(obj)
	return nil
}

// MarshalJSON outputs tags as strings for API input compatibility.
func (t Tag) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.ID)
}

// NewTag creates a new Tag from a string.
func NewTag(name string) Tag {
	return Tag{ID: name, Label: name}
}

// NewTags creates a slice of Tags from a slice of strings.
func NewTags(names []string) []Tag {
	if names == nil {
		return nil
	}
	tags := make([]Tag, len(names))
	for i, name := range names {
		tags[i] = NewTag(name)
	}
	return tags
}

// TagNames returns the tag names (IDs) from a slice of Tag objects.
// This is a convenience method for getting just the tag names.
func TagNames(tags []Tag) []string {
	if tags == nil {
		return nil
	}
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.ID
	}
	return names
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
	Tags        []Tag          `json:"tags,omitempty"`
	CreatedAt   *Timestamp     `json:"createdAt,omitempty"`
	UpdatedAt   *Timestamp     `json:"updatedAt,omitempty"`

	// Additional fields added in v1.0.0
	Team       *string `json:"team,omitempty"`
	OwnerEmail *string `json:"ownerEmail,omitempty"`

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
	Enabled     bool             `json:"enabled,omitempty"`
	Metrics     *ResourceMetrics `json:"metrics,omitempty"`

	// Organizational fields
	Tags       []Tag   `json:"tags,omitempty"`
	TeamID     *string `json:"teamId,omitempty"`
	Team       *string `json:"team,omitempty"`
	OwnerEmail *string `json:"ownerEmail,omitempty"`
	Visibility *string `json:"visibility,omitempty"`

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
//
// All fields are optional. The SDK uses a three-state semantics pattern:
//   - nil pointer/slice: field will not be updated (omitted from request)
//   - pointer to zero value or empty slice: field will be cleared/set to empty
//   - pointer to value or populated slice: field will be set to that value
//
// Examples:
//   - Don't update tags: Tags = nil
//   - Clear all tags: Tags = []string{}
//   - Set specific tags: Tags = []string{"tag1", "tag2"}
//
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
	ID           *string        `json:"id,omitempty"`
	Name         string         `json:"name"`
	URL          string         `json:"url"`
	Description  *string        `json:"description,omitempty"`
	Transport    string         `json:"transport,omitempty"`
	Enabled      bool           `json:"enabled,omitempty"`
	Reachable    bool           `json:"reachable,omitempty"`
	Capabilities map[string]any `json:"capabilities,omitempty"`

	// Authentication fields
	PassthroughHeaders        []string            `json:"passthroughHeaders,omitempty"`
	AuthType                  *string             `json:"authType,omitempty"`
	AuthUsername              *string             `json:"authUsername,omitempty"`
	AuthPassword              *string             `json:"authPassword,omitempty"`
	AuthToken                 *string             `json:"authToken,omitempty"`
	AuthHeaderKey             *string             `json:"authHeaderKey,omitempty"`
	AuthHeaderValue           *string             `json:"authHeaderValue,omitempty"`
	AuthHeaders               []map[string]string `json:"authHeaders,omitempty"`
	AuthValue                 *string             `json:"authValue,omitempty"`
	OAuthConfig               map[string]any      `json:"oauthConfig,omitempty"`
	AuthQueryParamKey         *string             `json:"authQueryParamKey,omitempty"`
	AuthQueryParamValue       *string             `json:"authQueryParamValue,omitempty"`
	AuthQueryParamValueMasked *string             `json:"authQueryParamValueMasked,omitempty"`

	// Organizational fields
	Tags       []Tag   `json:"tags,omitempty"`
	TeamID     *string `json:"teamId,omitempty"`
	Team       *string `json:"team,omitempty"`
	OwnerEmail *string `json:"ownerEmail,omitempty"`
	Visibility *string `json:"visibility,omitempty"`

	// Timestamps
	CreatedAt *Timestamp `json:"createdAt,omitempty"`
	UpdatedAt *Timestamp `json:"updatedAt,omitempty"`
	LastSeen  *Timestamp `json:"lastSeen,omitempty"`

	// Metadata fields (read-only)
	CreatedBy              *string    `json:"createdBy,omitempty"`
	CreatedFromIP          *string    `json:"createdFromIp,omitempty"`
	CreatedVia             *string    `json:"createdVia,omitempty"`
	CreatedUserAgent       *string    `json:"createdUserAgent,omitempty"`
	ModifiedBy             *string    `json:"modifiedBy,omitempty"`
	ModifiedFromIP         *string    `json:"modifiedFromIp,omitempty"`
	ModifiedVia            *string    `json:"modifiedVia,omitempty"`
	ModifiedUserAgent      *string    `json:"modifiedUserAgent,omitempty"`
	ImportBatchID          *string    `json:"importBatchId,omitempty"`
	FederationSource       *string    `json:"federationSource,omitempty"`
	Version                *int       `json:"version,omitempty"`
	Slug                   *string    `json:"slug,omitempty"`
	RefreshIntervalSeconds *int       `json:"refreshIntervalSeconds,omitempty"`
	LastRefreshAt          *Timestamp `json:"lastRefreshAt,omitempty"`
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
	Enabled     bool           `json:"enabled,omitempty"`
	Metrics     *ServerMetrics `json:"metrics,omitempty"`

	// Association fields
	AssociatedTools     []string `json:"associatedTools,omitempty"`
	AssociatedResources []string `json:"associatedResources,omitempty"`
	AssociatedPrompts   []string `json:"associatedPrompts,omitempty"`
	AssociatedA2aAgents []string `json:"associatedA2aAgents,omitempty"`

	// Organizational fields
	Tags       []Tag   `json:"tags,omitempty"`
	TeamID     *string `json:"teamId,omitempty"`
	Team       *string `json:"team,omitempty"`
	OwnerEmail *string `json:"ownerEmail,omitempty"`
	Visibility *string `json:"visibility,omitempty"`

	// Timestamps
	CreatedAt *Timestamp `json:"createdAt,omitempty"`
	UpdatedAt *Timestamp `json:"updatedAt,omitempty"`

	// Metadata fields (read-only)
	CreatedBy         *string        `json:"createdBy,omitempty"`
	CreatedFromIP     *string        `json:"createdFromIp,omitempty"`
	CreatedVia        *string        `json:"createdVia,omitempty"`
	CreatedUserAgent  *string        `json:"createdUserAgent,omitempty"`
	ModifiedBy        *string        `json:"modifiedBy,omitempty"`
	ModifiedFromIP    *string        `json:"modifiedFromIp,omitempty"`
	ModifiedVia       *string        `json:"modifiedVia,omitempty"`
	ModifiedUserAgent *string        `json:"modifiedUserAgent,omitempty"`
	ImportBatchID     *string        `json:"importBatchId,omitempty"`
	FederationSource  *string        `json:"federationSource,omitempty"`
	Version           *int           `json:"version,omitempty"`
	OAuthEnabled      bool           `json:"oauthEnabled,omitempty"`
	OAuthConfig       map[string]any `json:"oauthConfig,omitempty"`
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
//
// All fields are optional. The SDK uses a three-state semantics pattern:
//   - nil pointer/slice: field will not be updated (omitted from request)
//   - pointer to zero value or empty slice: field will be cleared/set to empty
//   - pointer to value or populated slice: field will be set to that value
//
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
	// Note: ID changed from int to string in v1.0.0
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	OriginalName   *string          `json:"originalName,omitempty"`
	CustomName     *string          `json:"customName,omitempty"`
	CustomNameSlug *string          `json:"customNameSlug,omitempty"`
	DisplayName    *string          `json:"displayName,omitempty"`
	GatewaySlug    *string          `json:"gatewaySlug,omitempty"`
	Description    *string          `json:"description,omitempty"`
	Template       string           `json:"template"`
	Arguments      []PromptArgument `json:"arguments"`
	CreatedAt      *Timestamp       `json:"createdAt,omitempty"`
	UpdatedAt      *Timestamp       `json:"updatedAt,omitempty"`
	IsActive       bool             `json:"isActive"`
	Enabled        bool             `json:"enabled,omitempty"` // v1.0.0 uses 'enabled' in addition to 'isActive'
	Tags           []Tag            `json:"tags,omitempty"`
	Metrics        *PromptMetrics   `json:"metrics,omitempty"`

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
	CustomName  *string          `json:"custom_name,omitempty"`
	DisplayName *string          `json:"display_name,omitempty"`
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
//
// All fields are optional. The SDK uses a three-state semantics pattern:
//   - nil pointer/slice: field will not be updated (omitted from request)
//   - pointer to zero value or empty slice: field will be cleared/set to empty
//   - pointer to value or populated slice: field will be set to that value
//
// Note: Uses camelCase field names as required by the API.
type PromptUpdate struct {
	Name        *string          `json:"name,omitempty"`
	CustomName  *string          `json:"customName,omitempty"`
	DisplayName *string          `json:"displayName,omitempty"`
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

// Agent represents an A2A (Agent-to-Agent) agent in the ContextForge API.
// A2A agents enable inter-agent communication using the ContextForge A2A protocol.
type Agent struct {
	// Core fields
	ID                        string         `json:"id"`
	Name                      string         `json:"name"`
	Slug                      string         `json:"slug"`
	Description               *string        `json:"description,omitempty"`
	EndpointURL               string         `json:"endpointUrl"`
	AgentType                 string         `json:"agentType"`
	ProtocolVersion           string         `json:"protocolVersion"`
	Capabilities              map[string]any `json:"capabilities,omitempty"`
	Config                    map[string]any `json:"config,omitempty"`
	AuthType                  *string        `json:"authType,omitempty"`
	OAuthConfig               map[string]any `json:"oauthConfig,omitempty"`
	AuthQueryParamKey         *string        `json:"authQueryParamKey,omitempty"`
	AuthQueryParamValueMasked *string        `json:"authQueryParamValueMasked,omitempty"`
	Enabled                   bool           `json:"enabled"`
	Reachable                 bool           `json:"reachable"`

	// Timestamps
	CreatedAt       *Timestamp `json:"createdAt,omitempty"`
	UpdatedAt       *Timestamp `json:"updatedAt,omitempty"`
	LastInteraction *Timestamp `json:"lastInteraction,omitempty"`

	// Organizational fields
	Tags       []Tag         `json:"tags,omitempty"`
	Metrics    *AgentMetrics `json:"metrics,omitempty"`
	TeamID     *string       `json:"teamId,omitempty"`
	OwnerEmail *string       `json:"ownerEmail,omitempty"`
	Visibility *string       `json:"visibility,omitempty"`

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

// AgentMetrics represents performance statistics for an agent.
type AgentMetrics struct {
	TotalExecutions      int        `json:"totalExecutions"`
	SuccessfulExecutions int        `json:"successfulExecutions"`
	FailedExecutions     int        `json:"failedExecutions"`
	FailureRate          float64    `json:"failureRate"`
	MinResponseTime      *float64   `json:"minResponseTime,omitempty"`
	MaxResponseTime      *float64   `json:"maxResponseTime,omitempty"`
	AvgResponseTime      *float64   `json:"avgResponseTime,omitempty"`
	LastExecutionTime    *Timestamp `json:"lastExecutionTime,omitempty"`
}

// AgentCreate represents the request body for creating an A2A agent.
// Note: Uses snake_case field names as required by the API.
type AgentCreate struct {
	// Required fields
	Name        string `json:"name"`
	EndpointURL string `json:"endpoint_url"`

	// Optional core fields
	Slug            *string        `json:"slug,omitempty"`
	Description     *string        `json:"description,omitempty"`
	AgentType       string         `json:"agent_type,omitempty"`       // default: "generic"
	ProtocolVersion string         `json:"protocol_version,omitempty"` // default: "1.0"
	Capabilities    map[string]any `json:"capabilities,omitempty"`
	Config          map[string]any `json:"config,omitempty"`

	// Authentication fields
	AuthType            *string        `json:"auth_type,omitempty"`
	AuthValue           *string        `json:"auth_value,omitempty"` // Will be encrypted by API
	OAuthConfig         map[string]any `json:"oauth_config,omitempty"`
	AuthQueryParamKey   *string        `json:"auth_query_param_key,omitempty"`
	AuthQueryParamValue *string        `json:"auth_query_param_value,omitempty"`

	// Organizational fields (snake_case)
	Tags       []string `json:"tags,omitempty"`
	TeamID     *string  `json:"team_id,omitempty"`
	OwnerEmail *string  `json:"owner_email,omitempty"`
	Visibility *string  `json:"visibility,omitempty"` // default: "public"
}

// AgentUpdate represents the request body for updating an agent.
//
// All fields are optional. The SDK uses a three-state semantics pattern:
//   - nil pointer/slice: field will not be updated (omitted from request)
//   - pointer to zero value or empty slice: field will be cleared/set to empty
//   - pointer to value or populated slice: field will be set to that value
//
// Note: Uses camelCase field names as required by the API.
type AgentUpdate struct {
	// All fields optional (camelCase per API spec)
	Name                *string        `json:"name,omitempty"`
	Description         *string        `json:"description,omitempty"`
	EndpointURL         *string        `json:"endpointUrl,omitempty"`
	AgentType           *string        `json:"agentType,omitempty"`
	ProtocolVersion     *string        `json:"protocolVersion,omitempty"`
	Capabilities        map[string]any `json:"capabilities,omitempty"`
	Config              map[string]any `json:"config,omitempty"`
	AuthType            *string        `json:"authType,omitempty"`
	AuthValue           *string        `json:"authValue,omitempty"`
	OAuthConfig         map[string]any `json:"oauthConfig,omitempty"`
	AuthQueryParamKey   *string        `json:"authQueryParamKey,omitempty"`
	AuthQueryParamValue *string        `json:"authQueryParamValue,omitempty"`
	Tags                []string       `json:"tags,omitempty"`
	TeamID              *string        `json:"teamId,omitempty"`
	OwnerEmail          *string        `json:"ownerEmail,omitempty"`
	Visibility          *string        `json:"visibility,omitempty"`
}

// AgentListOptions specifies the optional parameters to the
// AgentsService.List method.
// Note: Upstream v1.0.0-BETA-2 supports cursor pagination. Skip remains
// available for backward compatibility.
type AgentListOptions struct {
	// Skip specifies the number of items to skip (offset)
	// Deprecated: Upstream v1.0.0-BETA-2 uses cursor pagination.
	Skip int `url:"skip,omitempty"`

	// Limit specifies the maximum number of items to return.
	Limit int `url:"limit,omitempty"`

	// Cursor is an opaque string used for pagination.
	Cursor string `url:"cursor,omitempty"`

	// IncludePagination requests body-based pagination metadata in responses.
	IncludePagination bool `url:"include_pagination,omitempty"`

	// IncludeInactive includes inactive agents in the results
	IncludeInactive bool `url:"include_inactive,omitempty"`

	// Tags filters agents by tags (comma-separated)
	Tags string `url:"tags,omitempty"`

	// TeamID filters agents by team ID
	TeamID string `url:"team_id,omitempty"`

	// Visibility filters agents by visibility (public, private, etc.)
	Visibility string `url:"visibility,omitempty"`
}

// AgentCreateOptions specifies additional options for creating an agent.
// These fields are placed at the top level of the request wrapper.
type AgentCreateOptions struct {
	TeamID     *string
	Visibility *string
}

// AgentInvokeRequest represents the request body for invoking an A2A agent.
type AgentInvokeRequest struct {
	Parameters      map[string]any `json:"parameters,omitempty"`
	InteractionType string         `json:"interaction_type,omitempty"` // default: "query"
}

// ResourceInfoOptions specifies optional parameters for ResourcesService.GetInfo.
type ResourceInfoOptions struct {
	IncludeInactive bool `url:"include_inactive,omitempty"`
}

// GatewayRefreshOptions specifies optional parameters for GatewaysService.RefreshTools.
type GatewayRefreshOptions struct {
	IncludeResources bool `url:"include_resources,omitempty"`
	IncludePrompts   bool `url:"include_prompts,omitempty"`
}

// GatewayRefreshResponse represents the response from manual gateway refresh.
type GatewayRefreshResponse struct {
	GatewayID        string     `json:"gateway_id"`
	Success          bool       `json:"success"`
	Error            *string    `json:"error,omitempty"`
	ToolsAdded       int        `json:"tools_added,omitempty"`
	ToolsUpdated     int        `json:"tools_updated,omitempty"`
	ToolsRemoved     int        `json:"tools_removed,omitempty"`
	ResourcesAdded   int        `json:"resources_added,omitempty"`
	ResourcesUpdated int        `json:"resources_updated,omitempty"`
	ResourcesRemoved int        `json:"resources_removed,omitempty"`
	PromptsAdded     int        `json:"prompts_added,omitempty"`
	PromptsUpdated   int        `json:"prompts_updated,omitempty"`
	PromptsRemoved   int        `json:"prompts_removed,omitempty"`
	ValidationErrors []string   `json:"validation_errors,omitempty"`
	DurationMS       float64    `json:"duration_ms,omitempty"`
	RefreshedAt      *Timestamp `json:"refreshed_at,omitempty"`
}

// CancellationRequest represents a cancellation request payload.
type CancellationRequest struct {
	RequestID string  `json:"requestId"`
	Reason    *string `json:"reason,omitempty"`
}

// CancellationResponse represents the response from cancellation requests.
type CancellationResponse struct {
	Status    string  `json:"status"`
	RequestID string  `json:"requestId"`
	Reason    *string `json:"reason,omitempty"`
}

// CancellationStatus represents cancellation status details for a run.
type CancellationStatus struct {
	Name         *string  `json:"name,omitempty"`
	RegisteredAt *float64 `json:"registered_at,omitempty"`
	Cancelled    bool     `json:"cancelled,omitempty"`
	CancelledAt  *float64 `json:"cancelled_at,omitempty"`
	CancelReason *string  `json:"cancel_reason,omitempty"`
}

// Team represents a ContextForge team.
type Team struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description *string    `json:"description,omitempty"`
	IsPersonal  bool       `json:"is_personal"`
	Visibility  *string    `json:"visibility,omitempty"`
	MaxMembers  *int       `json:"max_members,omitempty"`
	MemberCount int        `json:"member_count"`
	IsActive    bool       `json:"is_active"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   *Timestamp `json:"created_at,omitempty"`
	UpdatedAt   *Timestamp `json:"updated_at,omitempty"`
}

// TeamCreate represents the request body for creating a team.
type TeamCreate struct {
	Name        string  `json:"name"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`
	Visibility  *string `json:"visibility,omitempty"`
	MaxMembers  *int    `json:"max_members,omitempty"`
}

// TeamUpdate represents the request body for updating a team.
//
// All fields are optional. The SDK uses a three-state semantics pattern:
//   - nil pointer: field will not be updated (omitted from request)
//   - pointer to zero value: field will be cleared/set to empty
//   - pointer to value: field will be set to that value
type TeamUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Visibility  *string `json:"visibility,omitempty"`
	MaxMembers  *int    `json:"max_members,omitempty"`
}

// TeamListResponse represents the response from the list teams endpoint.
type TeamListResponse struct {
	Teams []*Team `json:"teams"`
	Total int     `json:"total"`
}

// TeamListOptions specifies the optional parameters for listing teams.
type TeamListOptions struct {
	// Skip specifies the number of items to skip (offset)
	Skip int `url:"skip,omitempty"`

	// Limit specifies the maximum number of items to return (max: 100, default: 50)
	Limit int `url:"limit,omitempty"`
}

// TeamMember represents a member of a team.
type TeamMember struct {
	ID        string     `json:"id"`
	TeamID    string     `json:"team_id"`
	UserEmail string     `json:"user_email"`
	Role      string     `json:"role"`
	JoinedAt  *Timestamp `json:"joined_at,omitempty"`
	InvitedBy *string    `json:"invited_by,omitempty"`
	IsActive  bool       `json:"is_active"`
}

// TeamMemberUpdate represents the request body for updating a team member's role.
type TeamMemberUpdate struct {
	Role string `json:"role"`
}

// TeamInvitation represents a team invitation.
type TeamInvitation struct {
	ID        string     `json:"id"`
	TeamID    string     `json:"team_id"`
	TeamName  string     `json:"team_name"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	InvitedBy string     `json:"invited_by"`
	InvitedAt *Timestamp `json:"invited_at,omitempty"`
	ExpiresAt *Timestamp `json:"expires_at,omitempty"`
	Token     string     `json:"token"`
	IsActive  bool       `json:"is_active"`
	IsExpired bool       `json:"is_expired"`
}

// TeamInvite represents the request body for inviting a user to a team.
type TeamInvite struct {
	Email string  `json:"email"`
	Role  *string `json:"role,omitempty"`
}

// TeamDiscovery represents a team in the discovery list.
type TeamDiscovery struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	MemberCount int        `json:"member_count"`
	CreatedAt   *Timestamp `json:"created_at,omitempty"`
	IsJoinable  bool       `json:"is_joinable"`
}

// TeamDiscoverOptions specifies the optional parameters for discovering teams.
type TeamDiscoverOptions struct {
	// Skip specifies the number of items to skip (offset)
	Skip int `url:"skip,omitempty"`

	// Limit specifies the maximum number of items to return (max: 100, default: 50)
	Limit int `url:"limit,omitempty"`
}

// TeamJoinRequest represents the request body for joining a team.
type TeamJoinRequest struct {
	Message *string `json:"message,omitempty"`
}

// TeamJoinRequestResponse represents a team join request.
type TeamJoinRequestResponse struct {
	ID          string     `json:"id"`
	TeamID      string     `json:"team_id"`
	TeamName    string     `json:"team_name"`
	UserEmail   string     `json:"user_email"`
	Message     *string    `json:"message,omitempty"`
	Status      string     `json:"status"`
	RequestedAt *Timestamp `json:"requested_at,omitempty"`
	ExpiresAt   *Timestamp `json:"expires_at,omitempty"`
}

// ResourceContent represents resource content in MCP-compatible format.
// Returned by the GET /resources/{id} hybrid endpoint.
type ResourceContent struct {
	Type     string  `json:"type"`               // Always "resource"
	URI      string  `json:"uri"`                // Resource URI
	MimeType *string `json:"mimeType,omitempty"` // MIME type (optional)
	Text     *string `json:"text,omitempty"`     // Text content (one of text/blob)
	Blob     *string `json:"blob,omitempty"`     // Binary content as string (one of text/blob)
}

// PromptGetArgs represents the request body for POST /prompts/{id}.
type PromptGetArgs struct {
	Args map[string]string `json:"args,omitempty"` // Template arguments
}

// PromptResult represents the result of getting a prompt with arguments.
// Returned by POST /prompts/{id} and GET /prompts/{id} hybrid endpoints.
type PromptResult struct {
	Description *string          `json:"description,omitempty"` // Optional description
	Messages    []*PromptMessage `json:"messages"`              // Rendered messages
}

// PromptMessage represents a message in a prompt result.
type PromptMessage struct {
	Role    string                `json:"role"`    // Message role: "user" or "assistant"
	Content *PromptMessageContent `json:"content"` // Message content
}

// PromptMessageContent represents the content of a prompt message.
// Can be text, resource, JSON, or image content.
type PromptMessageContent struct {
	Type string `json:"type"` // Content type: "text", "resource", "json", "image"

	// Text content fields
	Text *string `json:"text,omitempty"`

	// Resource content fields
	URI      *string `json:"uri,omitempty"`
	MimeType *string `json:"mimeType,omitempty"`
	Blob     *string `json:"blob,omitempty"`

	// JSON/Image content fields (data can be string for images or any for JSON)
	Data any `json:"data,omitempty"`
}
