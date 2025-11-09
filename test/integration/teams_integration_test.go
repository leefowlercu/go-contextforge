//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

// TestTeamsService_BasicCRUD tests basic CRUD operations
func TestTeamsService_BasicCRUD(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create team with minimal fields", func(t *testing.T) {
		team := minimalTeamInput()

		created, _, err := client.Teams.Create(ctx, team)
		if err != nil {
			t.Fatalf("Failed to create team: %v", err)
		}

		t.Cleanup(func() {
			cleanupTeam(t, client, created.ID)
		})

		if created.ID == "" {
			t.Error("Expected created team to have an ID")
		}
		if created.Name != team.Name {
			t.Errorf("Expected team name %q, got %q", team.Name, created.Name)
		}
		if created.Slug == "" {
			t.Error("Expected created team to have a slug")
		}
		if created.MemberCount != 1 {
			t.Errorf("Expected member count 1 (creator), got %d", created.MemberCount)
		}
		if !created.IsActive {
			t.Error("Expected created team to be active")
		}

		t.Logf("Successfully created team: %s (ID: %s, Slug: %s)", created.Name, created.ID, created.Slug)
	})

	t.Run("create team with all optional fields", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-005: Teams API ignores user-provided slug field - see docs/upstream-bugs/teams-slug-ignored.md")
		team := completeTeamInput()

		created, _, err := client.Teams.Create(ctx, team)
		if err != nil {
			t.Fatalf("Failed to create team with all fields: %v", err)
		}

		t.Cleanup(func() {
			cleanupTeam(t, client, created.ID)
		})

		if created.ID == "" {
			t.Error("Expected created team to have an ID")
		}
		if created.Name != team.Name {
			t.Errorf("Expected team name %q, got %q", team.Name, created.Name)
		}
		if created.Slug != *team.Slug {
			t.Errorf("Expected slug %q, got %q", *team.Slug, created.Slug)
		}
		if created.Description == nil || *created.Description != *team.Description {
			t.Errorf("Expected description %q, got %v", *team.Description, created.Description)
		}
		if created.Visibility == nil || *created.Visibility != *team.Visibility {
			t.Errorf("Expected visibility %q, got %v", *team.Visibility, created.Visibility)
		}
		if created.MaxMembers == nil || *created.MaxMembers != *team.MaxMembers {
			t.Errorf("Expected max members %d, got %v", *team.MaxMembers, created.MaxMembers)
		}

		t.Logf("Successfully created team with all fields: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("get team by ID", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		created := createTestTeam(t, client, randomTeamName())

		retrieved, _, err := client.Teams.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to get team: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("Expected team ID %q, got %q", created.ID, retrieved.ID)
		}
		if retrieved.Name != created.Name {
			t.Errorf("Expected team name %q, got %q", created.Name, retrieved.Name)
		}
		if retrieved.Slug != created.Slug {
			t.Errorf("Expected slug %q, got %q", created.Slug, retrieved.Slug)
		}

		t.Logf("Successfully retrieved team: %s (ID: %s)", retrieved.Name, retrieved.ID)
	})

	t.Run("update team", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		created := createTestTeam(t, client, randomTeamName())

		update := &contextforge.TeamUpdate{
			Name:        contextforge.String("updated-team-name"),
			Description: contextforge.String("Updated description"),
			Visibility:  contextforge.String("public"),
		}

		updated, _, err := client.Teams.Update(ctx, created.ID, update)
		if err != nil {
			t.Fatalf("Failed to update team: %v", err)
		}

		if updated.Name != *update.Name {
			t.Errorf("Expected updated name %q, got %q", *update.Name, updated.Name)
		}
		if updated.Description == nil || *updated.Description != *update.Description {
			t.Errorf("Expected updated description %q, got %v", *update.Description, updated.Description)
		}
		if updated.Visibility == nil || *updated.Visibility != *update.Visibility {
			t.Errorf("Expected updated visibility %q, got %v", *update.Visibility, updated.Visibility)
		}

		t.Logf("Successfully updated team: %s (ID: %s)", updated.Name, updated.ID)
	})

	t.Run("delete team", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		created := createTestTeam(t, client, randomTeamName())

		// Delete manually (not via cleanup)
		resp, err := client.Teams.Delete(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to delete team: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify deletion by attempting to get the team
		_, getResp, err := client.Teams.Get(ctx, created.ID)
		if err == nil {
			t.Error("Expected error when getting deleted team")
		}
		if getResp != nil && getResp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for deleted team, got %d", getResp.StatusCode)
		}

		t.Logf("Successfully deleted team: %s", created.ID)
	})
}

