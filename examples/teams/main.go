// Package main demonstrates comprehensive usage of the TeamsService
// from the go-contextforge SDK. This example highlights team management,
// member operations, invitations, and team discovery with skip/limit pagination.
// Uses a mock HTTP server for self-contained demonstration.
//
// Run: go run examples/teams/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

func main() {
	// Create mock server with all necessary endpoints
	mux := http.NewServeMux()
	setupMockEndpoints(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	fmt.Println("=== ContextForge SDK - Teams Service Example ===")

	// Step 1: Authentication
	fmt.Println("1. Authenticating...")
	token := authenticate(server.URL)
	fmt.Printf("   ✓ Obtained JWT token: %s...\n\n", token[:20])

	// Step 2: Create client
	client, err := contextforge.NewClient(nil, server.URL, token)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Step 3: Create a basic team
	fmt.Println("2. Creating a basic team...")
	newTeam := &contextforge.TeamCreate{
		Name:        "engineering",
		Description: contextforge.String("Engineering team for product development"),
	}

	createdTeam1, resp, err := client.Teams.Create(ctx, newTeam)
	if err != nil {
		log.Fatalf("Failed to create team: %v", err)
	}
	fmt.Printf("   ✓ Created team: %s (ID: %s)\n", createdTeam1.Name, createdTeam1.ID)
	fmt.Printf("   ✓ Slug: %s (auto-generated)\n", createdTeam1.Slug)
	fmt.Printf("   ✓ Member count: %d (creator)\n", createdTeam1.MemberCount)
	fmt.Printf("   ✓ Is active: %v\n", createdTeam1.IsActive)
	fmt.Printf("   ✓ Rate limit: %d/%d remaining\n\n", resp.Rate.Remaining, resp.Rate.Limit)

	// Step 4: Create a team with all optional fields
	fmt.Println("3. Creating a public team with all optional fields...")
	completeTeam := &contextforge.TeamCreate{
		Name:        "design",
		Slug:        contextforge.String("design-team"),
		Description: contextforge.String("Design team responsible for UI/UX"),
		Visibility:  contextforge.String("public"),
		MaxMembers:  contextforge.Int(25),
	}

	createdTeam2, _, err := client.Teams.Create(ctx, completeTeam)
	if err != nil {
		log.Fatalf("Failed to create team with all fields: %v", err)
	}
	fmt.Printf("   ✓ Created team: %s (ID: %s)\n", createdTeam2.Name, createdTeam2.ID)
	fmt.Printf("   ✓ Custom slug: %s\n", createdTeam2.Slug)
	if createdTeam2.Visibility != nil {
		fmt.Printf("   ✓ Visibility: %s\n", *createdTeam2.Visibility)
	}
	if createdTeam2.MaxMembers != nil {
		fmt.Printf("   ✓ Max members: %d\n", *createdTeam2.MaxMembers)
	}
	fmt.Println()

	// Step 5: Get a specific team
	fmt.Println("4. Retrieving team by ID...")
	retrievedTeam, _, err := client.Teams.Get(ctx, createdTeam1.ID)
	if err != nil {
		log.Fatalf("Failed to get team: %v", err)
	}
	fmt.Printf("   ✓ Retrieved: %s\n", retrievedTeam.Name)
	fmt.Printf("   ✓ Slug: %s\n", retrievedTeam.Slug)
	if retrievedTeam.Description != nil {
		fmt.Printf("   ✓ Description: %s\n", *retrievedTeam.Description)
	}
	fmt.Printf("   ✓ Created by: %s\n", retrievedTeam.CreatedBy)
	fmt.Println()

	// Step 6: List all teams
	fmt.Println("5. Listing all teams...")
	teams, _, err := client.Teams.List(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list teams: %v", err)
	}
	fmt.Printf("   ✓ Found %d team(s)\n", len(teams))
	for i, team := range teams {
		visibility := "private"
		if team.Visibility != nil {
			visibility = *team.Visibility
		}
		fmt.Printf("   %d. %s (ID: %s, Members: %d, Visibility: %s)\n",
			i+1, team.Name, team.ID, team.MemberCount, visibility)
	}
	fmt.Println()

	// Step 7: Demonstrate skip/limit pagination
	fmt.Println("6. Demonstrating skip/limit pagination...")
	fmt.Println("   NOTE: Teams use skip/limit (offset-based) pagination")
	listOpts := &contextforge.TeamListOptions{
		Skip:  0,
		Limit: 10,
	}
	pagedTeams, _, err := client.Teams.List(ctx, listOpts)
	if err != nil {
		log.Fatalf("Failed to list teams with pagination: %v", err)
	}
	fmt.Printf("   ✓ Retrieved %d team(s) with skip=0, limit=10\n\n", len(pagedTeams))

	// Step 8: Update a team
	fmt.Println("7. Updating team...")
	updateTeam := &contextforge.TeamUpdate{
		Description: contextforge.String("Updated: Engineering team for product and platform development"),
		Visibility:  contextforge.String("public"),
		MaxMembers:  contextforge.Int(50),
	}

	updatedTeam, _, err := client.Teams.Update(ctx, createdTeam1.ID, updateTeam)
	if err != nil {
		log.Fatalf("Failed to update team: %v", err)
	}
	fmt.Printf("   ✓ Updated description: %s\n", *updatedTeam.Description)
	fmt.Printf("   ✓ Updated visibility: %s\n", *updatedTeam.Visibility)
	fmt.Printf("   ✓ Updated max members: %d\n\n", *updatedTeam.MaxMembers)

	// Step 9: List team members
	fmt.Println("8. Listing team members...")
	members, _, err := client.Teams.ListMembers(ctx, createdTeam1.ID)
	if err != nil {
		log.Fatalf("Failed to list members: %v", err)
	}
	fmt.Printf("   ✓ Found %d member(s)\n", len(members))
	for i, member := range members {
		fmt.Printf("   %d. %s (Role: %s, Active: %v)\n",
			i+1, member.UserEmail, member.Role, member.IsActive)
	}
	fmt.Println()

	// Step 10: Invite a new member
	fmt.Println("9. Inviting a new member to the team...")
	invite := &contextforge.TeamInvite{
		Email: "newuser@example.com",
		Role:  contextforge.String("member"),
	}

	invitation, _, err := client.Teams.InviteMember(ctx, createdTeam1.ID, invite)
	if err != nil {
		log.Fatalf("Failed to create invitation: %v", err)
	}
	fmt.Printf("   ✓ Created invitation for: %s\n", invitation.Email)
	fmt.Printf("   ✓ Invitation ID: %s\n", invitation.ID)
	fmt.Printf("   ✓ Role: %s\n", invitation.Role)
	fmt.Printf("   ✓ Invitation token: %s\n", invitation.Token)
	if invitation.ExpiresAt != nil {
		fmt.Printf("   ✓ Expires at: %s\n", invitation.ExpiresAt.Format(time.RFC3339))
	}
	fmt.Println()

	// Step 11: List team invitations
	fmt.Println("10. Listing team invitations...")
	invitations, _, err := client.Teams.ListInvitations(ctx, createdTeam1.ID)
	if err != nil {
		log.Fatalf("Failed to list invitations: %v", err)
	}
	fmt.Printf("   ✓ Found %d active invitation(s)\n", len(invitations))
	for i, inv := range invitations {
		expired := ""
		if inv.IsExpired {
			expired = " [EXPIRED]"
		}
		fmt.Printf("   %d. %s (Role: %s, Invited by: %s)%s\n",
			i+1, inv.Email, inv.Role, inv.InvitedBy, expired)
	}
	fmt.Println()

	// Step 12: Discover public teams
	fmt.Println("11. Discovering public teams...")
	discoverOpts := &contextforge.TeamDiscoverOptions{
		Limit: 10,
	}
	discoveredTeams, _, err := client.Teams.Discover(ctx, discoverOpts)
	if err != nil {
		log.Fatalf("Failed to discover teams: %v", err)
	}
	fmt.Printf("   ✓ Discovered %d public team(s)\n", len(discoveredTeams))
	for i, team := range discoveredTeams {
		joinable := ""
		if team.IsJoinable {
			joinable = " [JOINABLE]"
		}
		fmt.Printf("   %d. %s (Members: %d)%s\n", i+1, team.Name, team.MemberCount, joinable)
	}
	fmt.Println()

	// Step 13: Error handling example
	fmt.Println("12. Demonstrating error handling...")
	_, _, err = client.Teams.Get(ctx, "non-existent-team-id")
	if err != nil {
		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			fmt.Printf("   ✓ Caught expected error: HTTP %d - %s\n",
				apiErr.Response.StatusCode, apiErr.Message)
		} else {
			fmt.Printf("   ✓ Caught error: %v\n", err)
		}
	}
	fmt.Println()

	// Step 14: Cancel invitation
	fmt.Println("13. Canceling invitation...")
	_, err = client.Teams.CancelInvitation(ctx, invitation.ID)
	if err != nil {
		log.Fatalf("Failed to cancel invitation: %v", err)
	}
	fmt.Printf("   ✓ Canceled invitation: %s\n\n", invitation.ID)

	// Step 15: Delete teams
	fmt.Println("14. Deleting teams...")
	for _, id := range []string{createdTeam1.ID, createdTeam2.ID} {
		_, err = client.Teams.Delete(ctx, id)
		if err != nil {
			log.Fatalf("Failed to delete team %s: %v", id, err)
		}
		fmt.Printf("   ✓ Deleted team: %s\n", id)
	}
	fmt.Println()

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("• Team CRUD operations")
	fmt.Println("• Skip/limit (offset-based) pagination")
	fmt.Println("• Auto-generated slugs from team names")
	fmt.Println("• Team member management")
	fmt.Println("• Invitation system (invite, list, cancel)")
	fmt.Println("• Team discovery (public teams)")
	fmt.Println("• Visibility control (private/public)")
	fmt.Println("• Max members limits")
	fmt.Println("\nAPI Patterns:")
	fmt.Println("• No request wrapping (unlike tools/resources)")
	fmt.Println("• List returns structured response: {teams: [], total: N}")
	fmt.Println("• Member endpoints use email as identifier (not ID)")
	fmt.Println("• Invitation acceptance uses token in path")
	fmt.Println("\nTo use with a real ContextForge instance:")
	fmt.Println("1. Replace server.URL with your ContextForge base URL")
	fmt.Println("2. Use real authentication credentials")
	fmt.Println("3. Manage actual team members and invitations")
}

