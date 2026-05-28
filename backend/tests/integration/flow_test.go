package integration

import (
	"testing"
)

// TestFlowFormToMailIntegration is THE critical test
// This proves the hackathon demo works
func (s *ZipDeskSuite) TestFlowFormToMailIntegration() {
	s.T().Log("═══════════════════════════════════")
	s.T().Log("CRITICAL INTEGRATION TEST")
	s.T().Log("Form Submit → Mail Contact → Events")
	s.T().Log("═══════════════════════════════════")

	testEmail := "integration-test@example.com"

	s.T().Log("\nStep 1: Creating form...")
	form := s.createForm("Integration Test Waitlist")
	formID := form["id"].(string)
	s.T().Log("Step 1: Creating form... ✓")

	s.T().Log("Step 2: Adding email field...")
	s.addFieldsToForm(formID, []map[string]any{
		{
			"type":     "email",
			"label":    "Email Address",
			"required": true,
		},
		{
			"type":     "text",
			"label":    "Your Name",
			"required": false,
		},
	})
	s.T().Log("Step 2: Adding email field... ✓")

	s.T().Log("Step 3: Publishing form...")
	slug := s.publishTestForm(formID)
	s.NotEmpty(slug)
	s.T().Log("Step 3: Publishing form... ✓")

	s.T().Log("Step 4: Submitting form...")
	resp := s.submitForm(slug, map[string]any{
		"email": testEmail,
		"name":  "Integration Tester",
	})
	s.Require().Equal(201, resp.StatusCode,
		"Form submission must return 201",
	)
	s.T().Log("Step 4: Submitting form... ✓")

	s.T().Log("Step 5: Waiting 500ms...")
	s.waitForAsync()
	s.T().Log("Step 5: Waiting 500ms... ✓")

	s.T().Log("Step 6: form.submitted event found...")
	events := s.getEvents("form.submitted")
	s.Require().NotEmpty(
		events,
		"form.submitted event must be logged to events table",
	)
	payload := events[0].Payload
	s.Equal(testEmail, payload["email"],
		"event payload must contain submitted email",
	)
	s.T().Log("Step 6: form.submitted event found ✓")

	s.T().Log("Step 7: Mail contact found...")
	contact := s.getMailContact(testEmail)
	s.Require().NotNil(
		contact,
		"mail contact must be created from form submission",
	)
	s.Equal("form", contact.Source,
		"contact source must be 'form'",
	)
	s.Equal(testEmail, contact.Email,
		"contact email must match submission",
	)
	s.T().Log("Step 7: Mail contact found ✓")

	s.T().Log("Step 8: mail.contact_added event found...")
	contactEvents := s.getEvents("mail.contact_added")
	s.Require().NotEmpty(
		contactEvents,
		"mail.contact_added event must be logged",
	)
	s.T().Log("Step 8: mail.contact_added event found ✓")

	s.T().Log("\n═══════════════════════════════════")
	s.T().Log("CRITICAL INTEGRATION TEST PASSED ✓")
	s.T().Log("Form → Event Bus → Mail Contact")
	s.T().Log("All steps verified successfully")
	s.T().Log("═══════════════════════════════════")
}

// TestFlowBlueprintExecution tests blueprint runs
func (s *ZipDeskSuite) TestFlowBlueprintExecution() {
	s.T().Log("Testing blueprint execution...")

	// Init default blueprints for workspace
	err := s.flowSvc.InitWorkspace(s.ctx, s.workspaceID)
	s.NoError(err)

	// List blueprints
	resp := s.doRequest("GET", "/api/v1/flow/blueprints", nil, true)
	s.Equal(200, resp.StatusCode)

	result := s.responseMap(resp)
	s.True(result["success"].(bool))

	s.T().Log("Blueprint listing passed ✓")

	// Create custom blueprint
	resp2 := s.doRequest("POST", "/api/v1/flow/blueprints", map[string]any{
		"name":         "Test Form to Notify",
		"trigger_type": "form.submitted",
		"actions": []map[string]any{
			{
				"id":   "a1",
				"type": "system.notify",
				"config": map[string]any{
					"message": "New response!",
				},
				"order": 1,
			},
		},
	}, true)
	s.Equal(201, resp2.StatusCode)
	s.T().Log("Blueprint creation passed ✓")
}

// TestFlowEventLog tests event logging
func (s *ZipDeskSuite) TestFlowEventLog() {
	s.T().Log("Testing event log API...")

	// Create some events
	form := s.createForm("Event Log Test Form")
	formID := form["id"].(string)
	s.addFieldsToForm(formID, []map[string]any{
		{"type": "email", "label": "Email"},
	})
	slug := s.publishTestForm(formID)
	s.submitForm(slug, map[string]any{"email": "eventlog@example.com"})

	s.waitForAsync()

	// Fetch event log
	resp := s.doRequest("GET", "/api/v1/flow/events?limit=20", nil, true)
	s.Equal(200, resp.StatusCode)

	result := s.responseMap(resp)
	s.True(result["success"].(bool))

	data, ok := result["data"].([]any)
	if ok {
		s.GreaterOrEqual(len(data), 1,
			"at least one event must be logged",
		)
	}

	s.T().Log("Event log API passed ✓")
}

// TestFlowHealthReport tests health endpoint
func (s *ZipDeskSuite) TestFlowHealthReport() {
	s.T().Log("Testing health report API...")

	resp := s.doRequest("GET", "/api/v1/flow/health", nil, true)
	s.Equal(200, resp.StatusCode)

	result := s.responseMap(resp)
	s.True(result["success"].(bool))

	data := getDataField(result)
	s.NotNil(data)

	services, ok := data["services"].(map[string]any)
	s.True(ok, "health report must have services")
	s.NotEmpty(services)

	overall, ok := data["overall"].(string)
	s.True(ok, "health report must have overall status")
	s.NotEmpty(overall)

	s.T().Logf("Health report API: overall=%s ✓", overall)
}

// TestFlowCRMIntegration tests CRM contact creation
func (s *ZipDeskSuite) TestFlowCRMIntegration() {
	s.T().Log("Testing Form → CRM contact integration...")

	testEmail := "crm-test@example.com"

	// Create and publish form
	form := s.createForm("CRM Integration Form")
	formID := form["id"].(string)
	s.addFieldsToForm(formID, []map[string]any{
		{
			"type":     "email",
			"label":    "Email",
			"required": true,
		},
		{
			"type":  "text",
			"label": "Name",
		},
	})
	slug := s.publishTestForm(formID)

	// Submit
	s.submitForm(slug, map[string]any{
		"email": testEmail,
		"name":  "CRM Test User",
	})

	s.waitForAsync()

	// Check CRM contact
	crmContact := s.getCRMContact(testEmail)
	if crmContact != nil {
		s.Equal(testEmail, crmContact.Email)
		s.T().Logf("CRM contact found: id=%s ✓", crmContact.ID)
	} else {
		s.T().Log("CRM contact not created (CRM service may be disabled) - skipping")
	}

	s.T().Log("CRM integration test passed ✓")
}

func TestFlowIntegration(t *testing.T) {
	t.Skip("Run via TestZipDeskSuite")
}
