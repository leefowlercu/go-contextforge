package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestTeamsService_List(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"teams":[{"id":"1","name":"test-team","slug":"test-team","is_personal":false,"member_count":5,"is_active":true,"created_by":"admin@test.local"}],"total":1}`)
	})

	ctx := context.Background()
	teams, _, err := client.Teams.List(ctx, nil)

	if err != nil {
		t.Errorf("Teams.List returned error: %v", err)
	}

	if len(teams) != 1 {
		t.Errorf("Teams.List returned %d teams, want 1", len(teams))
	}

	if teams[0].Name != "test-team" {
		t.Errorf("Teams.List returned team name %q, want %q", teams[0].Name, "test-team")
	}

	if teams[0].MemberCount != 5 {
		t.Errorf("Teams.List returned member count %d, want %d", teams[0].MemberCount, 5)
	}
}

func TestTeamsService_List_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		// Verify query parameters (skip/limit)
		q := r.URL.Query()
		if got := q.Get("skip"); got != "10" {
			t.Errorf("skip = %q, want %q", got, "10")
		}
		if got := q.Get("limit"); got != "20" {
			t.Errorf("limit = %q, want %q", got, "20")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"teams":[],"total":0}`)
	})

	opts := &TeamListOptions{
		Skip:  10,
		Limit: 20,
	}

	ctx := context.Background()
	_, _, err := client.Teams.List(ctx, opts)

	if err != nil {
		t.Errorf("Teams.List returned error: %v", err)
	}
}

func TestTeamsService_Get(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/123/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"test-team","slug":"test-team","is_personal":false,"member_count":5,"is_active":true,"created_by":"admin@test.local"}`)
	})

	ctx := context.Background()
	team, _, err := client.Teams.Get(ctx, "123")

	if err != nil {
		t.Errorf("Teams.Get returned error: %v", err)
	}

	if team.ID != "123" {
		t.Errorf("Teams.Get returned id %q, want %q", team.ID, "123")
	}

	if team.Name != "test-team" {
		t.Errorf("Teams.Get returned name %q, want %q", team.Name, "test-team")
	}
}

func TestTeamsService_Create(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &TeamCreate{
		Name:        "new-team",
		Slug:        String("new-team"),
		Description: String("A test team"),
		Visibility:  String("private"),
		MaxMembers:  Int(10),
	}

	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		var body TeamCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Request body decode error: %v", err)
		}

		if body.Name != "new-team" {
			t.Errorf("Request body name = %q, want %q", body.Name, "new-team")
		}
		if *body.Slug != "new-team" {
			t.Errorf("Request body slug = %q, want %q", *body.Slug, "new-team")
		}
		if *body.Description != "A test team" {
			t.Errorf("Request body description = %q, want %q", *body.Description, "A test team")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":"1","name":"new-team","slug":"new-team","is_personal":false,"member_count":1,"is_active":true,"created_by":"admin@test.local"}`)
	})

	ctx := context.Background()
	team, _, err := client.Teams.Create(ctx, input)

	if err != nil {
		t.Errorf("Teams.Create returned error: %v", err)
	}

	if team.Name != "new-team" {
		t.Errorf("Teams.Create returned name %q, want %q", team.Name, "new-team")
	}
}

func TestTeamsService_Update(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &TeamUpdate{
		Name:        String("updated-name"),
		Description: String("Updated description"),
		Visibility:  String("public"),
	}

	mux.HandleFunc("/teams/123/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")

		var body TeamUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Request body decode error: %v", err)
		}

		if *body.Name != "updated-name" {
			t.Errorf("Request body name = %q, want %q", *body.Name, "updated-name")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"updated-name","slug":"test-team","is_personal":false,"member_count":5,"is_active":true,"created_by":"admin@test.local"}`)
	})

	ctx := context.Background()
	team, _, err := client.Teams.Update(ctx, "123", input)

	if err != nil {
		t.Errorf("Teams.Update returned error: %v", err)
	}

	if team.Name != "updated-name" {
		t.Errorf("Teams.Update returned name %q, want %q", team.Name, "updated-name")
	}
}

func TestTeamsService_Delete(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/123/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.Background()
	_, err := client.Teams.Delete(ctx, "123")

	if err != nil {
		t.Errorf("Teams.Delete returned error: %v", err)
	}
}

func TestTeamsService_ListMembers(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/123/members/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"1","team_id":"123","user_email":"user@test.local","role":"member","is_active":true}]`)
	})

	ctx := context.Background()
	members, _, err := client.Teams.ListMembers(ctx, "123")

	if err != nil {
		t.Errorf("Teams.ListMembers returned error: %v", err)
	}

	if len(members) != 1 {
		t.Errorf("Teams.ListMembers returned %d members, want 1", len(members))
	}

	if members[0].UserEmail != "user@test.local" {
		t.Errorf("Teams.ListMembers returned email %q, want %q", members[0].UserEmail, "user@test.local")
	}
}

