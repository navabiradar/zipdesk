package forms

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/zipdesk/backend/internal/flow"
)

type Service struct {
	repo     *Repository
	queue    interface{}
	eventBus *flow.EventBus
	storage  interface{}
	logger   *zap.Logger
}

func NewService(repo *Repository, queue interface{}, eventBus *flow.EventBus, storage interface{}, logger *zap.Logger) *Service {
	return &Service{repo: repo, queue: queue, eventBus: eventBus, storage: storage, logger: logger}
}

func (s *Service) CreateForm(ctx context.Context, workspaceID, userID string, input CreateFormInput) (*Form, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, &FormError{
			Code:    "VALIDATION_ERROR",
			Message: "form title is required",
			Field:   "title",
		}
	}

	slug := generateSlug(title)
	if slug == "" {
		return nil, &FormError{
			Code:    "VALIDATION_ERROR",
			Message: "form title cannot be converted to a slug",
			Field:   "title",
		}
	}

	exists, err := s.repo.SlugExists(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("forms.Service.CreateForm: %w", err)
	}
	if exists {
		return nil, &FormError{
			Code:    "SLUG_TAKEN",
			Message: "form slug is already in use",
			Field:   "title",
		}
	}

	for idx := range input.Fields {
		input.Fields[idx].FieldOrder = idx + 1
	}

	form := &Form{
		WorkspaceID: workspaceID,
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		Slug:        slug,
		Settings:    input.Settings,
		CreatedBy:   userID,
		Fields:      input.Fields,
	}

	if err := s.repo.CreateForm(ctx, form); err != nil {
		return nil, fmt.Errorf("forms.Service.CreateForm: %w", err)
	}

	return form, nil
}

func (s *Service) UpdateForm(ctx context.Context, id, workspaceID string, input UpdateFormInput) (*Form, error) {
	form, err := s.repo.GetFormByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &FormError{Code: "NOT_FOUND", Message: "form not found"}
	}

	if strings.TrimSpace(input.Title) != "" {
		form.Title = strings.TrimSpace(input.Title)
	}
	if input.Description != "" {
		form.Description = strings.TrimSpace(input.Description)
	}
	if input.Settings != (FormSettings{}) {
		form.Settings = input.Settings
	}
	if input.Fields != nil {
		for idx := range input.Fields {
			input.Fields[idx].FormID = form.ID
			input.Fields[idx].FieldOrder = idx + 1
		}
		form.Fields = input.Fields
	}

	if err := s.repo.UpdateForm(ctx, form); err != nil {
		return nil, fmt.Errorf("forms.Service.UpdateForm: %w", err)
	}

	return form, nil
}

func (s *Service) DeleteForm(ctx context.Context, id, workspaceID string) error {
	if err := s.repo.DeleteForm(ctx, id, workspaceID); err != nil {
		return fmt.Errorf("forms.Service.DeleteForm: %w", err)
	}
	return nil
}

func (s *Service) GetForm(ctx context.Context, id, workspaceID string) (*Form, error) {
	form, err := s.repo.GetFormByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &FormError{Code: "NOT_FOUND", Message: "form not found"}
	}
	return form, nil
}

func (s *Service) ListForms(ctx context.Context, workspaceID string, params ListParams) (*ListResponse, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PerPage <= 0 {
		params.PerPage = 20
	}

	items, total, err := s.repo.ListForms(ctx, workspaceID, params)
	if err != nil {
		return nil, fmt.Errorf("forms.Service.ListForms: %w", err)
	}

	return &ListResponse{Items: items, Total: total, Page: params.Page, PerPage: params.PerPage}, nil
}

func (s *Service) SubmitResponse(ctx context.Context, id, workspaceID string, input SubmitInput, meta ResponseMeta) (*FormResponse, error) {
	form, err := s.repo.GetFormByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &FormError{Code: "NOT_FOUND", Message: "form not found"}
	}
	return s.createResponse(ctx, form, workspaceID, input, meta)
}

