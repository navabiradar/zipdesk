package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// TestFullHackathonDemo is the E2E demo test
func TestFullHackathonDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E in short mode")
	}

	baseURL := os.Getenv("API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	t.Log("╔══════════════════════════════════════╗")
	t.Log("║  ZIPDESK HACKATHON DEMO E2E TEST     ║")
	t.Log("║  Form → Event Bus → Mail → Events    ║")
	t.Log("╚══════════════════════════════════════╝")

	ctx := context.Background()
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// STEP 1: Register user
	t.Log("\n[1/8] Registering test user...")

	ts := time.Now().Unix()
	regPayload := map[string]any{
		"name":           "Demo User",
		"email":          fmt.Sprintf("demo-%d@zipdesk-e2e.com", ts),
		"password":       "demopassword123",
		"workspace_name": fmt.Sprintf("Demo Workspace %d", ts),
	}

	regResp := doPost(t, client, baseURL+"/api/v1/auth/register", regPayload, "")
	require.Equal(t, 201, regResp["status"])

	data := regResp["data"].(map[string]any)
	accessToken := data["access_token"].(string)
	workspace := data["workspace"].(map[string]any)
	workspaceID := workspace["id"].(string)

	t.Logf("  ✓ User registered, workspace: %s", workspaceID)

	// STEP 2: Create form
	t.Log("\n[2/8] Creating waitlist form...")

	formResp := doPost(t, client, baseURL+"/api/v1/forms", map[string]any{
		"title":       fmt.Sprintf("ZipDesk Waitlist %d", ts),
		"description": "Join our waitlist",
	}, accessToken)
	require.Equal(t, 201, formResp["status"])

	formData := formResp["data"].(map[string]any)
	formID := formData["id"].(string)

	t.Logf("  ✓ Form created: %s", formID)

	// STEP 3: Add email field
	t.Log("\n[3/8] Adding email field...")

	doPut(t, client, fmt.Sprintf("%s/api/v1/forms/%s", baseURL, formID), map[string]any{
		"fields": []map[string]any{
			{
				"type":     "email",
				"label":    "Email Address",
				"required": true,
			},
			{
				"type":  "text",
				"label": "Your Name",
			},
		},
	}, accessToken)

	t.Log("  ✓ Email field added")

	// STEP 4: Publish form
	t.Log("\n[4/8] Publishing form...")

	pubResp := doPost(t, client, fmt.Sprintf("%s/api/v1/forms/%s/publish", baseURL, formID), nil, accessToken)
	require.Equal(t, 200, pubResp["status"])

	pubData := pubResp["data"].(map[string]any)
	formSlug := pubData["slug"].(string)

	t.Logf("  ✓ Form published: slug=%s", formSlug)

	// STEP 5: Submit form
	demoEmail := fmt.Sprintf("judge-%d@hackathon.com", time.Now().Unix())

	t.Logf("\n[5/8] Submitting form as: %s", demoEmail)

	submitResp := doPost(t, client, fmt.Sprintf("%s/f/%s/submit", baseURL, formSlug), map[string]any{
		"data": map[string]any{
			"email": demoEmail,
			"name":  "Hackathon Judge",
		},
	}, "")
	require.Equal(t, 201, submitResp["status"])
	t.Log("  ✓ Form submitted successfully")

	// STEP 6: Wait for event processing
	t.Log("\n[6/8] Waiting for event processing...")
	time.Sleep(500 * time.Millisecond)
	t.Log("  ✓ Waited 500ms")

	// STEP 7: Verify events table
	t.Log("\n[7/8] Verifying events table...")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:test@localhost:5432/zipdesk_test?sslmode=disable"
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dbURL)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	var formEvents []struct {
		ID          string         `bun:"id"`
		Type        string         `bun:"type"`
		WorkspaceID string         `bun:"workspace_id"`
		Source      string         `bun:"source"`
		Payload     map[string]any `bun:"payload"`
		OccurredAt  time.Time      `bun:"occurred_at"`
	}

	err := db.NewSelect().TableExpr("events").
		Column("id", "type", "workspace_id", "source", "payload", "occurred_at").
		Where("workspace_id = ? AND type = ?", workspaceID, "form.submitted").
		Scan(ctx, &formEvents)

	require.NoError(t, err)
	require.NotEmpty(t, formEvents, "form.submitted event must exist in events table")

	t.Logf("  ✓ form.submitted event logged (id=%s)", formEvents[0].ID)

	emailInPayload, _ := formEvents[0].Payload["email"].(string)
	require.Equal(t, demoEmail, emailInPayload, "event payload must contain submitted email")
	t.Log("  ✓ Email found in event payload")

	// STEP 8: Verify mail contact created
	t.Log("\n[8/8] Verifying mail contact created...")

	var contacts []struct {
		ID          string `bun:"id"`
		Email       string `bun:"email"`
		Source      string `bun:"source"`
		WorkspaceID string `bun:"workspace_id"`
	}

	err = db.NewSelect().TableExpr("mail_contacts").
		Column("id", "email", "source", "workspace_id").
		Where("workspace_id = ? AND email = ?", workspaceID, demoEmail).
		Scan(ctx, &contacts)

	require.NoError(t, err)
	require.NotEmpty(t, contacts, "mail contact must be created from form submission")

	t.Logf("  ✓ Mail contact created (id=%s)", contacts[0].ID)
	t.Logf("  ✓ Contact source: %s", contacts[0].Source)

	require.Equal(t, demoEmail, contacts[0].Email, "contact email must match submission")

	t.Log("\n╔══════════════════════════════════════╗")
	t.Log("║  ALL STEPS PASSED ✓                  ║")
	t.Log("║  HACKATHON DEMO IS READY             ║")
	t.Log("╚══════════════════════════════════════╝")

	t.Log("\nSummary:")
	t.Log("  → User registered")
	t.Log("  → Form created with email field")
	t.Log("  → Form published")
	t.Log("  → Form submitted")
	t.Log("  → form.submitted event logged")
	t.Log("  → Mail contact auto-created")
}

func doPost(t *testing.T, client *http.Client, url string, body map[string]any, token string) map[string]any {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	result["status"] = resp.StatusCode
	return result
}

func doPut(t *testing.T, client *http.Client, url string, body map[string]any, token string) map[string]any {
	reqBody, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result map[string]any
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	result["status"] = resp.StatusCode
	return result
}