func TestTeamsService_UpdateMember(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &TeamMemberUpdate{
		Role: "owner",
	}

	mux.HandleFunc("/teams/123/members/user@test.local/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")

		var body TeamMemberUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Request body decode error: %v", err)
		}

		if body.Role != "owner" {
			t.Errorf("Request body role = %q, want %q", body.Role, "owner")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"1","team_id":"123","user_email":"user@test.local","role":"owner","is_active":true}`)
	})

	ctx := context.Background()
	member, _, err := client.Teams.UpdateMember(ctx, "123", "user@test.local", input)

	if err != nil {
		t.Errorf("Teams.UpdateMember returned error: %v", err)
	}

	if member.Role != "owner" {
		t.Errorf("Teams.UpdateMember returned role %q, want %q", member.Role, "owner")
	}
}

func TestTeamsService_RemoveMember(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/123/members/user@test.local/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.Background()
	_, err := client.Teams.RemoveMember(ctx, "123", "user@test.local")

	if err != nil {
		t.Errorf("Teams.RemoveMember returned error: %v", err)
	}
}

func TestTeamsService_InviteMember(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &TeamInvite{
		Email: "newuser@test.local",
		Role:  String("member"),
	}

	mux.HandleFunc("/teams/123/invitations/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		var body TeamInvite
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Request body decode error: %v", err)
		}

		if body.Email != "newuser@test.local" {
			t.Errorf("Request body email = %q, want %q", body.Email, "newuser@test.local")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":"1","team_id":"123","team_name":"test-team","email":"newuser@test.local","role":"member","invited_by":"admin@test.local","token":"test-token","is_active":true,"is_expired":false}`)
	})

	ctx := context.Background()
	invitation, _, err := client.Teams.InviteMember(ctx, "123", input)

	if err != nil {
		t.Errorf("Teams.InviteMember returned error: %v", err)
	}

	if invitation.Email != "newuser@test.local" {
		t.Errorf("Teams.InviteMember returned email %q, want %q", invitation.Email, "newuser@test.local")
	}
}

func TestTeamsService_ListInvitations(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/123/invitations/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"1","team_id":"123","team_name":"test-team","email":"user@test.local","role":"member","invited_by":"admin@test.local","token":"test-token","is_active":true,"is_expired":false}]`)
	})

	ctx := context.Background()
	invitations, _, err := client.Teams.ListInvitations(ctx, "123")

	if err != nil {
		t.Errorf("Teams.ListInvitations returned error: %v", err)
	}

	if len(invitations) != 1 {
		t.Errorf("Teams.ListInvitations returned %d invitations, want 1", len(invitations))
	}

	if invitations[0].Email != "user@test.local" {
		t.Errorf("Teams.ListInvitations returned email %q, want %q", invitations[0].Email, "user@test.local")
	}
}

func TestTeamsService_AcceptInvitation(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/invitations/test-token/accept/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"1","team_id":"123","user_email":"user@test.local","role":"member","is_active":true}`)
	})

	ctx := context.Background()
	member, _, err := client.Teams.AcceptInvitation(ctx, "test-token")

	if err != nil {
		t.Errorf("Teams.AcceptInvitation returned error: %v", err)
	}

	if member.UserEmail != "user@test.local" {
		t.Errorf("Teams.AcceptInvitation returned email %q, want %q", member.UserEmail, "user@test.local")
	}
}

