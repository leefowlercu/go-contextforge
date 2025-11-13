//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

func TestGatewayError(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	gateway := minimalGatewayInput()
	t.Logf("Creating gateway with URL: %s", gateway.URL)

	_, resp, err := client.Gateways.Create(ctx, gateway, nil)
	if err != nil {
		t.Logf("Error response status: %d", resp.StatusCode)

		// Try to extract ErrorResponse details
		var errResp *contextforge.ErrorResponse
		if errors.As(err, &errResp) {
			t.Logf("ErrorResponse Message: %s", errResp.Message)
			if len(errResp.Errors) > 0 {
				t.Logf("ErrorResponse Errors: %+v", errResp.Errors)
			}
		}

		// Also try reading body directly (though it may be consumed)
		if resp.Body != nil {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			var errorResp map[string]any
			if json.Unmarshal(body[:n], &errorResp) == nil {
				t.Logf("Error response body: %+v", errorResp)
			} else {
				t.Logf("Error response raw: %s", string(body[:n]))
			}
		}

		t.Logf("Error: %v", err)
	}
}
