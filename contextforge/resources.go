package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func normalizeResource(resource *Resource) *Resource {
	if resource == nil {
		return nil
	}
	if resource.Enabled && !resource.IsActive {
		resource.IsActive = true
	}
	if resource.IsActive && !resource.Enabled {
		resource.Enabled = true
	}
	return resource
}

// ResourcesService handles communication with the resource-related
// methods of the ContextForge API.
//
// Note: This service intentionally excludes certain endpoints:
// - POST /resources/subscribe/{id} - SSE streaming for real-time change notifications
// The SSE endpoint is for event streaming, not REST API management.
//
// The /rpc endpoint handles MCP JSON-RPC protocol (resources/read, etc.)
// which is separate from these REST management endpoints.

// List retrieves a paginated list of resources from the ContextForge API.
func (s *ResourcesService) List(ctx context.Context, opts *ResourceListOptions) ([]*Resource, *Response, error) {
	reqOpts := &ResourceListOptions{}
	if opts != nil {
		*reqOpts = *opts
	}
	reqOpts.IncludePagination = true

	u := "resources"
	u, err := addOptions(u, reqOpts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var raw json.RawMessage
	resp, err := s.client.Do(ctx, req, &raw)
	if err != nil {
		return nil, resp, err
	}

	resources, nextCursor, err := decodeListResponse[Resource](raw, "resources")
	if err != nil {
		return nil, resp, err
	}
	for i := range resources {
		resources[i] = normalizeResource(resources[i])
	}
	if nextCursor != "" {
		resp.NextCursor = nextCursor
	}

	return resources, resp, nil
}

// Get retrieves the content of a specific resource by its ID.
// This is a hybrid REST endpoint that returns resource content in MCP-compatible format.
func (s *ResourcesService) Get(ctx context.Context, resourceID string) (*ResourceContent, *Response, error) {
	u := fmt.Sprintf("resources/%s", url.PathEscape(resourceID))

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var content *ResourceContent
	resp, err := s.client.Do(ctx, req, &content)
	if err != nil {
		return nil, resp, err
	}

	return content, resp, nil
}

// GetInfo retrieves metadata for a specific resource by its ID.
// Unlike Get, this endpoint returns resource metadata rather than content.
func (s *ResourcesService) GetInfo(ctx context.Context, resourceID string, opts *ResourceInfoOptions) (*Resource, *Response, error) {
	u := fmt.Sprintf("resources/%s/info", url.PathEscape(resourceID))
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var resource *Resource
	resp, err := s.client.Do(ctx, req, &resource)
	if err != nil {
		return nil, resp, err
	}

	return normalizeResource(resource), resp, nil
}

// Create creates a new resource.
// The opts parameter allows setting team_id and visibility at the request wrapper level.
func (s *ResourcesService) Create(ctx context.Context, resource *ResourceCreate, opts *ResourceCreateOptions) (*Resource, *Response, error) {
	u := "resources"

	// Build the request wrapper with resource and additional fields
	body := map[string]any{
		"resource": resource,
	}

	// Add optional fields from opts if provided
	if opts != nil {
		if opts.TeamID != nil {
			body["team_id"] = *opts.TeamID
		}
		if opts.Visibility != nil {
			body["visibility"] = *opts.Visibility
		}
	}

	req, err := s.client.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, nil, err
	}

	var created *Resource
	resp, err := s.client.Do(ctx, req, &created)
	if err != nil {
		return nil, resp, err
	}

	return normalizeResource(created), resp, nil
}

// Update updates an existing resource.
// Unlike Create, Update does not use request wrapping.
func (s *ResourcesService) Update(ctx context.Context, resourceID string, resource *ResourceUpdate) (*Resource, *Response, error) {
	u := fmt.Sprintf("resources/%s", url.PathEscape(resourceID))

	// No wrapper for update (direct ResourceUpdate object)
	req, err := s.client.NewRequest(http.MethodPut, u, resource)
	if err != nil {
		return nil, nil, err
	}

	var updated *Resource
	resp, err := s.client.Do(ctx, req, &updated)
	if err != nil {
		return nil, resp, err
	}

	return normalizeResource(updated), resp, nil
}

// Delete deletes a resource by its ID.
func (s *ResourcesService) Delete(ctx context.Context, resourceID string) (*Response, error) {
	u := fmt.Sprintf("resources/%s", url.PathEscape(resourceID))

	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	return resp, err
}