func TestTeamsService_CancelInvitation(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/invitations/123/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.Background()
	_, err := client.Teams.CancelInvitation(ctx, "123")

	if err != nil {
		t.Errorf("Teams.CancelInvitation returned error: %v", err)
	}
}

func TestTeamsService_Discover(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/discover", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"1","name":"public-team","member_count":10,"is_joinable":true}]`)
	})

	ctx := context.Background()
	teams, _, err := client.Teams.Discover(ctx, nil)

	if err != nil {
		t.Errorf("Teams.Discover returned error: %v", err)
	}

	if len(teams) != 1 {
		t.Errorf("Teams.Discover returned %d teams, want 1", len(teams))
	}

	if teams[0].Name != "public-team" {
		t.Errorf("Teams.Discover returned name %q, want %q", teams[0].Name, "public-team")
	}
}

func TestTeamsService_Discover_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/discover", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		// Verify query parameters
		q := r.URL.Query()
		if got := q.Get("skip"); got != "5" {
			t.Errorf("skip = %q, want %q", got, "5")
		}
		if got := q.Get("limit"); got != "15" {
			t.Errorf("limit = %q, want %q", got, "15")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})

	opts := &TeamDiscoverOptions{
		Skip:  5,
		Limit: 15,
	}

	ctx := context.Background()
	_, _, err := client.Teams.Discover(ctx, opts)

	if err != nil {
		t.Errorf("Teams.Discover returned error: %v", err)
	}
}

func TestTeamsService_Join(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &TeamJoinRequest{
		Message: String("I would like to join"),
	}

	mux.HandleFunc("/teams/123/join/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		var body TeamJoinRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Request body decode error: %v", err)
		}

		if *body.Message != "I would like to join" {
			t.Errorf("Request body message = %q, want %q", *body.Message, "I would like to join")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"1","team_id":"123","team_name":"test-team","user_email":"user@test.local","status":"pending"}`)
	})

	ctx := context.Background()
	joinRequest, _, err := client.Teams.Join(ctx, "123", input)

	if err != nil {
		t.Errorf("Teams.Join returned error: %v", err)
	}

	if joinRequest.Status != "pending" {
		t.Errorf("Teams.Join returned status %q, want %q", joinRequest.Status, "pending")
	}
}

func TestTeamsService_Leave(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/123/leave/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.Background()
	_, err := client.Teams.Leave(ctx, "123")

	if err != nil {
		t.Errorf("Teams.Leave returned error: %v", err)
	}
}

func TestTeamsService_ListJoinRequests(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/123/join-requests/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"1","team_id":"123","team_name":"test-team","user_email":"user@test.local","status":"pending"}]`)
	})

	ctx := context.Background()
	requests, _, err := client.Teams.ListJoinRequests(ctx, "123")

	if err != nil {
		t.Errorf("Teams.ListJoinRequests returned error: %v", err)
	}

	if len(requests) != 1 {
		t.Errorf("Teams.ListJoinRequests returned %d requests, want 1", len(requests))
	}

	if requests[0].Status != "pending" {
		t.Errorf("Teams.ListJoinRequests returned status %q, want %q", requests[0].Status, "pending")
	}
}

func TestTeamsService_ApproveJoinRequest(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/123/join-requests/456/approve/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"1","team_id":"123","user_email":"user@test.local","role":"member","is_active":true}`)
	})

	ctx := context.Background()
	member, _, err := client.Teams.ApproveJoinRequest(ctx, "123", "456")

	if err != nil {
		t.Errorf("Teams.ApproveJoinRequest returned error: %v", err)
	}

	if member.UserEmail != "user@test.local" {
		t.Errorf("Teams.ApproveJoinRequest returned email %q, want %q", member.UserEmail, "user@test.local")
	}
}

func TestTeamsService_RejectJoinRequest(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/123/join-requests/456/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.Background()
	_, err := client.Teams.RejectJoinRequest(ctx, "123", "456")

	if err != nil {
		t.Errorf("Teams.RejectJoinRequest returned error: %v", err)
	}
}
