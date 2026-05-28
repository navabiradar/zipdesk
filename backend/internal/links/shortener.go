package links

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

const (
	charset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	codeLength = 6
)

var (
	slugRegex = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	rng       = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// GenerateCode creates a random short code
func GenerateCode(length int) string {
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rng.Intn(len(charset))]
	}
	return string(code)
}

// GenerateUniqueCode generates a code not in DB
func GenerateUniqueCode(
	ctx context.Context,
	db *bun.DB,
	length int,
) (string, error) {
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		code := GenerateCode(length)
		exists, err := slugExists(ctx, db, code)
		if err != nil {
			return "", fmt.Errorf(
				"GenerateUniqueCode: check exists: %w", err,
			)
		}
		if !exists {
			return code, nil
		}
	}
	// Try longer code if collisions
	return GenerateUniqueCode(ctx, db, length+1)
}

// ValidateCustomSlug checks if slug is valid
func ValidateCustomSlug(slug string) error {
	if len(slug) < 3 {
		return &ValidationError{
			Field:   "custom_slug",
			Message: "slug must be at least 3 characters",
		}
	}
	if len(slug) > 50 {
		return &ValidationError{
			Field:   "custom_slug",
			Message: "slug must be less than 50 characters",
		}
	}
	if !slugRegex.MatchString(slug) {
		return &ValidationError{
			Field:   "custom_slug",
			Message: "slug can only contain letters, numbers and hyphens",
		}
	}
	// Reserved slugs
	reserved := []string{
		"api", "auth", "admin", "dashboard",
		"health", "static", "assets", "s",
		"f", "d", "bio", "board",
	}
	for _, r := range reserved {
		if strings.ToLower(slug) == r {
			return &ValidationError{
				Field:   "custom_slug",
				Message: "this slug is reserved",
			}
		}
	}
	return nil
}

// NormalizeURL cleans and validates a URL
func NormalizeURL(raw string) (string, error) {
	if raw == "" {
		return "", &ValidationError{
			Field:   "original_url",
			Message: "URL is required",
		}
	}

	// Add scheme if missing
	if !strings.HasPrefix(raw, "http://") &&
		!strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return "", &ValidationError{
			Field:   "original_url",
			Message: "invalid URL format",
		}
	}

	if parsed.Host == "" {
		return "", &ValidationError{
			Field:   "original_url",
			Message: "URL must have a valid host",
		}
	}

	return parsed.String(), nil
}

// slugExists checks if code/slug already exists
func slugExists(
	ctx context.Context,
	db *bun.DB,
	slug string,
) (bool, error) {
	count, err := db.NewSelect().
		TableExpr("links").
		Where("short_code = ? OR custom_slug = ?", slug, slug).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// BuildUTMURL appends UTM params to URL
func BuildUTMURL(
	originalURL string,
	params map[string]any,
) string {
	if len(params) == 0 {
		return originalURL
	}

	parsed, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	q := parsed.Query()
	utmKeys := []string{
		"utm_source", "utm_medium",
		"utm_campaign", "utm_term", "utm_content",
	}
	for _, key := range utmKeys {
		if val, ok := params[key]; ok && val != "" {
			q.Set(key, fmt.Sprintf("%v", val))
		}
	}

	parsed.RawQuery = q.Encode()
	return parsed.String()
}

// ValidationError is a field validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
