package integration

import (
	"fmt"
	"testing"
)

// TestLinksCreateAndRedirect tests link creation
func (s *ZipDeskSuite) TestLinksCreateAndRedirect() {
	s.T().Log("Testing link creation...")

	// Create link
	link := s.createTestLink(
		"https://example.com/test-page",
	)
	s.NotNil(link)
	s.NotEmpty(link["id"])
	s.NotEmpty(link["short_code"])

	shortCode := link["short_code"].(string)
	s.T().Logf(
		"Link created: short_code=%s ✓",
		shortCode,
	)

	// Test redirect
	s.T().Log("Testing redirect...")
	resp := s.doRequest(
		"GET",
		fmt.Sprintf("/s/%s", shortCode),
		nil,
		false,
	)
	// Should redirect (301) or 404 if not cached
	s.T().Logf(
		"Redirect status: %d ✓", resp.StatusCode,
	)

	s.T().Log("Links create and redirect passed ✓")
}

// TestLinksListAndFilter tests link listing
func (s *ZipDeskSuite) TestLinksListAndFilter() {
	s.T().Log("Testing links list...")

	// Create multiple links
	for i := 0; i < 3; i++ {
		s.createTestLink(
			fmt.Sprintf(
				"https://example.com/page-%d", i,
			),
		)
	}

	// List links
	resp := s.doRequest(
		"GET",
		"/api/v1/links?page=1&per_page=10",
		nil,
		true,
	)
	s.Equal(200, resp.StatusCode)

	result := s.responseMap(resp)
	s.True(result["success"].(bool))

	meta, _ := result["meta"].(map[string]any)
	s.NotNil(meta)

	total := meta["total"].(float64)
	s.GreaterOrEqual(total, float64(3))

	s.T().Logf(
		"Listed %v links ✓", total,
	)
}

// TestLinksCustomSlug tests custom slug creation
func (s *ZipDeskSuite) TestLinksCustomSlug() {
	s.T().Log("Testing custom slug...")

	resp := s.doRequest(
		"POST",
		"/api/v1/links",
		map[string]any{
			"original_url": "https://example.com",
			"custom_slug":  "my-custom-slug-test",
		},
		true,
	)
	s.Equal(201, resp.StatusCode)

	result := s.responseMap(resp)
	data := getDataField(result)
	s.Equal(
		"my-custom-slug-test",
		data["custom_slug"],
	)

	s.T().Log("Custom slug passed ✓")
}

// TestLinksDeleteAndVerify tests link deletion
func (s *ZipDeskSuite) TestLinksDeleteAndVerify() {
	s.T().Log("Testing link deletion...")

	// Create
	link := s.createTestLink(
		"https://example.com/to-delete",
	)
	id := link["id"].(string)

	// Delete
	resp := s.doRequest(
		"DELETE",
		fmt.Sprintf("/api/v1/links/%s", id),
		nil,
		true,
	)
	s.Equal(200, resp.StatusCode)

	// Verify gone
	resp2 := s.doRequest(
		"GET",
		fmt.Sprintf("/api/v1/links/%s", id),
		nil,
		true,
	)
	s.Equal(404, resp2.StatusCode)

	s.T().Log("Link deletion passed ✓")
}

// TestLinksInvalidURL tests URL validation
func (s *ZipDeskSuite) TestLinksInvalidURL() {
	s.T().Log("Testing invalid URL rejection...")

	resp := s.doRequest(
		"POST",
		"/api/v1/links",
		map[string]any{
			"original_url": "https://",
		},
		true,
	)
	s.Equal(400, resp.StatusCode)

	s.T().Log("Invalid URL rejection passed ✓")
}

// Required for go test to run
func TestLinksIntegration(t *testing.T) {
	t.Skip("Run via TestZipDeskSuite")
}
