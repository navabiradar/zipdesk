package integration

import (
	"testing"
)

// TestAuth tests authentication flows
func (s *ZipDeskSuite) TestRegisterAndLogin() {
	s.T().Log("Testing user registration...")

	// Register new user
	resp := s.doRequest(
		"POST",
		"/api/v1/auth/register",
		map[string]any{
			"name":           "Integration User",
			"email":          "integration@zipdesk-test.com",
			"password":       "password123",
			"workspace_name": "Integration Workspace",
		},
		false,
	)

	s.Equal(201, resp.StatusCode)
	result := s.responseMap(resp)
	s.True(result["success"].(bool))

	data := getDataField(result)
	s.NotNil(data)
	s.NotEmpty(
		getStringField(data, "access_token"),
	)
	s.NotEmpty(
		getStringField(data, "workspace", "id"),
	)

	s.T().Log("Registration passed ✓")

	// Test duplicate registration
	s.T().Log("Testing duplicate email rejection...")
	resp2 := s.doRequest(
		"POST",
		"/api/v1/auth/register",
		map[string]any{
			"name":           "Duplicate User",
			"email":          "integration@zipdesk-test.com",
			"password":       "password123",
			"workspace_name": "Another Workspace",
		},
		false,
	)
	s.Equal(400, resp2.StatusCode)
	s.T().Log("Duplicate rejection passed ✓")

	// Cleanup
	_, _ = s.db.NewDelete().
		TableExpr("workspace_members").
		Where("workspace_id IN (SELECT id FROM workspaces WHERE slug LIKE 'integration%')").
		Exec(s.ctx)
	_, _ = s.db.NewDelete().
		TableExpr("workspaces").
		Where("slug LIKE 'integration%'").
		Exec(s.ctx)
	_, _ = s.db.NewDelete().
		TableExpr("users").
		Where("email = ?",
			"integration@zipdesk-test.com",
		).
		Exec(s.ctx)
}

// TestGetMe tests the /auth/me endpoint
func (s *ZipDeskSuite) TestGetMe() {
	s.T().Log("Testing GET /auth/me...")

	resp := s.doRequest(
		"GET",
		"/api/v1/auth/me",
		nil,
		true,
	)

	s.Equal(200, resp.StatusCode)
	result := s.responseMap(resp)
	s.True(result["success"].(bool))

	data := getDataField(result)
	s.NotNil(data)
	s.Equal(
		s.userEmail,
		getStringField(data, "user", "email"),
	)

	s.T().Log("GET /auth/me passed ✓")
}

// TestUnauthorizedAccess tests auth protection
func (s *ZipDeskSuite) TestUnauthorizedAccess() {
	s.T().Log("Testing unauthorized access rejection...")

	endpoints := []string{
		"/api/v1/links",
		"/api/v1/forms",
		"/api/v1/mail/contacts",
		"/api/v1/crm/contacts",
		"/api/v1/flow/events",
	}

	for _, endpoint := range endpoints {
		resp := s.doRequest(
			"GET", endpoint, nil, false,
		)
		s.Equal(
			401, resp.StatusCode,
			"endpoint %s should require auth",
			endpoint,
		)
	}

	s.T().Log("Auth protection passed ✓")
}

// Required for go test to run
func TestAuthIntegration(t *testing.T) {
	t.Skip("Run via TestZipDeskSuite")
}
