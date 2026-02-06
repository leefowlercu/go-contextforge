package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func normalizeServer(server *Server) *Server {
	if server == nil {
		return nil
	}
	if server.Enabled && !server.IsActive {
		server.IsActive = true
	}
	if server.IsActive && !server.Enabled {
		server.Enabled = true
	}
	return server
}

// ServersService handles communication with the server-related
// methods of the ContextForge API.
//
// Note: This service intentionally excludes certain MCP protocol transport endpoints:
// - GET /servers/{server_id}/sse - SSE connection for MCP protocol proxying
// - POST /servers/{server_id}/message - JSON-RPC message relay for SSE sessions
// These endpoints are for MCP protocol transport infrastructure, not REST API management.
//
// The /rpc endpoint handles MCP JSON-RPC protocol which is separate from these REST management endpoints.

// List retrieves a paginated list of servers from the ContextForge API.
func (s *ServersService) List(ctx context.Context, opts *ServerListOptions) ([]*Server, *Response, error) {
	reqOpts := &ServerListOptions{}
	if opts != nil {
		*reqOpts = *opts
	}
	reqOpts.IncludePagination = true

	u := "servers"
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

	servers, nextCursor, err := decodeListResponse[Server](raw, "servers")
	if err != nil {
		return nil, resp, err
	}
	for i := range servers {
		servers[i] = normalizeServer(servers[i])
	}
	if nextCursor != "" {
		resp.NextCursor = nextCursor
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
	return normalizeServer(server), resp, nil
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

	return normalizeServer(created), resp, nil
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

	return normalizeServer(updated), resp, nil
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

// SetState sets a server's active status using the preferred /state endpoint.
func (s *ServersService) SetState(ctx context.Context, serverID string, activate bool) (*Server, *Response, error) {
	return s.setState(ctx, serverID, activate, "state")
}

// Toggle toggles a server's active status using the legacy /toggle endpoint.
func (s *ServersService) Toggle(ctx context.Context, serverID string, activate bool) (*Server, *Response, error) {
	return s.setState(ctx, serverID, activate, "toggle")
}

func (s *ServersService) setState(ctx context.Context, serverID string, activate bool, endpoint string) (*Server, *Response, error) {
	u := fmt.Sprintf("servers/%s/%s?activate=%t", url.PathEscape(serverID), endpoint, activate)

	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var server *Server
	resp, err := s.client.Do(ctx, req, &server)
	if err != nil {
		return nil, resp, err
	}

	return normalizeServer(server), resp, nil
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
