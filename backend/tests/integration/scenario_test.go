package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// =============================================================================
// Hackathon Demo: 5-Step Integration Test
// =============================================================================
// Step 1: Register (or login) → get auth token + workspace ID
// Step 2: Create a form   → get form slug
// Step 3: Submit the form (public) → triggers event
// Step 4: Verify mail contact was created by the trigger
// Step 5: Verify events were created (form.submitted + mail.contact_added)

func (s *ZipDeskSuite) TestFullHackathonDemo() {
	s.T().Logf("Step 1: Already registered user=%s workspace=%s",
		s.userID, s.workspaceID)

	// ---- Step 2: Create a Form ----
	s.T().Log("Step 2: Creating form via POST /api/v1/forms")

	createPayload := `{
		"title": "Waitlist Signup %d",
		"fields": [
			{"name": "email", "type": "email", "label": "Email Address", "required": true},
			{"name": "name", "type": "text", "label": "Full Name", "required": false}
		]
	}`

	req := s.createRequest("POST", "/api/v1/forms", fmt.Sprintf(createPayload, time.Now().UnixMilli()))
	resp, err := s.app.Test(req, 5000)
	s.Require().NoError(err)
	s.Require().Equal(fiber.StatusCreated, resp.StatusCode,
		"expected 201 when creating form")

	formResult := s.parseResponse(resp)
	s.True(formResult["success"].(bool), "form creation should succeed")

	formData := formResult["data"].(map[string]any)
	formID := formData["id"].(string)
	formSlug := formData["slug"].(string)
	s.T().Logf("  Created form: id=%s slug=%s", formID, formSlug)

	// Verify form exists in DB
	dbCount, _ := s.db.NewSelect().TableExpr("forms").Where("id = ?", formID).Count(s.ctx)
	s.T().Logf("  Form exists in DB count: %d", dbCount)

	// Publish the form
	s.T().Logf("  Form ID: %s, Workspace ID: %s", formID, s.workspaceID)
	pubReq := s.createRequest("POST", fmt.Sprintf("/api/v1/forms/%s/publish", formID), "")
	pubResp, err := s.app.Test(pubReq, 5000)
	s.Require().NoError(err)
	pubResult := s.parseResponse(pubResp)
	s.T().Logf("  Publish response: %+v", pubResult)
	s.Require().Equal(fiber.StatusOK, pubResp.StatusCode, "expected 200 when publishing form")
	s.T().Log("  Form published")

	// ---- Step 3: Submit the form (public endpoint) ----
	s.T().Log("Step 3: Submitting form via public submit endpoint")

	submitPayload := `{
		"data": {
			"email": "test@example.com",
			"name": "Jane Doe"
		}
	}`

	submitPath := fmt.Sprintf("/f/%s/submit", formSlug)
	submitReq := s.createRequest("POST", submitPath, submitPayload)
	submitResp, err := s.app.Test(submitReq, 5000)
	s.Require().NoError(err)
	s.Require().Equal(fiber.StatusCreated, submitResp.StatusCode,
		"expected 201 when submitting form")

	submitResult := s.parseResponse(submitResp)
	s.True(submitResult["success"].(bool), "form submission should succeed")
	s.T().Log("  Form submitted successfully")

	// Allow async triggers to fire
	s.waitForAsync()

	// ---- Step 4: Verify mail contact was created ----
	s.T().Log("Step 4: Verifying mail contact was created")

	contact := s.getMailContact("test@example.com")
	s.Require().NotNil(contact,
		"mail contact should exist after form submit")
	s.Equal("test@example.com", contact.Email)
	s.T().Logf("  Found mail contact: id=%s email=%s", contact.ID, contact.Email)

	// ---- Step 5: Verify events were created ----
	s.T().Log("Step 5: Verifying events")

	formEvents := s.getEvents("form.submitted")
	s.Require().NotEmpty(formEvents,
		"should have at least one form.submitted event")
	s.T().Logf("  Found %d form.submitted event(s)", len(formEvents))

	mailEvents := s.getEvents("mail.contact_added")
	s.Require().NotEmpty(mailEvents,
		"should have at least one mail.contact_added event")
	s.T().Logf("  Found %d mail.contact_added event(s)", len(mailEvents))

	crmEvents := s.getEvents("crm.contact_created")
	s.T().Logf("  Found %d crm.contact_created event(s)", len(crmEvents))

	s.T().Log("=== HACKATHON DEMO: ALL 5 STEPS PASSED ===")
}

// ---------------------------------------------------------------------------
// Additional edge-case tests
// ---------------------------------------------------------------------------

