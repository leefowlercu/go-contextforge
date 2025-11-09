package contextforge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// TeamsService handles communication with the team-related
// methods of the ContextForge API.
//
// Note: All /teams/* endpoints are REST API management endpoints.
// There are no MCP protocol endpoints to exclude for this service.

// List retrieves a paginated list of teams from the ContextForge API.
// Note: Teams use skip/limit (offset-based) pagination instead of cursor-based.
func (s *TeamsService) List(ctx context.Context, opts *TeamListOptions) ([]*Team, *Response, error) {
	u := "teams"
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var result *TeamListResponse
	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	return result.Teams, resp, nil
}

// Get retrieves a specific team by its ID.
func (s *TeamsService) Get(ctx context.Context, teamID string) (*Team, *Response, error) {
	u := fmt.Sprintf("teams/%s/", url.PathEscape(teamID))

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var team *Team
	resp, err := s.client.Do(ctx, req, &team)
	if err != nil {
		return nil, resp, err
	}

	return team, resp, nil
}

// Create creates a new team.
// Note: The API does not wrap the request body for team creation.
func (s *TeamsService) Create(ctx context.Context, team *TeamCreate) (*Team, *Response, error) {
	u := "teams"

	req, err := s.client.NewRequest(http.MethodPost, u, team)
	if err != nil {
		return nil, nil, err
	}

	var created *Team
	resp, err := s.client.Do(ctx, req, &created)
	if err != nil {
		return nil, resp, err
	}

	return created, resp, nil
}

// Update updates an existing team.
// Note: The API does not wrap the request body for team updates.
func (s *TeamsService) Update(ctx context.Context, teamID string, team *TeamUpdate) (*Team, *Response, error) {
	u := fmt.Sprintf("teams/%s/", url.PathEscape(teamID))

	req, err := s.client.NewRequest(http.MethodPut, u, team)
	if err != nil {
		return nil, nil, err
	}

	var updated *Team
	resp, err := s.client.Do(ctx, req, &updated)
	if err != nil {
		return nil, resp, err
	}

	return updated, resp, nil
}

// Delete deletes a team by ID.
func (s *TeamsService) Delete(ctx context.Context, teamID string) (*Response, error) {
	u := fmt.Sprintf("teams/%s/", url.PathEscape(teamID))

	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	return resp, err
}

// ListMembers retrieves a list of team members.
func (s *TeamsService) ListMembers(ctx context.Context, teamID string) ([]*TeamMember, *Response, error) {
	u := fmt.Sprintf("teams/%s/members/", url.PathEscape(teamID))

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var members []*TeamMember
	resp, err := s.client.Do(ctx, req, &members)
	if err != nil {
		return nil, resp, err
	}

	return members, resp, nil
}

// UpdateMember updates a team member's role.
// Note: Uses email as the member identifier, not ID.
func (s *TeamsService) UpdateMember(ctx context.Context, teamID, userEmail string, update *TeamMemberUpdate) (*TeamMember, *Response, error) {
	u := fmt.Sprintf("teams/%s/members/%s/", url.PathEscape(teamID), url.PathEscape(userEmail))

	req, err := s.client.NewRequest(http.MethodPut, u, update)
	if err != nil {
		return nil, nil, err
	}

	var member *TeamMember
	resp, err := s.client.Do(ctx, req, &member)
	if err != nil {
		return nil, resp, err
	}

	return member, resp, nil
}

// RemoveMember removes a member from a team.
// Note: Uses email as the member identifier, not ID.
func (s *TeamsService) RemoveMember(ctx context.Context, teamID, userEmail string) (*Response, error) {
	u := fmt.Sprintf("teams/%s/members/%s/", url.PathEscape(teamID), url.PathEscape(userEmail))

	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	return resp, err
}

// InviteMember invites a user to join a team.
func (s *TeamsService) InviteMember(ctx context.Context, teamID string, invite *TeamInvite) (*TeamInvitation, *Response, error) {
	u := fmt.Sprintf("teams/%s/invitations/", url.PathEscape(teamID))

	req, err := s.client.NewRequest(http.MethodPost, u, invite)
	if err != nil {
		return nil, nil, err
	}

	var invitation *TeamInvitation
	resp, err := s.client.Do(ctx, req, &invitation)
	if err != nil {
		return nil, resp, err
	}

	return invitation, resp, nil
}

