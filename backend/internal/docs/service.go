package docs

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/internal/flow"
	"github.com/zipdesk/backend/pkg/queue"
	"github.com/zipdesk/backend/pkg/storage"
)

// Service handles docs business logic
type Service struct {
	repo      *Repository
	storage   *storage.Client
	queue     *queue.Client
	eventBus  *flow.EventBus
	generator *Generator
	log       *zap.Logger
}

// NewService creates a new docs service
func NewService(
	repo *Repository,
	storage *storage.Client,
	q *queue.Client,
	eventBus *flow.EventBus,
	log *zap.Logger,
) *Service {
	return &Service{
		repo:      repo,
		storage:   storage,
		queue:     q,
		eventBus:  eventBus,
		generator: NewGenerator(),
		log:       log,
	}
}

// CreateDocument creates a new document
func (s *Service) CreateDocument(
	ctx context.Context,
	workspaceID string,
	userID string,
	input CreateDocInput,
) (*Document, error) {
	slug, err := s.generateSlug(ctx, input.Title)
	if err != nil {
		return nil, fmt.Errorf(
			"docs.Service.CreateDocument: %w", err,
		)
	}

	if input.Type == "" {
		input.Type = DocTypeOther
	}

	if input.Content.Blocks == nil {
		input.Content.Blocks = []ContentBlock{}
	}
	if input.Content.Variables == nil {
		input.Content.Variables = map[string]string{}
	}

	doc := &Document{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Title:       input.Title,
		Slug:        slug,
		Type:        input.Type,
		Status:      DocStatusDraft,
		Content:     input.Content,
		Settings:    input.Settings,
		IsPublished: false,
		CreatedBy:   userID,
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf(
			"docs.Service.CreateDocument: %w", err,
		)
	}

	return doc, nil
}

// GetDocument returns a document
func (s *Service) GetDocument(
	ctx context.Context,
	id string,
	workspaceID string,
) (*Document, error) {
	doc, err := s.repo.GetByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &DocsError{
			Code:    "NOT_FOUND",
			Message: "document not found",
		}
	}
	return doc, nil
}

// UpdateDocument updates a document
func (s *Service) UpdateDocument(
	ctx context.Context,
	id string,
	workspaceID string,
	input UpdateDocInput,
) (*Document, error) {
	doc, err := s.repo.GetByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &DocsError{
			Code:    "NOT_FOUND",
			Message: "document not found",
		}
	}

	if input.Title != "" {
		doc.Title = input.Title
	}
	if input.Content.Blocks != nil {
		doc.Content = input.Content
	}

	doc.Settings = input.Settings

	if err := s.repo.Update(ctx, doc); err != nil {
		return nil, fmt.Errorf(
			"docs.Service.UpdateDocument: %w", err,
		)
	}

	return doc, nil
}

// DeleteDocument removes a document
func (s *Service) DeleteDocument(
	ctx context.Context,
	id string,
	workspaceID string,
) error {
	return s.repo.Delete(ctx, id, workspaceID)
}

// PublishDocument toggles published state
func (s *Service) PublishDocument(
	ctx context.Context,
	id string,
	workspaceID string,
) (*Document, error) {
	doc, err := s.repo.GetByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &DocsError{
			Code:    "NOT_FOUND",
			Message: "document not found",
		}
	}

	doc.IsPublished = !doc.IsPublished
	if doc.IsPublished {
		doc.Status = DocStatusPublished
	} else {
		doc.Status = DocStatusDraft
	}

	if err := s.repo.Update(ctx, doc); err != nil {
		return nil, fmt.Errorf(
			"docs.Service.PublishDocument: %w", err,
		)
	}

	if doc.IsPublished {
		s.eventBus.PublishAsync(flow.DocPublishedEvent{
			BaseEvent: flow.BaseEvent{
				Type:        flow.EventDocPublished,
				WorkspaceID: workspaceID,
				Source:      "docs",
				OccurredAt:  time.Now(),
			},
			DocID:    doc.ID,
			DocTitle: doc.Title,
			DocSlug:  doc.Slug,
		})
	}

	return doc, nil
}

// ListDocuments returns paginated documents
func (s *Service) ListDocuments(
	ctx context.Context,
	workspaceID string,
	params ListParams,
) ([]Document, int64, error) {
	return s.repo.List(ctx, workspaceID, params)
}

// GetPublicDocument returns published doc
func (s *Service) GetPublicDocument(
	ctx context.Context,
	slug string,
) (*Document, error) {
	doc, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, &DocsError{
			Code:    "NOT_FOUND",
			Message: "document not found",
		}
	}
	return doc, nil
}

// generateSlug creates unique slug
func (s *Service) generateSlug(
	ctx context.Context,
	title string,
) (string, error) {
	base := strings.ToLower(title)
	base = strings.ReplaceAll(base, " ", "-")

	var clean strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '-' {
			clean.WriteRune(r)
		}
	}

	baseSlug := clean.String()
	if len(baseSlug) > 40 {
		baseSlug = baseSlug[:40]
	}
	if baseSlug == "" {
		baseSlug = "document"
	}

	exists, _ := s.repo.SlugExists(ctx, baseSlug)
	if !exists {
		return baseSlug, nil
	}

	for i := 0; i < 10; i++ {
		suffix := fmt.Sprintf(
			"%04d", rand.Intn(9999),
		)
		slug := baseSlug + "-" + suffix
		exists, _ := s.repo.SlugExists(ctx, slug)
		if !exists {
			return slug, nil
		}
	}

	return "", fmt.Errorf(
		"docs.Service.generateSlug: exhausted attempts",
	)
}