// TestTeamsService_List tests list operations and pagination
func TestTeamsService_List(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("list teams with default options", func(t *testing.T) {
		teams, resp, err := client.Teams.List(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list teams: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if teams == nil {
			t.Fatal("Expected teams array, got nil")
		}

		t.Logf("Successfully listed %d teams", len(teams))
	})

	t.Run("list teams with pagination", func(t *testing.T) {
		// Create multiple teams for pagination test
		teamIDs := make([]string, 0)
		for i := 0; i < 3; i++ {
			created := createTestTeam(t, client, randomTeamName())
			teamIDs = append(teamIDs, created.ID)
		}

		opts := &contextforge.TeamListOptions{
			Skip:  0,
			Limit: 2,
		}

		teams, resp, err := client.Teams.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list teams with pagination: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if len(teams) > 2 {
			t.Errorf("Expected at most 2 teams due to limit, got %d", len(teams))
		}

		t.Logf("Successfully listed %d teams with limit 2", len(teams))
	})
}

// TestTeamsService_Members tests member management operations
func TestTeamsService_Members(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("list team members", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		created := createTestTeam(t, client, randomTeamName())

		members, resp, err := client.Teams.ListMembers(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to list team members: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if len(members) < 1 {
			t.Error("Expected at least 1 member (creator)")
		}

		// Verify creator is a member with owner role
		foundCreator := false
		for _, member := range members {
			if member.Role == "owner" {
				foundCreator = true
				if member.UserEmail != created.CreatedBy {
					t.Errorf("Expected owner email %q, got %q", created.CreatedBy, member.UserEmail)
				}
				break
			}
		}

		if !foundCreator {
			t.Error("Expected to find creator as owner in members list")
		}

		t.Logf("Successfully listed %d team members", len(members))
	})
}

// TestTeamsService_Invitations tests invitation operations
func TestTeamsService_Invitations(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create invitation", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		created := createTestTeam(t, client, randomTeamName())

		invite := &contextforge.TeamInvite{
			Email: "testuser@example.com",
			Role:  contextforge.String("member"),
		}

		invitation, resp, err := client.Teams.InviteMember(ctx, created.ID, invite)
		if err != nil {
			t.Fatalf("Failed to create invitation: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		if invitation.ID == "" {
			t.Error("Expected invitation to have an ID")
		}
		if invitation.Email != invite.Email {
			t.Errorf("Expected invitation email %q, got %q", invite.Email, invitation.Email)
		}
		if invitation.Role != *invite.Role {
			t.Errorf("Expected invitation role %q, got %q", *invite.Role, invitation.Role)
		}
		if invitation.Token == "" {
			t.Error("Expected invitation to have a token")
		}

		t.Cleanup(func() {
			client.Teams.CancelInvitation(ctx, invitation.ID)
		})

		t.Logf("Successfully created invitation: %s (Token: %s)", invitation.ID, invitation.Token)
	})

	t.Run("list team invitations", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		created := createTestTeam(t, client, randomTeamName())

		// Create an invitation first
		invite := &contextforge.TeamInvite{
			Email: "testuser@example.com",
		}

		invitation, _, err := client.Teams.InviteMember(ctx, created.ID, invite)
		if err != nil {
			t.Fatalf("Failed to create invitation: %v", err)
		}

		t.Cleanup(func() {
			client.Teams.CancelInvitation(ctx, invitation.ID)
		})

		// List invitations
		invitations, resp, err := client.Teams.ListInvitations(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to list invitations: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if len(invitations) < 1 {
			t.Error("Expected at least 1 invitation")
		}

		// Verify our invitation is in the list
		foundInvitation := false
		for _, inv := range invitations {
			if inv.ID == invitation.ID {
				foundInvitation = true
				break
			}
		}

		if !foundInvitation {
			t.Error("Expected to find created invitation in list")
		}

		t.Logf("Successfully listed %d invitations", len(invitations))
	})

	t.Run("cancel invitation", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		created := createTestTeam(t, client, randomTeamName())

		invite := &contextforge.TeamInvite{
			Email: "testuser@example.com",
		}

		invitation, _, err := client.Teams.InviteMember(ctx, created.ID, invite)
		if err != nil {
			t.Fatalf("Failed to create invitation: %v", err)
		}

		// Cancel the invitation
		resp, err := client.Teams.CancelInvitation(ctx, invitation.ID)
		if err != nil {
			t.Fatalf("Failed to cancel invitation: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		t.Logf("Successfully canceled invitation: %s", invitation.ID)
	})
}

// TestTeamsService_Discovery tests team discovery operations
func TestTeamsService_Discovery(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("discover public teams", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Team discovery endpoint rejects valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		// Create a public team for discovery
		team := &contextforge.TeamCreate{
			Name:       randomTeamName(),
			Visibility: contextforge.String("public"),
		}

		created, _, err := client.Teams.Create(ctx, team)
		if err != nil {
			t.Fatalf("Failed to create public team: %v", err)
		}

		t.Cleanup(func() {
			cleanupTeam(t, client, created.ID)
		})

		// Discover teams
		teams, resp, err := client.Teams.Discover(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to discover teams: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if teams == nil {
			t.Fatal("Expected teams array, got nil")
		}

		t.Logf("Successfully discovered %d public teams", len(teams))
	})

	t.Run("discover teams with pagination", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Team discovery endpoint rejects valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		opts := &contextforge.TeamDiscoverOptions{
			Skip:  0,
			Limit: 5,
		}

		teams, resp, err := client.Teams.Discover(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to discover teams with pagination: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if len(teams) > 5 {
			t.Errorf("Expected at most 5 teams due to limit, got %d", len(teams))
		}

		t.Logf("Successfully discovered %d teams with limit 5", len(teams))
	})
}

// TestTeamsService_ErrorHandling tests error scenarios
func TestTeamsService_ErrorHandling(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("get non-existent team returns 404", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		_, resp, err := client.Teams.Get(ctx, "non-existent-id")
		if err == nil {
			t.Error("Expected error when getting non-existent team")
		}

		if resp == nil || resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %v", resp)
		}

		t.Logf("Correctly returned 404 for non-existent team")
	})

	t.Run("update non-existent team returns 404", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		update := &contextforge.TeamUpdate{
			Name: contextforge.String("updated-name"),
		}

		_, resp, err := client.Teams.Update(ctx, "non-existent-id", update)
		if err == nil {
			t.Error("Expected error when updating non-existent team")
		}

		if resp == nil || resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %v", resp)
		}

		t.Logf("Correctly returned 404 for non-existent team update")
	})

	t.Run("delete non-existent team returns 404", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-004: Individual team endpoints reject valid authentication - see docs/upstream-bugs/teams-auth-individual-endpoints.md")
		resp, err := client.Teams.Delete(ctx, "non-existent-id")
		if err == nil {
			t.Error("Expected error when deleting non-existent team")
		}

		if resp == nil || resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %v", resp)
		}

		t.Logf("Correctly returned 404 for non-existent team deletion")
	})

	t.Run("create team without required name returns 400", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-006: Teams API returns 422 instead of 400 for validation errors - see docs/upstream-bugs/teams-validation-error-code.md")
		team := &contextforge.TeamCreate{
			// Missing required Name field
			Description: contextforge.String("A team without a name"),
		}

		_, resp, err := client.Teams.Create(ctx, team)
		if err == nil {
			t.Error("Expected error when creating team without name")
		}

		if resp == nil || resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %v", resp)
		}

		t.Logf("Correctly returned 400 for team creation without name")
	})
}

