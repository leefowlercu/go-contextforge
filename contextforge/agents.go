package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// AgentsService handles communication with the A2A agent-related
// methods of the ContextForge API.
//
// A2A (Agent-to-Agent) agents enable inter-agent communication using
// the ContextForge A2A protocol. This service provides management
// operations for creating, updating, and invoking A2A agents.
//
// Note: All /a2a/* endpoints are REST API management endpoints.
// There are no MCP protocol endpoints to exclude for this service.

// List retrieves a paginated list of agents from the ContextForge API.
// Upstream v1.0.0-BETA-2 uses cursor pagination; legacy skip is still accepted
// for backward compatibility in AgentListOptions.
func (s *AgentsService) List(ctx context.Context, opts *AgentListOptions) ([]*Agent, *Response, error) {
	reqOpts := &AgentListOptions{}
	if opts != nil {
		*reqOpts = *opts
	}
	reqOpts.IncludePagination = true

	u := "a2a"
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

	agents, nextCursor, err := decodeListResponse[Agent](raw, "agents")
	if err != nil {
		return nil, resp, err
	}
	if nextCursor != "" {
		resp.NextCursor = nextCursor
	}

	return agents, resp, nil
}

// Get retrieves a specific agent by its ID.
func (s *AgentsService) Get(ctx context.Context, agentID string) (*Agent, *Response, error) {
	u := fmt.Sprintf("a2a/%s", url.PathEscape(agentID))

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var agent *Agent
	resp, err := s.client.Do(ctx, req, &agent)
	if err != nil {
		return nil, resp, err
	}

	return agent, resp, nil
}

// Create creates a new A2A agent.
// The opts parameter allows setting team_id and visibility at the request wrapper level.
func (s *AgentsService) Create(ctx context.Context, agent *AgentCreate, opts *AgentCreateOptions) (*Agent, *Response, error) {
	u := "a2a"

	// Build the request wrapper with agent and additional fields
	body := map[string]any{
		"agent": agent,
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

	var created *Agent
	resp, err := s.client.Do(ctx, req, &created)
	if err != nil {
		return nil, resp, err
	}

	return created, resp, nil
}

// Update updates an existing agent.
// Note: The API does not wrap the request body for agent updates.
func (s *AgentsService) Update(ctx context.Context, agentID string, agent *AgentUpdate) (*Agent, *Response, error) {
	u := fmt.Sprintf("a2a/%s", url.PathEscape(agentID))

	req, err := s.client.NewRequest(http.MethodPut, u, agent)
	if err != nil {
		return nil, nil, err
	}

	var updated *Agent
	resp, err := s.client.Do(ctx, req, &updated)
	if err != nil {
		return nil, resp, err
	}

	return updated, resp, nil
}

// Delete deletes an agent by ID.
func (s *AgentsService) Delete(ctx context.Context, agentID string) (*Response, error) {
	u := fmt.Sprintf("a2a/%s", url.PathEscape(agentID))

	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	return resp, err
}

// SetState sets an agent's enabled status using the preferred /state endpoint.
func (s *AgentsService) SetState(ctx context.Context, agentID string, activate bool) (*Agent, *Response, error) {
	return s.setState(ctx, agentID, activate, "state")
}

// Toggle toggles an agent's enabled status using the legacy /toggle endpoint.
func (s *AgentsService) Toggle(ctx context.Context, agentID string, activate bool) (*Agent, *Response, error) {
	return s.setState(ctx, agentID, activate, "toggle")
}

func (s *AgentsService) setState(ctx context.Context, agentID string, activate bool, endpoint string) (*Agent, *Response, error) {
	u := fmt.Sprintf("a2a/%s/%s?activate=%t", url.PathEscape(agentID), endpoint, activate)

	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var agent *Agent
	resp, err := s.client.Do(ctx, req, &agent)
	if err != nil {
		return nil, resp, err
	}

	return agent, resp, nil
}

// Invoke invokes an A2A agent by name with specified parameters.
// Note: Uses agent name (not ID) as identifier.
// The req parameter is optional; pass nil to use default parameters.
func (s *AgentsService) Invoke(ctx context.Context, agentName string, req *AgentInvokeRequest) (map[string]any, *Response, error) {
	u := fmt.Sprintf("a2a/%s/invoke", url.PathEscape(agentName))

	httpReq, err := s.client.NewRequest(http.MethodPost, u, req)
	if err != nil {
		return nil, nil, err
	}

	var result map[string]any
	resp, err := s.client.Do(ctx, httpReq, &result)
	if err != nil {
		return nil, resp, err
	}

	return result, resp, nil
}