// authenticate performs mock authentication and returns a JWT token
func authenticate(baseURL string) string {
	loginURL := baseURL + "/auth/login"
	payload := strings.NewReader(`{"email":"admin@example.com","password":"secret"}`)

	resp, err := http.Post(loginURL, "application/json", payload)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	defer resp.Body.Close()

	var authResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		log.Fatalf("Failed to decode auth response: %v", err)
	}

	return authResp.AccessToken
}

// setupMockEndpoints configures all the mock HTTP endpoints
func setupMockEndpoints(mux *http.ServeMux) {
	// Mock authentication endpoint
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "mock-jwt-token-99999",
			"token_type":   "bearer",
		})
	})

	// Mock storage
	teams := make(map[string]*contextforge.Team)
	members := make(map[string][]*contextforge.TeamMember)
	invitations := make(map[string][]*contextforge.TeamInvitation)
	invitationsByID := make(map[string]*contextforge.TeamInvitation)
	var teamCounter, memberCounter, invitationCounter int

	// POST /teams - Create team
	// GET /teams - List teams
	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req contextforge.TeamCreate
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if req.Name == "" {
				http.Error(w, `{"message":"Name is required"}`, http.StatusBadRequest)
				return
			}

			teamCounter++
			id := fmt.Sprintf("team-%d", teamCounter)
			now := time.Now()

			// Generate slug from name if not provided
			slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
			if req.Slug != nil {
				slug = *req.Slug
			}

			team := &contextforge.Team{
				ID:          id,
				Name:        req.Name,
				Slug:        slug,
				Description: req.Description,
				IsPersonal:  false,
				Visibility:  req.Visibility,
				MaxMembers:  req.MaxMembers,
				MemberCount: 1, // Creator
				IsActive:    true,
				CreatedBy:   "admin@example.com",
				CreatedAt:   &contextforge.Timestamp{Time: now},
				UpdatedAt:   &contextforge.Timestamp{Time: now},
			}

			// Create owner member
			memberCounter++
			owner := &contextforge.TeamMember{
				ID:        fmt.Sprintf("member-%d", memberCounter),
				TeamID:    id,
				UserEmail: "admin@example.com",
				Role:      "owner",
				JoinedAt:  &contextforge.Timestamp{Time: now},
				IsActive:  true,
			}

			teams[id] = team
			members[id] = []*contextforge.TeamMember{owner}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "995")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(time.Hour).Unix()))
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(team)

		case http.MethodGet:
			query := r.URL.Query()
			result := []*contextforge.Team{}

			for _, team := range teams {
				result = append(result, team)
			}

			// Handle skip/limit pagination
			skip := 0
			if s := query.Get("skip"); s != "" {
				fmt.Sscanf(s, "%d", &skip)
			}

			limit := 50
			if l := query.Get("limit"); l != "" {
				fmt.Sscanf(l, "%d", &limit)
			}

			total := len(result)

			// Apply skip and limit
			if skip >= len(result) {
				result = []*contextforge.Team{}
			} else {
				result = result[skip:]
				if len(result) > limit {
					result = result[:limit]
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"teams": result,
				"total": total,
			})
		}
	})

	// GET /teams/{id} - Get team
	// PUT /teams/{id} - Update team
	// DELETE /teams/{id} - Delete team
	mux.HandleFunc("/teams/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Handle special endpoints
		if len(parts) == 3 && parts[2] == "discover" {
			handleTeamDiscover(w, r, teams)
			return
		}

		if len(parts) < 3 || parts[2] == "" {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		teamID := parts[2]

		// Handle member endpoints
		if len(parts) >= 4 && parts[3] == "members" {
			handleTeamMembers(w, r, teamID, parts, members, &memberCounter)
			return
		}

		// Handle invitation endpoints
		if len(parts) >= 4 && parts[3] == "invitations" {
			handleTeamInvitations(w, r, teamID, teams, invitations, invitationsByID, &invitationCounter)
			return
		}

		// Standard CRUD operations
		switch r.Method {
		case http.MethodGet:
			team, exists := teams[teamID]
			if !exists {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"message": "Team not found",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(team)

		case http.MethodPut:
			team, exists := teams[teamID]
			if !exists {
				http.Error(w, `{"message":"Team not found"}`, http.StatusNotFound)
				return
			}

			var req contextforge.TeamUpdate
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if req.Name != nil {
				team.Name = *req.Name
			}
			if req.Description != nil {
				team.Description = req.Description
			}
			if req.Visibility != nil {
				team.Visibility = req.Visibility
			}
			if req.MaxMembers != nil {
				team.MaxMembers = req.MaxMembers
			}
			team.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(team)

		case http.MethodDelete:
			if _, exists := teams[teamID]; !exists {
				http.Error(w, `{"message":"Team not found"}`, http.StatusNotFound)
				return
			}

			delete(teams, teamID)
			delete(members, teamID)
			delete(invitations, teamID)
			w.WriteHeader(http.StatusOK)
		}
	})

	// Handle invitations/{token}/accept
	mux.HandleFunc("/teams/invitations/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		invitationID := parts[3]

		if len(parts) == 4 && r.Method == http.MethodDelete {
			// Cancel invitation
			delete(invitationsByID, invitationID)
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "Not implemented", http.StatusNotImplemented)
	})
}

