package flow

import "time"

// NewBaseEvent creates a base event with generated ID and current timestamp
func NewBaseEvent(eventType EventType, workspaceID, source string) BaseEvent {
	return BaseEvent{
		ID:          GenerateID(),
		Type:        eventType,
		WorkspaceID: workspaceID,
		Source:      source,
		OccurredAt:  Now(),
	}
}

// NewFormSubmittedEvent creates a new form submitted event
func NewFormSubmittedEvent(workspaceID string, formID, formName, responseID string, data map[string]any, email, name string) FormSubmittedEvent {
	return FormSubmittedEvent{
		BaseEvent:  NewBaseEvent(EventFormSubmitted, workspaceID, "forms"),
		FormID:     formID,
		FormName:   formName,
		ResponseID: responseID,
		Data:       data,
		Email:      email,
		Name:       name,
	}
}

// NewFormPublishedEvent creates a new form published event
func NewFormPublishedEvent(workspaceID string, formID, formName, formSlug string) FormPublishedEvent {
	return FormPublishedEvent{
		BaseEvent: NewBaseEvent(EventFormPublished, workspaceID, "forms"),
		FormID:    formID,
		FormName:  formName,
		FormSlug:  formSlug,
	}
}

// NewMailContactAddedEvent creates a new mail contact added event
func NewMailContactAddedEvent(workspaceID, contactID, email, source string) MailContactAddedEvent {
	return MailContactAddedEvent{
		BaseEvent: NewBaseEvent(EventMailContactAdded, workspaceID, "mail"),
		ContactID: contactID,
		Email:     email,
		Source:    source,
	}
}

// NewMailUnsubscribedEvent creates a new mail unsubscribed event
func NewMailUnsubscribedEvent(workspaceID, contactID, email string) MailUnsubscribedEvent {
	return MailUnsubscribedEvent{
		BaseEvent: NewBaseEvent(EventMailUnsubscribed, workspaceID, "mail"),
		ContactID: contactID,
		Email:     email,
	}
}

// NewMailCampaignSentEvent creates a new campaign sent event
func NewMailCampaignSentEvent(workspaceID, campaignID string, recipients int) CampaignSentEvent {
	return CampaignSentEvent{
		BaseEvent:  NewBaseEvent(EventMailCampaignSent, workspaceID, "mail"),
		CampaignID: campaignID,
		Recipients: recipients,
	}
}

// NewMailOpenedEvent creates a new mail opened event
func NewMailOpenedEvent(workspaceID, campaignID, contactID, email string) CampaignOpenedEvent {
	return CampaignOpenedEvent{
		BaseEvent:  NewBaseEvent(EventMailOpened, workspaceID, "mail"),
		CampaignID: campaignID,
		ContactID:  contactID,
		Email:      email,
	}
}

// NewMailClickedEvent creates a new mail clicked event
func NewMailClickedEvent(workspaceID, campaignID, contactID, link string) CampaignClickedEvent {
	return CampaignClickedEvent{
		BaseEvent:  NewBaseEvent(EventMailClicked, workspaceID, "mail"),
		CampaignID: campaignID,
		ContactID:  contactID,
		Link:       link,
	}
}

// NewLinkClickedEvent creates a new link clicked event
func NewLinkClickedEvent(workspaceID, linkID, shortCode, country, device, contactEmail string) LinkClickedEvent {
	return LinkClickedEvent{
		BaseEvent:    NewBaseEvent(EventLinkClicked, workspaceID, "links"),
		LinkID:       linkID,
		ShortCode:    shortCode,
		Country:      country,
		Device:       device,
		ContactEmail: contactEmail,
	}
}

// NewLinkCreatedEvent creates a new link created event
func NewLinkCreatedEvent(workspaceID, linkID, shortCode string) LinkCreatedEvent {
	return LinkCreatedEvent{
		BaseEvent: NewBaseEvent(EventLinkCreated, workspaceID, "links"),
		LinkID:    linkID,
		ShortCode: shortCode,
	}
}

// NewDocViewedEvent creates a new doc viewed event
func NewDocViewedEvent(workspaceID, docID, viewerEmail string) DocViewedEvent {
	return DocViewedEvent{
		BaseEvent:   NewBaseEvent(EventDocViewed, workspaceID, "docs"),
		DocID:       docID,
		ViewerEmail: viewerEmail,
	}
}

// NewDocPublishedEvent creates a new doc published event
func NewDocPublishedEvent(workspaceID, docID, docTitle, docSlug string) DocPublishedEvent {
	return DocPublishedEvent{
		BaseEvent: NewBaseEvent(EventDocPublished, workspaceID, "docs"),
		DocID:     docID,
		DocTitle:  docTitle,
		DocSlug:   docSlug,
	}
}

// NewCRMContactCreatedEvent creates a new CRM contact created event
func NewCRMContactCreatedEvent(workspaceID, contactID, email, source string) CRMContactCreatedEvent {
	return CRMContactCreatedEvent{
		BaseEvent: NewBaseEvent(EventCRMContactCreated, workspaceID, "crm"),
		ContactID: contactID,
		Email:     email,
		Source:    source,
	}
}

// NewCRMDealCreatedEvent creates a new CRM deal created event
func NewCRMDealCreatedEvent(workspaceID, dealID, title string, value int) CRMDealCreatedEvent {
	return CRMDealCreatedEvent{
		BaseEvent: NewBaseEvent(EventCRMDealCreated, workspaceID, "crm"),
		DealID:    dealID,
		Title:     title,
		Value:     value,
	}
}

// NewCRMDealWonEvent creates a new CRM deal won event
func NewCRMDealWonEvent(workspaceID, dealID, title string, value int) CRMDealWonEvent {
	return CRMDealWonEvent{
		BaseEvent: NewBaseEvent(EventCRMDealWon, workspaceID, "crm"),
		DealID:    dealID,
		Title:     title,
		Value:     value,
	}
}

// NewCRMDealLostEvent creates a new CRM deal lost event
func NewCRMDealLostEvent(workspaceID, dealID, title, reason string) CRMDealLostEvent {
	return CRMDealLostEvent{
		BaseEvent: NewBaseEvent(EventCRMDealLost, workspaceID, "crm"),
		DealID:    dealID,
		Title:     title,
		Reason:    reason,
	}
}

// NewHealthCheckEvent creates a new health check event
func NewHealthCheckEvent(workspaceID, service, status string, quota, quotaMax, latencyMs int64) HealthCheckEvent {
	return HealthCheckEvent{
		BaseEvent: NewBaseEvent(EventHealthCheck, workspaceID, "health"),
		Service:   service,
		Status:    status,
		Quota:     quota,
		QuotaMax:  quotaMax,
		LatencyMs: latencyMs,
	}
}

// NewQuotaWarningEvent creates a new quota warning event
func NewQuotaWarningEvent(workspaceID, service string, quota, quotaMax int64) QuotaWarningEvent {
	var pct float64
	if quotaMax > 0 {
		pct = float64(quota) / float64(quotaMax) * 100
	}
	return QuotaWarningEvent{
		BaseEvent: NewBaseEvent(EventQuotaWarning, workspaceID, "system"),
		Service:   service,
		Quota:     quota,
		QuotaMax:  quotaMax,
		QuotaPct:  pct,
	}
}

// EventEnvelope wraps any event for transport
type EventEnvelope struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	SentAt  time.Time   `json:"sent_at"`
}

// WrapEvent wraps an event in an envelope
func WrapEvent(eventType string, payload interface{}) EventEnvelope {
	return EventEnvelope{
		Type:    eventType,
		Payload: payload,
		SentAt:  time.Now().UTC(),
	}
}