// toggleResourceResponse represents the toggle endpoint's response format.
// The toggle endpoint uses snake_case field names, unlike other endpoints which use camelCase.
type toggleResourceResponse struct {
	ID                *FlexibleID `json:"id,omitempty"`
	URI               string      `json:"uri"`
	Name              string      `json:"name"`
	Description       *string     `json:"description,omitempty"`
	MimeType          *string     `json:"mime_type,omitempty"`
	Size              *int        `json:"size,omitempty"`
	IsActive          bool        `json:"is_active"`
	Enabled           bool        `json:"enabled,omitempty"`
	Tags              []Tag       `json:"tags,omitempty"`
	TeamID            *string     `json:"team_id,omitempty"`
	Team              *string     `json:"team,omitempty"`
	OwnerEmail        *string     `json:"owner_email,omitempty"`
	Visibility        *string     `json:"visibility,omitempty"`
	CreatedAt         *Timestamp  `json:"created_at,omitempty"`
	UpdatedAt         *Timestamp  `json:"updated_at,omitempty"`
	CreatedBy         *string     `json:"created_by,omitempty"`
	CreatedFromIP     *string     `json:"created_from_ip,omitempty"`
	CreatedVia        *string     `json:"created_via,omitempty"`
	CreatedUserAgent  *string     `json:"created_user_agent,omitempty"`
	ModifiedBy        *string     `json:"modified_by,omitempty"`
	ModifiedFromIP    *string     `json:"modified_from_ip,omitempty"`
	ModifiedVia       *string     `json:"modified_via,omitempty"`
	ModifiedUserAgent *string     `json:"modified_user_agent,omitempty"`
	ImportBatchID     *string     `json:"import_batch_id,omitempty"`
	FederationSource  *string     `json:"federation_source,omitempty"`
	Version           *int        `json:"version,omitempty"`
}

// SetState enables or disables a resource using the preferred /state endpoint.
func (s *ResourcesService) SetState(ctx context.Context, resourceID string, activate bool) (*Resource, *Response, error) {
	return s.setState(ctx, resourceID, activate, "state")
}

// Toggle enables or disables a resource using the legacy /toggle endpoint.
func (s *ResourcesService) Toggle(ctx context.Context, resourceID string, activate bool) (*Resource, *Response, error) {
	return s.setState(ctx, resourceID, activate, "toggle")
}

func (s *ResourcesService) setState(ctx context.Context, resourceID string, activate bool, endpoint string) (*Resource, *Response, error) {
	u := fmt.Sprintf("resources/%s/%s?activate=%t", url.PathEscape(resourceID), endpoint, activate)

	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, nil, err
	}

	// State endpoints return a wrapped response like: {"status": "...", "resource": {...}}
	var result struct {
		Status   string                  `json:"status"`
		Message  string                  `json:"message"`
		Resource *toggleResourceResponse `json:"resource"`
	}

	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	if result.Resource == nil {
		return nil, resp, fmt.Errorf("toggle response missing 'resource' field")
	}

	// Convert toggle response to standard Resource struct
	resource := &Resource{
		ID:                result.Resource.ID,
		URI:               result.Resource.URI,
		Name:              result.Resource.Name,
		Description:       result.Resource.Description,
		MimeType:          result.Resource.MimeType,
		Size:              result.Resource.Size,
		IsActive:          result.Resource.IsActive || result.Resource.Enabled,
		Enabled:           result.Resource.Enabled || result.Resource.IsActive,
		Tags:              result.Resource.Tags,
		TeamID:            result.Resource.TeamID,
		Team:              result.Resource.Team,
		OwnerEmail:        result.Resource.OwnerEmail,
		Visibility:        result.Resource.Visibility,
		CreatedAt:         result.Resource.CreatedAt,
		UpdatedAt:         result.Resource.UpdatedAt,
		CreatedBy:         result.Resource.CreatedBy,
		CreatedFromIP:     result.Resource.CreatedFromIP,
		CreatedVia:        result.Resource.CreatedVia,
		CreatedUserAgent:  result.Resource.CreatedUserAgent,
		ModifiedBy:        result.Resource.ModifiedBy,
		ModifiedFromIP:    result.Resource.ModifiedFromIP,
		ModifiedVia:       result.Resource.ModifiedVia,
		ModifiedUserAgent: result.Resource.ModifiedUserAgent,
		ImportBatchID:     result.Resource.ImportBatchID,
		FederationSource:  result.Resource.FederationSource,
		Version:           result.Resource.Version,
	}

	return normalizeResource(resource), resp, nil
}

// ListTemplates retrieves available resource templates.
func (s *ResourcesService) ListTemplates(ctx context.Context) (*ListResourceTemplatesResult, *Response, error) {
	u := "resources/templates/list"

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var result *ListResourceTemplatesResult
	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	return result, resp, nil
}
