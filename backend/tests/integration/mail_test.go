package integration

import (
	"fmt"
	"testing"
)

// TestMailContactUpsert tests contact upsert
func (s *ZipDeskSuite) TestMailContactUpsert() {
	s.T().Log("Testing mail contact upsert...")

	contactID, err := s.mailSvc.UpsertContact(
		s.ctx,
		s.workspaceID,
		"upsert@example.com",
		map[string]any{
			"source": "test",
			"name":   "Upsert User",
		},
	)
	s.NoError(err)
	s.NotEmpty(contactID)

	contactID2, err := s.mailSvc.UpsertContact(
		s.ctx,
		s.workspaceID,
		"upsert@example.com",
		map[string]any{
			"source": "form",
			"name":   "Updated Name",
		},
	)
	s.NoError(err)
	s.NotEmpty(contactID2)

	contact := s.getMailContact("upsert@example.com")
	s.NotNil(contact)
	s.Equal("upsert@example.com", contact.Email)

	s.T().Log("Mail contact upsert passed ✓")
}

// TestMailContactAPI tests REST API
func (s *ZipDeskSuite) TestMailContactAPI() {
	s.T().Log("Testing mail contacts API...")

	resp := s.doRequest("POST", "/api/v1/mail/contacts", map[string]any{
		"email":      "api-contact@example.com",
		"first_name": "API",
		"last_name":  "User",
		"source":     "api",
	}, true)
	s.Equal(201, resp.StatusCode)

	result := s.responseMap(resp)
	s.True(result["success"].(bool))

	data := getDataField(result)
	s.Equal("api-contact@example.com", data["email"])

	resp2 := s.doRequest("GET", "/api/v1/mail/contacts", nil, true)
	s.Equal(200, resp2.StatusCode)

	resp3 := s.doRequest("POST", "/api/v1/mail/contacts", map[string]any{
		"email": "api-contact@example.com",
	}, true)
	s.Equal(409, resp3.StatusCode)

	s.T().Log("Mail contact API passed ✓")
}

// TestMailCampaignLifecycle tests campaigns
func (s *ZipDeskSuite) TestMailCampaignLifecycle() {
	s.T().Log("Testing mail campaign lifecycle...")

	resp := s.doRequest("POST", "/api/v1/mail/campaigns", map[string]any{
		"name":       "Test Campaign",
		"subject":    "Hello from ZipDesk",
		"from_name":  "ZipDesk",
		"from_email": "hello@zipdesk.io",
		"content": map[string]any{
			"html": "<h1>Hello!</h1>",
			"text": "Hello!",
		},
	}, true)
	s.Equal(201, resp.StatusCode)

	result := s.responseMap(resp)
	data := getDataField(result)
	s.NotEmpty(data["id"])
	s.Equal("draft", data["status"])

	campaignID := data["id"].(string)

	resp2 := s.doRequest("GET", fmt.Sprintf("/api/v1/mail/campaigns/%s/stats", campaignID), nil, true)
	s.Equal(200, resp2.StatusCode)

	s.T().Log("Campaign lifecycle passed ✓")
}

func TestMailIntegration(t *testing.T) {
	t.Skip("Run via TestZipDeskSuite")
}