func (s *Service) SubmitPublicResponse(ctx context.Context, slug string, input SubmitInput, meta ResponseMeta) (*FormResponse, error) {
	form, err := s.repo.GetFormBySlug(ctx, slug)
	if err != nil {
		return nil, &FormError{Code: "NOT_FOUND", Message: "form not found"}
	}
	if !form.IsPublished {
		return nil, &FormError{Code: "FORM_NOT_PUBLISHED", Message: "form is not published"}
	}
	if form.Settings.CloseDate != nil {
		closeDate, parseErr := time.Parse("2006-01-02", *form.Settings.CloseDate)
		if parseErr == nil && time.Now().After(closeDate) {
			return nil, &FormError{Code: "FORM_CLOSED", Message: "form has been closed"}
		}
	}
	if form.Settings.ResponseLimit != nil {
		totalResponses, countErr := s.repo.GetResponseCount(ctx, form.ID)
		if countErr != nil {
			return nil, fmt.Errorf("forms.Service.SubmitPublicResponse: %w", countErr)
		}
		if totalResponses >= int64(*form.Settings.ResponseLimit) {
			return nil, &FormError{Code: "RESPONSE_LIMIT_REACHED", Message: "response limit reached"}
		}
	}
	return s.createResponse(ctx, form, form.WorkspaceID, input, meta)
}

func (s *Service) createResponse(ctx context.Context, form *Form, workspaceID string, input SubmitInput, meta ResponseMeta) (*FormResponse, error) {
	response := &FormResponse{
		FormID:         form.ID,
		WorkspaceID:    workspaceID,
		Data:           input.Data,
		CompletionTime: input.CompletionTime,
		IsComplete:     true,
		IPAddress:      meta.IP,
		UserAgent:      meta.UserAgent,
		Referrer:       meta.Referrer,
	}

	if err := s.repo.CreateResponse(ctx, response); err != nil {
		return nil, fmt.Errorf("forms.Service.createResponse: %w", err)
	}

	if s.eventBus != nil {
		email := ExtractEmail(form.Fields, input.Data)
		name := ExtractName(form.Fields, input.Data)
		s.eventBus.PublishAsync(flow.FormSubmittedEvent{
			BaseEvent: flow.BaseEvent{
				Type:        flow.EventFormSubmitted,
				WorkspaceID: workspaceID,
				Source:      "forms",
				OccurredAt:  time.Now(),
			},
			FormID:     form.ID,
			FormName:   form.Title,
			ResponseID: response.ID,
			Data:       input.Data,
			Email:      email,
			Name:       name,
		})
	}

	return response, nil
}

func (s *Service) GetResponses(ctx context.Context, id, workspaceID string, params ListParams) (*ResponseListResponse, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PerPage <= 0 {
		params.PerPage = 20
	}

	items, total, err := s.repo.ListResponses(ctx, id, workspaceID, params)
	if err != nil {
		return nil, fmt.Errorf("forms.Service.GetResponses: %w", err)
	}

	return &ResponseListResponse{Items: items, Total: total, Page: params.Page, PerPage: params.PerPage}, nil
}

func (s *Service) GetAnalytics(ctx context.Context, id, workspaceID string) (*FormAnalytics, error) {
	if _, err := s.repo.GetFormByID(ctx, id, workspaceID); err != nil {
		return nil, &FormError{Code: "NOT_FOUND", Message: "form not found"}
	}

	totalViews, err := s.repo.GetViewCount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("forms.Service.GetAnalytics: %w", err)
	}

	totalResponses, err := s.repo.GetResponseCount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("forms.Service.GetAnalytics: %w", err)
	}

	completionRate := 0.0
	if totalViews > 0 {
		completionRate = float64(totalResponses) / float64(totalViews)
	}

	return &FormAnalytics{
		TotalViews:     totalViews,
		TotalResponses: totalResponses,
		CompletionRate: completionRate,
		AverageTime:    0,
		ResponsesByDay: []DayCount{},
		FieldDropoff:   []FieldDropoff{},
	}, nil
}

