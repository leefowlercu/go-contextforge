package contextforge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ServersService handles communication with the server-related
// methods of the ContextForge API.
//
// Note: This service intentionally excludes certain MCP protocol endpoints:
// - GET /servers/{server_id}/sse - SSE connection for MCP protocol communication
// - POST /servers/{server_id}/message - MCP protocol message handling
// These endpoints are for MCP protocol communication, not REST API management.

// List retrieves a paginated list of servers from the ContextForge API.
func (s *ServersService) List(ctx context.Context, opts *ServerListOptions) ([]*Server, *Response, error) {
	u := "servers"
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var servers []*Server
	resp, err := s.client.Do(ctx, req, &servers)
	if err != nil {
		return nil, resp, err
	}

	return servers, resp, nil
}

// Get retrieves a specific server by its ID.
func (s *ServersService) Get(ctx context.Context, serverID string) (*Server, *Response, error) {
	u := fmt.Sprintf("servers/%s", url.PathEscape(serverID))

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var server *Server
	resp, err := s.client.Do(ctx, req, &server)
	if err != nil {
		return nil, resp, err
	}

	return server, resp, nil
}

// Create creates a new server.
// The opts parameter allows setting team_id and visibility at the request wrapper level.
func (s *ServersService) Create(ctx context.Context, server *ServerCreate, opts *ServerCreateOptions) (*Server, *Response, error) {
	u := "servers"

	// Build the request wrapper with server and additional fields
	body := map[string]any{
		"server": server,
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

	var created *Server
	resp, err := s.client.Do(ctx, req, &created)
	if err != nil {
		return nil, resp, err
	}

	return created, resp, nil
}

// Update updates an existing server.
// Note: The API does not wrap the request body for server updates.
func (s *ServersService) Update(ctx context.Context, serverID string, server *ServerUpdate) (*Server, *Response, error) {
	u := fmt.Sprintf("servers/%s", url.PathEscape(serverID))

	req, err := s.client.NewRequest(http.MethodPut, u, server)
	if err != nil {
		return nil, nil, err
	}

	var updated *Server
	resp, err := s.client.Do(ctx, req, &updated)
	if err != nil {
		return nil, resp, err
	}

	return updated, resp, nil
}

// Delete deletes a server by its ID.
func (s *ServersService) Delete(ctx context.Context, serverID string) (*Response, error) {
	u := fmt.Sprintf("servers/%s", url.PathEscape(serverID))

	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// Toggle toggles a server's active status.
func (s *ServersService) Toggle(ctx context.Context, serverID string, activate bool) (*Server, *Response, error) {
	u := fmt.Sprintf("servers/%s/toggle?activate=%t", url.PathEscape(serverID), activate)

	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var server *Server
	resp, err := s.client.Do(ctx, req, &server)
	if err != nil {
		return nil, resp, err
	}

	return server, resp, nil
}

// ListTools retrieves all tools associated with a specific server.
func (s *ServersService) ListTools(ctx context.Context, serverID string, opts *ServerAssociationOptions) ([]*Tool, *Response, error) {
	u := fmt.Sprintf("servers/%s/tools", url.PathEscape(serverID))
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var tools []*Tool
	resp, err := s.client.Do(ctx, req, &tools)
	if err != nil {
		return nil, resp, err
	}

	return tools, resp, nil
}

// ListResources retrieves all resources associated with a specific server.
func (s *ServersService) ListResources(ctx context.Context, serverID string, opts *ServerAssociationOptions) ([]*Resource, *Response, error) {
	u := fmt.Sprintf("servers/%s/resources", url.PathEscape(serverID))
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var resources []*Resource
	resp, err := s.client.Do(ctx, req, &resources)
	if err != nil {
		return nil, resp, err
	}

	return resources, resp, nil
}

// ListPrompts retrieves all prompts associated with a specific server.
func (s *ServersService) ListPrompts(ctx context.Context, serverID string, opts *ServerAssociationOptions) ([]*Prompt, *Response, error) {
	u := fmt.Sprintf("servers/%s/prompts", url.PathEscape(serverID))
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var prompts []*Prompt
	resp, err := s.client.Do(ctx, req, &prompts)
	if err != nil {
		return nil, resp, err
	}

	return prompts, resp, nil
}