// ListInvitations retrieves a list of team invitations.
func (s *TeamsService) ListInvitations(ctx context.Context, teamID string) ([]*TeamInvitation, *Response, error) {
	u := fmt.Sprintf("teams/%s/invitations/", url.PathEscape(teamID))

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var invitations []*TeamInvitation
	resp, err := s.client.Do(ctx, req, &invitations)
	if err != nil {
		return nil, resp, err
	}

	return invitations, resp, nil
}

// AcceptInvitation accepts a team invitation using the invitation token.
func (s *TeamsService) AcceptInvitation(ctx context.Context, token string) (*TeamMember, *Response, error) {
	u := fmt.Sprintf("teams/invitations/%s/accept/", url.PathEscape(token))

	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var member *TeamMember
	resp, err := s.client.Do(ctx, req, &member)
	if err != nil {
		return nil, resp, err
	}

	return member, resp, nil
}

// CancelInvitation cancels a team invitation.
func (s *TeamsService) CancelInvitation(ctx context.Context, invitationID string) (*Response, error) {
	u := fmt.Sprintf("teams/invitations/%s/", url.PathEscape(invitationID))

	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	return resp, err
}

// Discover retrieves a list of public teams that the user can join.
func (s *TeamsService) Discover(ctx context.Context, opts *TeamDiscoverOptions) ([]*TeamDiscovery, *Response, error) {
	u := "teams/discover"
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var teams []*TeamDiscovery
	resp, err := s.client.Do(ctx, req, &teams)
	if err != nil {
		return nil, resp, err
	}

	return teams, resp, nil
}

// Join requests to join a public team.
func (s *TeamsService) Join(ctx context.Context, teamID string, request *TeamJoinRequest) (*TeamJoinRequestResponse, *Response, error) {
	u := fmt.Sprintf("teams/%s/join/", url.PathEscape(teamID))

	req, err := s.client.NewRequest(http.MethodPost, u, request)
	if err != nil {
		return nil, nil, err
	}

	var joinRequest *TeamJoinRequestResponse
	resp, err := s.client.Do(ctx, req, &joinRequest)
	if err != nil {
		return nil, resp, err
	}

	return joinRequest, resp, nil
}

// Leave removes the current user from a team.
func (s *TeamsService) Leave(ctx context.Context, teamID string) (*Response, error) {
	u := fmt.Sprintf("teams/%s/leave/", url.PathEscape(teamID))

	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	return resp, err
}

// ListJoinRequests retrieves a list of join requests for a team.
// Only team owners can view join requests.
func (s *TeamsService) ListJoinRequests(ctx context.Context, teamID string) ([]*TeamJoinRequestResponse, *Response, error) {
	u := fmt.Sprintf("teams/%s/join-requests/", url.PathEscape(teamID))

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var requests []*TeamJoinRequestResponse
	resp, err := s.client.Do(ctx, req, &requests)
	if err != nil {
		return nil, resp, err
	}

	return requests, resp, nil
}

// ApproveJoinRequest approves a join request, adding the user to the team.
func (s *TeamsService) ApproveJoinRequest(ctx context.Context, teamID, requestID string) (*TeamMember, *Response, error) {
	u := fmt.Sprintf("teams/%s/join-requests/%s/approve/", url.PathEscape(teamID), url.PathEscape(requestID))

	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var member *TeamMember
	resp, err := s.client.Do(ctx, req, &member)
	if err != nil {
		return nil, resp, err
	}

	return member, resp, nil
}

// RejectJoinRequest rejects a join request.
func (s *TeamsService) RejectJoinRequest(ctx context.Context, teamID, requestID string) (*Response, error) {
	u := fmt.Sprintf("teams/%s/join-requests/%s/", url.PathEscape(teamID), url.PathEscape(requestID))

	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	return resp, err
}