// TestTeamsService_Validation tests input validation
func TestTeamsService_Validation(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create team with valid slug pattern", func(t *testing.T) {
		t.Skip("CONTEXTFORGE-005: Teams API ignores user-provided slug field - see docs/upstream-bugs/teams-slug-ignored.md")
		team := &contextforge.TeamCreate{
			Name: randomTeamName(),
			Slug: contextforge.String("valid-slug-123"),
		}

		created, _, err := client.Teams.Create(ctx, team)
		if err != nil {
			t.Fatalf("Failed to create team with valid slug: %v", err)
		}

		t.Cleanup(func() {
			cleanupTeam(t, client, created.ID)
		})

		if created.Slug != *team.Slug {
			t.Errorf("Expected slug %q, got %q", *team.Slug, created.Slug)
		}

		t.Logf("Successfully created team with custom slug: %s", created.Slug)
	})

	t.Run("create team with visibility values", func(t *testing.T) {
		testCases := []struct {
			name       string
			visibility string
		}{
			{"private visibility", "private"},
			{"public visibility", "public"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				team := &contextforge.TeamCreate{
					Name:       randomTeamName(),
					Visibility: contextforge.String(tc.visibility),
				}

				created, _, err := client.Teams.Create(ctx, team)
				if err != nil {
					t.Fatalf("Failed to create team with %s: %v", tc.visibility, err)
				}

				t.Cleanup(func() {
					cleanupTeam(t, client, created.ID)
				})

				if created.Visibility == nil || *created.Visibility != tc.visibility {
					t.Errorf("Expected visibility %q, got %v", tc.visibility, created.Visibility)
				}

				t.Logf("Successfully created team with visibility: %s", tc.visibility)
			})
		}
	})
}
