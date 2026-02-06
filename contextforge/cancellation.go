package contextforge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Cancel requests cancellation for an in-flight run or request.
func (s *CancellationService) Cancel(ctx context.Context, req *CancellationRequest) (*CancellationResponse, *Response, error) {
	if req == nil {
		return nil, nil, fmt.Errorf("cancellation request is nil")
	}

	u := "cancellation/cancel"
	httpReq, err := s.client.NewRequest(http.MethodPost, u, req)
	if err != nil {
		return nil, nil, err
	}

	var result *CancellationResponse
	resp, err := s.client.Do(ctx, httpReq, &result)
	if err != nil {
		return nil, resp, err
	}

	return result, resp, nil
}

// Status retrieves cancellation status for a request ID.
func (s *CancellationService) Status(ctx context.Context, requestID string) (*CancellationStatus, *Response, error) {
	u := fmt.Sprintf("cancellation/status/%s", url.PathEscape(requestID))
	httpReq, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var status *CancellationStatus
	resp, err := s.client.Do(ctx, httpReq, &status)
	if err != nil {
		return nil, resp, err
	}

	return status, resp, nil
}
