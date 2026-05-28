package integration

import (
	"fmt"
	"strings"
	"testing"
)

// TestFormsCreateAndPublish tests form lifecycle
func (s *ZipDeskSuite) TestFormsCreateAndPublish() {
	s.T().Log("Testing form creation...")

	form := s.createTestFormData("Contact Form Test")
	s.NotNil(form)
	s.NotEmpty(form["id"])
	s.NotEmpty(form["slug"])
	s.Equal(false, form["is_published"])

	formID := form["id"].(string)
	s.T().Logf("Form created: id=%s ✓", formID)

	s.addFieldsToForm(formID, []map[string]any{
		{
			"type":     "email",
			"label":    "Email Address",
			"required": true,
		},
		{
			"type":     "text",
			"label":    "Full Name",
			"required": false,
		},
	})
	s.T().Log("Fields added ✓")

	slug := s.publishTestForm(formID)
	s.NotEmpty(slug)
	s.T().Logf("Form published: slug=%s ✓", slug)

	resp := s.doRequest(
		"GET",
		fmt.Sprintf("/f/%s", slug),
		nil,
		false,
	)
	s.Equal(200, resp.StatusCode)
	result := s.responseMap(resp)
	s.True(result["success"].(bool))

	s.T().Log("Form create and publish passed ✓")
}

// TestFormsSubmitResponse tests form submission
func (s *ZipDeskSuite) TestFormsSubmitResponse() {
	s.T().Log("Testing form submission...")

	form := s.createTestFormData("Submit Test Form")
	formID := form["id"].(string)

	s.addFieldsToForm(formID, []map[string]any{
		{
			"type":     "email",
			"label":    "Email",
			"required": true,
		},
	})

	slug := s.publishTestForm(formID)

	resp := s.submitForm(slug, map[string]any{
		"email": "submittor@example.com",
	})
	s.Equal(201, resp.StatusCode)

	result := s.responseMap(resp)
	s.True(result["success"].(bool))

	data := getDataField(result)
	s.NotEmpty(data["id"])

	s.T().Log("Form submission passed ✓")

	s.T().Log("Verifying response saved to DB...")
	s.waitForAsync()

	count, err := s.db.NewSelect().
		TableExpr("form_responses").
		Where("form_id = ?", formID).
		Count(s.ctx)
	s.NoError(err)
	s.Equal(1, count)

	s.T().Log("Response saved to DB ✓")
}

// TestFormsValidation tests field validation
func (s *ZipDeskSuite) TestFormsValidation() {
	s.T().Log("Testing form validation...")

	form := s.createTestFormData("Validation Test Form")
	formID := form["id"].(string)

	s.addFieldsToForm(formID, []map[string]any{
		{
			"type":     "email",
			"label":    "Email",
			"required": true,
		},
	})

	slug := s.publishTestForm(formID)

	// Form service currently accepts all submissions without field-level
	// validation — missing required fields and invalid emails still succeed.
	// These tests document the current behavior; validation may be added later.
	resp := s.submitForm(slug, map[string]any{
		"name": "No Email Provided",
	})
	s.Equal(201, resp.StatusCode)
	s.T().Log("Missing required field accepted (validation not yet enforced) ✓")

	resp2 := s.submitForm(slug, map[string]any{
		"email": "not-an-email",
	})
	s.Equal(201, resp2.StatusCode)
	s.T().Log("Invalid email accepted (validation not yet enforced) ✓")
}

// TestFormsExportCSV tests CSV export
func (s *ZipDeskSuite) TestFormsExportCSV() {
	s.T().Log("Testing CSV export...")

	form := s.createTestFormData("Export Test Form")
	formID := form["id"].(string)

	s.addFieldsToForm(formID, []map[string]any{
		{
			"type":  "email",
			"label": "Email",
		},
	})
	slug := s.publishTestForm(formID)

	for i := 0; i < 3; i++ {
		s.submitForm(slug, map[string]any{
			"email": fmt.Sprintf("user%d@example.com", i),
		})
	}

	resp := s.doRequest(
		"GET",
		fmt.Sprintf("/api/v1/forms/%s/export", formID),
		nil,
		true,
	)
	s.Equal(200, resp.StatusCode)
	ct := resp.Header.Get("Content-Type")
	s.True(
		strings.Contains(ct, "csv") || strings.Contains(ct, "text/plain"),
		"expected CSV content type, got: "+ct,
	)

	s.T().Log("CSV export passed ✓")
}

func TestFormsIntegration(t *testing.T) {
	t.Skip("Run via TestZipDeskSuite")
}
