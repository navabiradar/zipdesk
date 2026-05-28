package crm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/internal/flow"
)

// Service handles CRM business logic
type Service struct {
	repo     *Repository
	eventBus *flow.EventBus
	log      *zap.Logger
}

// NewService creates a new CRM service
func NewService(
	repo *Repository,
	eventBus *flow.EventBus,
	log *zap.Logger,
) *Service {
	return &Service{
		repo:     repo,
		eventBus: eventBus,
		log:      log,
	}
}

// CreateContactFromEvent creates contact from event
// Called by Flow system triggers
func (s *Service) CreateContactFromEvent(
	ctx context.Context,
	workspaceID string,
	email string,
	name string,
	source string,
) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", nil
	}

	existing, _ := s.repo.GetContactByEmail(ctx, workspaceID, email)
	if existing != nil {
		return existing.ID, nil
	}

	firstName := name
	lastName := ""
	parts := strings.SplitN(name, " ", 2)
	if len(parts) == 2 {
		firstName = parts[0]
		lastName = parts[1]
	}

	contact := &CRMContact{
		WorkspaceID:  workspaceID,
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		LeadSource:   source,
		LeadStatus:   "new",
		LeadScore:    10,
		Tags:         []string{},
		CustomFields: map[string]any{},
	}

	contact.ID = uuid.New().String()

	if err := s.repo.CreateContact(ctx, contact); err != nil {
		return "", fmt.Errorf("crm.Service.CreateContactFromEvent: %w", err)
	}

	s.log.Info("CRM contact created from event",
		zap.String("email", email),
		zap.String("source", source),
	)

	s.eventBus.PublishAsync(flow.CRMContactCreatedEvent{
		BaseEvent: flow.BaseEvent{
			Type:        flow.EventCRMContactCreated,
			WorkspaceID: workspaceID,
			Source:      "crm",
			OccurredAt:  time.Now(),
		},
		ContactID: contact.ID,
		Email:     email,
		Source:    source,
	})

	return contact.ID, nil
}

// CreateContact creates a CRM contact manually
func (s *Service) CreateContact(
	ctx context.Context,
	workspaceID string,
	input CreateContactInput,
) (*CRMContact, error) {
	contact := &CRMContact{
		WorkspaceID:  workspaceID,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Email:        strings.ToLower(input.Email),
		Phone:        input.Phone,
		JobTitle:     input.JobTitle,
		LeadSource:   input.LeadSource,
		LeadStatus:   "new",
		Tags:         input.Tags,
		CustomFields: map[string]any{},
	}

	if contact.Tags == nil {
		contact.Tags = []string{}
	}

	if err := s.repo.CreateContact(ctx, contact); err != nil {
		return nil, fmt.Errorf("crm.Service.CreateContact: %w", err)
	}

	return contact, nil
}

// ListContacts returns paginated contacts
func (s *Service) ListContacts(ctx context.Context, workspaceID string, params ListParams) ([]CRMContact, int64, error) {
	return s.repo.ListContacts(ctx, workspaceID, params)
}

// ListDeals returns paginated deals
func (s *Service) ListDeals(ctx context.Context, workspaceID string, params ListParams) ([]CRMDeal, int64, error) {
	return s.repo.ListDeals(ctx, workspaceID, params)
}