func (s *Service) GetPublicForm(ctx context.Context, slug string) (*FormPublicView, error) {
	form, err := s.repo.GetFormBySlug(ctx, slug)
	if err != nil {
		return nil, &FormError{Code: "NOT_FOUND", Message: "form not found"}
	}
	if !form.IsPublished {
		return nil, &FormError{Code: "NOT_FOUND", Message: "form not published"}
	}

	visible := NewLogicEngine().GetVisibleFields(form.Fields, nil)

	view := &FormPublicView{
		ID:          form.ID,
		Title:       form.Title,
		Description: form.Description,
		Slug:        form.Slug,
		Settings:    form.Settings,
		Fields:      visible,
	}

	if form.Settings.Password != "" {
		view.Fields = nil
	}

	return view, nil
}

func (s *Service) RecordView(ctx context.Context, slug string, meta ResponseMeta) error {
	form, err := s.repo.GetFormBySlug(ctx, slug)
	if err != nil {
		return &FormError{Code: "NOT_FOUND", Message: "form not found"}
	}

	view := &FormView{
		FormID:    form.ID,
		IPAddress: meta.IP,
		Device:    meta.UserAgent,
		Referrer:  meta.Referrer,
	}

	return s.repo.RecordView(ctx, view)
}

func (s *Service) PublishForm(ctx context.Context, id, workspaceID string) (*Form, error) {
	form, err := s.repo.GetFormByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &FormError{Code: "NOT_FOUND", Message: "form not found"}
	}

	now := time.Now()
	wasPublished := form.IsPublished
	form.IsPublished = !form.IsPublished
	if form.IsPublished {
		form.PublishedAt = &now
	} else {
		form.PublishedAt = nil
	}

	if err := s.repo.UpdateForm(ctx, form); err != nil {
		return nil, fmt.Errorf("forms.Service.PublishForm: %w", err)
	}

	if s.eventBus != nil && form.IsPublished && !wasPublished {
		s.eventBus.PublishAsync(flow.FormSubmittedEvent{
			BaseEvent: flow.BaseEvent{
				Type:        flow.EventFormPublished,
				WorkspaceID: workspaceID,
				Source:      "forms",
				OccurredAt:  now,
			},
			FormID:   form.ID,
			FormName: form.Title,
		})
	}

	return form, nil
}

func (s *Service) ExportCSV(ctx context.Context, id, workspaceID string) ([]byte, error) {
	form, err := s.repo.GetFormByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &FormError{Code: "NOT_FOUND", Message: "form not found"}
	}

	responses, err := s.repo.ExportResponses(ctx, id, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("forms.Service.ExportCSV: %w", err)
	}

	var buf strings.Builder
	buf.WriteString("ID,Submitted At")
	for _, f := range form.Fields {
		buf.WriteString(",")
		buf.WriteString(csvEscape(f.Label))
	}
	buf.WriteString("\n")

	for _, r := range responses {
		buf.WriteString(r.ID)
		buf.WriteString(",")
		buf.WriteString(r.SubmittedAt.Format("2006-01-02 15:04:05"))
		for _, f := range form.Fields {
			buf.WriteString(",")
			if val, ok := r.Data[f.ID]; ok {
				buf.WriteString(csvEscape(fmt.Sprintf("%v", val)))
			}
		}
		buf.WriteString("\n")
	}

	return []byte(buf.String()), nil
}

func csvEscape(val string) string {
	if strings.ContainsAny(val, "\",\n\r") {
		val = strings.ReplaceAll(val, "\"", "\"\"")
		return "\"" + val + "\""
	}
	return val
}

func generateSlug(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	if slug == "" {
		return ""
	}

	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	slug = strings.Trim(slug, "-")
	return slug
}