func handleTeamDiscover(w http.ResponseWriter, r *http.Request, teams map[string]*contextforge.Team) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result := []*contextforge.TeamDiscovery{}
	for _, team := range teams {
		if team.Visibility != nil && *team.Visibility == "public" {
			discovery := &contextforge.TeamDiscovery{
				ID:          team.ID,
				Name:        team.Name,
				Description: team.Description,
				MemberCount: team.MemberCount,
				CreatedAt:   team.CreatedAt,
				IsJoinable:  true,
			}
			result = append(result, discovery)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleTeamMembers(w http.ResponseWriter, r *http.Request, teamID string, parts []string, members map[string][]*contextforge.TeamMember, memberCounter *int) {
	if r.Method == http.MethodGet && len(parts) == 4 {
		// List members
		teamMembers := members[teamID]
		if teamMembers == nil {
			teamMembers = []*contextforge.TeamMember{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(teamMembers)
		return
	}

	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func handleTeamInvitations(w http.ResponseWriter, r *http.Request, teamID string, teams map[string]*contextforge.Team, invitations map[string][]*contextforge.TeamInvitation, invitationsByID map[string]*contextforge.TeamInvitation, invitationCounter *int) {
	team, exists := teams[teamID]
	if !exists {
		http.Error(w, `{"message":"Team not found"}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPost:
		// Create invitation
		var req contextforge.TeamInvite
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		*invitationCounter++
		id := fmt.Sprintf("invitation-%d", *invitationCounter)
		now := time.Now()
		expiresAt := now.Add(7 * 24 * time.Hour)

		role := "member"
		if req.Role != nil {
			role = *req.Role
		}

		invitation := &contextforge.TeamInvitation{
			ID:        id,
			TeamID:    teamID,
			TeamName:  team.Name,
			Email:     req.Email,
			Role:      role,
			InvitedBy: "admin@example.com",
			InvitedAt: &contextforge.Timestamp{Time: now},
			ExpiresAt: &contextforge.Timestamp{Time: expiresAt},
			Token:     fmt.Sprintf("token-%d", *invitationCounter),
			IsActive:  true,
			IsExpired: false,
		}

		if invitations[teamID] == nil {
			invitations[teamID] = []*contextforge.TeamInvitation{}
		}
		invitations[teamID] = append(invitations[teamID], invitation)
		invitationsByID[id] = invitation

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(invitation)

	case http.MethodGet:
		// List invitations
		teamInvitations := invitations[teamID]
		if teamInvitations == nil {
			teamInvitations = []*contextforge.TeamInvitation{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(teamInvitations)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
