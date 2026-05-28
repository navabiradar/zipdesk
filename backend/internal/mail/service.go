package mail

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/internal/flow"
	"github.com/zipdesk/backend/pkg/queue"
)

// Service handles mail business logic
type Service struct {
	repo     *Repository
	queue    *queue.Client
	eventBus *flow.EventBus
	sender   *Sender
	log      *zap.Logger
}

// NewService creates a new mail service
func NewService(
	repo *Repository,
	q *queue.Client,
	eventBus *flow.EventBus,
	log *zap.Logger,
) *Service {
	return &Service{
		repo:     repo,
		queue:    q,
		eventBus: eventBus,
		sender:   NewSender(log),
		log:      log,
	}
}

// UpsertContact creates or updates a contact
// Called by Flow system triggers
func (s *Service) UpsertContact(
	ctx context.Context,
	workspaceID string,
	email string,
	data map[string]any,
) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", fmt.Errorf("mail.Service.UpsertContact: email required")
	}

	firstName, _ := data["name"].(string)
	source, _ := data["source"].(string)
	if source == "" {
		source = "api"
	}

	tags := []string{}
	if source == "form" {
		tags = append(tags, "form-lead")
	}
	if rawTags, ok := data["tags"]; ok {
		if tagSlice, ok := rawTags.([]string); ok {
			tags = append(tags, tagSlice...)
		}
	}

	contact := &Contact{
		ID:           uuid.New().String(),
		WorkspaceID:  workspaceID,
		Email:        email,
		FirstName:    firstName,
		Source:       source,
		Status:       ContactStatusSubscribed,
		Tags:         tags,
		CustomFields: map[string]any{},
		SubscribedAt: time.Now(),
	}
	if err := s.repo.UpsertContact(ctx, contact); err != nil {
		s.log.Warn("UpsertContact: repo error", zap.Error(err))
		return "", fmt.Errorf("mail.Service.UpsertContact: %w", err)
	}

	s.log.Debug("contact upserted",
		zap.String("email", email),
		zap.String("workspace_id", workspaceID),
		zap.String("source", source),
	)

	s.eventBus.PublishAsync(flow.MailContactAddedEvent{
		BaseEvent: flow.BaseEvent{
			Type:        flow.EventMailContactAdded,
			WorkspaceID: workspaceID,
			Source:      "mail",
			OccurredAt:  time.Now(),
		},
		Email:  email,
		Source: source,
	})

	return contact.ID, nil
}

// UnsubscribeContact marks contact as unsubscribed
func (s *Service) UnsubscribeContact(
	ctx context.Context,
	workspaceID string,
	email string,
) error {
	return s.repo.UnsubscribeContact(ctx, workspaceID, strings.ToLower(email))
}

// CreateContact creates a new contact
func (s *Service) CreateContact(
	ctx context.Context,
	workspaceID string,
	input CreateContactInput,
) (*Contact, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	existing, _ := s.repo.GetContactByEmail(ctx, workspaceID, email)
	if existing != nil {
		return nil, &MailError{
			Code:    "DUPLICATE_CONTACT",
			Message: "contact with this email already exists",
		}
	}

	if input.Tags == nil {
		input.Tags = []string{}
	}
	if input.CustomFields == nil {
		input.CustomFields = map[string]any{}
	}
	if input.Source == "" {
		input.Source = "manual"
	}

	contact := &Contact{
		ID:           uuid.New().String(),
		WorkspaceID:  workspaceID,
		Email:        email,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Company:      input.Company,
		Phone:        input.Phone,
		Tags:         input.Tags,
		CustomFields: input.CustomFields,
		Source:       input.Source,
		Status:       ContactStatusSubscribed,
	}

	if err := s.repo.CreateContact(ctx, contact); err != nil {
		return nil, fmt.Errorf("mail.Service.CreateContact: %w", err)
	}

	return contact, nil
}

// ListContacts returns paginated contacts
func (s *Service) ListContacts(ctx context.Context, workspaceID string, params ListParams) ([]Contact, int64, error) {
	return s.repo.ListContacts(ctx, workspaceID, params)
}

// GetContact retrieves a contact by ID
func (s *Service) GetContact(ctx context.Context, id string) (*Contact, error) {
	return s.repo.GetContact(ctx, id)
}

// DeleteContact removes a contact
func (s *Service) DeleteContact(ctx context.Context, id string, workspaceID string) error {
	return s.repo.DeleteContact(ctx, id, workspaceID)
}

// ==================== LISTS ====================

// CreateList creates a new mail list
func (s *Service) CreateList(ctx context.Context, workspaceID string, name, description string) (*MailList, error) {
	l := &MailList{
		WorkspaceID: workspaceID,
		Name:        name,
		Description: description,
	}
	if err := s.repo.CreateList(ctx, l); err != nil {
		return nil, fmt.Errorf("mail.Service.CreateList: %w", err)
	}
	return l, nil
}

// GetList retrieves a list by ID
func (s *Service) GetList(ctx context.Context, id string) (*MailList, error) {
	return s.repo.GetList(ctx, id)
}

// ListLists returns all lists for a workspace
func (s *Service) ListLists(ctx context.Context, workspaceID string) ([]MailList, error) {
	return s.repo.ListLists(ctx, workspaceID)
}

// DeleteList deletes a list
func (s *Service) DeleteList(ctx context.Context, id string) error {
	return s.repo.DeleteList(ctx, id)
}

// ==================== CAMPAIGNS ====================

// CreateCampaign creates a new campaign
func (s *Service) CreateCampaign(ctx context.Context, workspaceID string, input CreateCampaignInput) (*Campaign, error) {
	now := time.Now()
	campaign := &Campaign{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Name:        input.Name,
		Subject:     input.Subject,
		PreviewText: input.PreviewText,
		FromName:    input.FromName,
		FromEmail:   input.FromEmail,
		Content:     input.Content,
		ListID:      input.ListID,
		Status:      CampaignStatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateCampaign(ctx, campaign); err != nil {
		return nil, fmt.Errorf("mail.Service.CreateCampaign: %w", err)
	}

	stats := &CampaignStats{CampaignID: campaign.ID}
	_ = s.repo.CreateCampaignStats(ctx, stats)

	return campaign, nil
}

// ListCampaigns returns paginated campaigns
func (s *Service) ListCampaigns(ctx context.Context, workspaceID string, params ListParams) ([]Campaign, int64, error) {
	return s.repo.ListCampaigns(ctx, workspaceID, params)
}

// GetCampaign retrieves a campaign by ID
func (s *Service) GetCampaign(ctx context.Context, id string) (*Campaign, error) {
	return s.repo.GetCampaign(ctx, id)
}

// DeleteCampaign deletes a campaign
func (s *Service) DeleteCampaign(ctx context.Context, id string) error {
	return s.repo.DeleteCampaign(ctx, id)
}

// GetCampaignStats returns campaign metrics
func (s *Service) GetCampaignStats(ctx context.Context, id string, workspaceID string) (*CampaignStats, error) {
	_, err := s.repo.GetCampaignByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &MailError{
			Code:    "NOT_FOUND",
			Message: "campaign not found",
		}
	}
	return s.repo.GetCampaignStats(ctx, id)
}
