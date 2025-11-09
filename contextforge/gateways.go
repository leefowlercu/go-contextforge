package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GatewaysService handles communication with the gateway-related
// methods of the ContextForge API.
//
// Note: All /gateways/* endpoints are REST API management endpoints.
// There are no MCP protocol endpoints to exclude for this service.

// List retrieves a list of gateways from the ContextForge API.
func (s *GatewaysService) List(ctx context.Context, opts *GatewayListOptions) ([]*Gateway, *Response, error) {
	u := "gateways"
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var gateways []*Gateway
	resp, err := s.client.Do(ctx, req, &gateways)
	if err != nil {
		return nil, resp, err
	}

	return gateways, resp, nil
}

// Get retrieves a specific gateway by its ID.
func (s *GatewaysService) Get(ctx context.Context, gatewayID string) (*Gateway, *Response, error) {
	u := fmt.Sprintf("gateways/%s", url.PathEscape(gatewayID))

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var gateway *Gateway
	resp, err := s.client.Do(ctx, req, &gateway)
	if err != nil {
		return nil, resp, err
	}

	return gateway, resp, nil
}

// Create creates a new gateway.
// The opts parameter allows setting team_id and visibility fields.
// Note: Unlike other services, gateway creation does NOT wrap the gateway object.
func (s *GatewaysService) Create(ctx context.Context, gateway *Gateway, opts *GatewayCreateOptions) (*Gateway, *Response, error) {
	u := "gateways"

	// Convert gateway to map for merging with opts
	// Note: The API expects gateway fields at the top level, NOT wrapped
	gatewayMap := make(map[string]any)
	gatewayJSON, err := json.Marshal(gateway)
	if err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(gatewayJSON, &gatewayMap); err != nil {
		return nil, nil, err
	}

	// Add optional fields from opts if provided
	if opts != nil {
		if opts.TeamID != nil {
			gatewayMap["team_id"] = *opts.TeamID
		}
		if opts.Visibility != nil {
			gatewayMap["visibility"] = *opts.Visibility
		}
	}

	req, err := s.client.NewRequest(http.MethodPost, u, gatewayMap)
	if err != nil {
		return nil, nil, err
	}

	var created *Gateway
	resp, err := s.client.Do(ctx, req, &created)
	if err != nil {
		return nil, resp, err
	}

	return created, resp, nil
}

// Update updates an existing gateway.
func (s *GatewaysService) Update(ctx context.Context, gatewayID string, gateway *Gateway) (*Gateway, *Response, error) {
	u := fmt.Sprintf("gateways/%s", url.PathEscape(gatewayID))

	req, err := s.client.NewRequest(http.MethodPut, u, gateway)
	if err != nil {
		return nil, nil, err
	}

	var updated *Gateway
	resp, err := s.client.Do(ctx, req, &updated)
	if err != nil {
		return nil, resp, err
	}

	return updated, resp, nil
}

// Delete deletes a gateway by its ID.
func (s *GatewaysService) Delete(ctx context.Context, gatewayID string) (*Response, error) {
	u := fmt.Sprintf("gateways/%s", url.PathEscape(gatewayID))

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

// Toggle toggles a gateway's enabled status.
func (s *GatewaysService) Toggle(ctx context.Context, gatewayID string, activate bool) (*Gateway, *Response, error) {
	u := fmt.Sprintf("gateways/%s/toggle?activate=%t", url.PathEscape(gatewayID), activate)

	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, nil, err
	}

	// The API returns a response with the gateway data nested in the response
	var result map[string]any
	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	// Extract the gateway from the response
	var gateway *Gateway
	if gatewayData, ok := result["gateway"]; ok {
		// Re-marshal and unmarshal to convert to Gateway struct
		gatewayJSON, _ := json.Marshal(gatewayData)
		json.Unmarshal(gatewayJSON, &gateway)
	}

	return gateway, resp, nil
}