func (s *ZipDeskSuite) TestDuplicateContact() {
	slug := s.createTestForm("Duplicate Test Form")

	for i := 0; i < 2; i++ {
		payload := `{"data": {"email": "dupe@example.com"}}`
		path := fmt.Sprintf("/f/%s/submit", slug)
		req := s.createRequest("POST", path, payload)
		resp, err := s.app.Test(req, 5000)
		s.Require().NoError(err)
	s.Require().Equal(fiber.StatusCreated, resp.StatusCode,
		"duplicate submission should still succeed")
	}

	s.waitForAsync()

	contact := s.getMailContact("dupe@example.com")
	s.Require().NotNil(contact, "contact should exist after duplicate submit")
}

func (s *ZipDeskSuite) TestMissingEmailField() {
	slug := s.createTestForm("Missing Email Test")

	payload := `{"data": {"name": "No Email"}}`
	path := fmt.Sprintf("/f/%s/submit", slug)
	req := s.createRequest("POST", path, payload)
	resp, err := s.app.Test(req, 5000)
	s.Require().NoError(err)
	s.T().Logf("  Missing email response: %d", resp.StatusCode)
}

func (s *ZipDeskSuite) TestDocumentCRUD() {
	createPayload := `{
		"title": "Test Proposal",
		"type": "proposal",
		"content": {
			"blocks": [
				{"id": "b1", "type": "text", "content": "Hello world"}
			],
			"variables": {}
		},
		"settings": {"allow_download": true, "require_email": false, "password": "", "watermark": ""}
	}`

	req := s.createRequest("POST", "/api/v1/docs/", createPayload)
	resp, err := s.app.Test(req, 5000)
	s.Require().NoError(err)
	s.Require().Equal(fiber.StatusCreated, resp.StatusCode)

	result := s.parseResponse(resp)
	doc := result["data"].(map[string]any)
	docID := doc["id"].(string)
	s.T().Logf("  Created document: id=%s", docID)

	listReq := s.createRequest("GET", "/api/v1/docs/", "")
	listResp, err := s.app.Test(listReq, 5000)
	s.Require().NoError(err)
	s.Require().Equal(fiber.StatusOK, listResp.StatusCode)

	pubReq := s.createRequest("POST", fmt.Sprintf("/api/v1/docs/%s/publish", docID), "")
	pubResp, err := s.app.Test(pubReq, 5000)
	s.Require().NoError(err)
	s.Require().Equal(fiber.StatusOK, pubResp.StatusCode)

	pubResult := s.parseResponse(pubResp)
	pubDoc := pubResult["data"].(map[string]any)
	slug := pubDoc["slug"].(string)

	publicReq := s.createRequest("GET", fmt.Sprintf("/d/%s", slug), "")
	publicResp, err := s.app.Test(publicReq, 5000)
	s.Require().NoError(err)
	s.Require().Equal(fiber.StatusOK, publicResp.StatusCode)

	s.T().Logf("  Published doc accessible at /d/%s", slug)
}

// createTestForm creates a form with an email field and returns its slug
func (s *ZipDeskSuite) createTestForm(name string) string {
	payload := fmt.Sprintf(`{
		"title": "%s %d",
		"fields": [
			{"name": "email", "type": "email", "label": "Email", "required": true}
		]
	}`, name, time.Now().UnixMilli())

	req := s.createRequest("POST", "/api/v1/forms", payload)
	resp, err := s.app.Test(req, 5000)
	s.Require().NoError(err)
	s.Require().Equal(fiber.StatusCreated, resp.StatusCode)

	result := s.parseResponse(resp)
	form := result["data"].(map[string]any)
	formID := form["id"].(string)

	pubReq := s.createRequest("POST", fmt.Sprintf("/api/v1/forms/%s/publish", formID), "")
	pubResp, err := s.app.Test(pubReq, 5000)
	s.Require().NoError(err)
	s.Require().Equal(fiber.StatusOK, pubResp.StatusCode)

	return form["slug"].(string)
}

// createRequest builds a test HTTP request with auth headers
func (s *ZipDeskSuite) createRequest(method, path, body string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	if s.authToken != "" {
		req.Header.Set("Authorization", s.authHeader())
	}
	req.Header.Set("X-Workspace-ID", s.workspaceID)
	return req
}

// parseResponse unmarshals a JSON response
func (s *ZipDeskSuite) parseResponse(resp *http.Response) map[string]any {
	var result map[string]any
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.Require().NoError(err, "should parse JSON response")
	return result
}
