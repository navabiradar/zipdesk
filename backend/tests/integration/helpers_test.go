package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// doRequest makes HTTP request to test app
func (s *ZipDeskSuite) doRequest(
	method string,
	path string,
	body any,
	auth bool,
) *http.Response {
	var reqBody io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		s.Require().NoError(err)
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, path, reqBody)
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")
	if auth {
		req.Header.Set(
			"Authorization",
			s.authHeader(),
		)
	}

	resp, err := s.app.Test(req, 10000)
	s.Require().NoError(err)

	return resp
}

// parseResponse parses JSON response body
func parseResponse(
	t *testing.T,
	resp *http.Response,
	dest any,
) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf(
			"failed to parse JSON: %v\nbody: %s",
			err, string(body),
		)
	}
}

// responseMap parses response into generic map
func (s *ZipDeskSuite) responseMap(
	resp *http.Response,
) map[string]any {
	defer resp.Body.Close()
	var result map[string]any
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &result)
	return result
}

// getDataField extracts data field from response
func getDataField(
	result map[string]any,
) map[string]any {
	if data, ok := result["data"].(map[string]any); ok {
		return data
	}
	return nil
}

// getStringField extracts a string from nested map
func getStringField(
	m map[string]any,
	keys ...string,
) string {
	current := m
	for i, key := range keys {
		if i == len(keys)-1 {
			if val, ok := current[key].(string); ok {
				return val
			}
			return ""
		}
		if next, ok := current[key].(map[string]any); ok {
			current = next
		} else {
			return ""
		}
	}
	return ""
}

// createForm creates a form via API and returns data (alias for createTestFormData)
func (s *ZipDeskSuite) createForm(
	title string,
) map[string]any {
	return s.createTestFormData(title)
}

// createTestFormData creates a form via API and returns data
func (s *ZipDeskSuite) createTestFormData(
	title string,
) map[string]any {
	resp := s.doRequest(
		"POST",
		"/api/v1/forms",
		map[string]any{
			"title":       title,
			"description": "Test form",
		},
		true,
	)
	s.Require().Equal(201, resp.StatusCode)
	result := s.responseMap(resp)
	data := getDataField(result)
	s.Require().NotNil(data)
	return data
}

// addFieldsToForm adds fields via API
func (s *ZipDeskSuite) addFieldsToForm(
	formID string,
	fields []map[string]any,
) {
	resp := s.doRequest(
		"PUT",
		fmt.Sprintf("/api/v1/forms/%s", formID),
		map[string]any{
			"fields": fields,
		},
		true,
	)
	s.Require().Equal(200, resp.StatusCode)
}

// publishTestForm publishes a form via API
func (s *ZipDeskSuite) publishTestForm(
	formID string,
) string {
	resp := s.doRequest(
		"POST",
		fmt.Sprintf(
			"/api/v1/forms/%s/publish",
			formID,
		),
		nil,
		true,
	)
	s.Require().Equal(200, resp.StatusCode)
	result := s.responseMap(resp)
	data := getDataField(result)
	return getStringField(data, "slug")
}

// submitForm submits a form via public API
func (s *ZipDeskSuite) submitForm(
	slug string,
	data map[string]any,
) *http.Response {
	return s.doRequest(
		"POST",
		fmt.Sprintf("/f/%s/submit", slug),
		map[string]any{"data": data},
		false,
	)
}

// createTestLink creates a link via API
func (s *ZipDeskSuite) createTestLink(
	originalURL string,
) map[string]any {
	resp := s.doRequest(
		"POST",
		"/api/v1/links",
		map[string]any{
			"original_url": originalURL,
			"title":        "Test Link",
		},
		true,
	)
	s.Require().Equal(201, resp.StatusCode)
	result := s.responseMap(resp)
	return getDataField(result)
}
