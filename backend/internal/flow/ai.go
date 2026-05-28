package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// AIService provides AI chat integration
type AIService struct {
	apiKey       string
	modelURI     string
	headers      map[string]string
	maxRetries   int
	retryBackoff time.Duration
	logger       *zap.Logger
}

// NewAIService creates a new AI service
func NewAIService(apiKey, modelURI string, logger *zap.Logger) *AIService {
	if modelURI == "" {
		modelURI = "https://api.anthropic.com/v1/messages"
	}
	return &AIService{
		apiKey:       apiKey,
		modelURI:     modelURI,
		headers:      make(map[string]string),
		maxRetries:   3,
		retryBackoff: 500 * time.Millisecond,
		logger:       logger,
	}
}

// Chat sends a chat message and returns the response
func (s *AIService) Chat(ctx context.Context, input ChatInput) (string, error) {
	s.logger.Info("sending chat message", zap.Int("history_len", len(input.History)))

	messages := []map[string]interface{}{}
	for _, msg := range input.History {
		messages = append(messages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": input.Message,
	})

	body := map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"messages":   messages,
		"max_tokens": 4096,
	}

	response, err := s.postWithRetry(ctx, s.modelURI, body)
	if err != nil {
		s.logger.Error("chat request failed", zap.Error(err))
		return "", fmt.Errorf("ai.Chat: %w", err)
	}

	responseContent := ""
	if data, ok := response.(map[string]interface{}); ok {
		if contentArr, ok := data["content"].([]interface{}); ok && len(contentArr) > 0 {
			if first, ok := contentArr[0].(map[string]interface{}); ok {
				if text, ok := first["text"].(string); ok {
					responseContent = text
				}
			}
		}
	}

	if responseContent == "" {
		return "", fmt.Errorf("ai.Chat: empty response from AI")
	}

	s.logger.Debug("chat response received", zap.Int("len", len(responseContent)))
	return responseContent, nil
}

// postWithRetry sends a POST request with retries
func (s *AIService) postWithRetry(ctx context.Context, url string, body interface{}) (interface{}, error) {
	httpClient := getClient()
	defer putClient(httpClient)

	var lastErr error
	for i := 0; i < s.maxRetries; i++ {
		if i > 0 {
			time.Sleep(s.retryBackoff * time.Duration(i))
			s.logger.Warn("retrying AI request", zap.Int("attempt", i+1))
		}

		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", s.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		for k, v := range s.headers {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				return result, nil
			}
			return result, nil
		}

		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("postWithRetry: max retries exceeded: %w", lastErr)
}

var clientPool = make(chan *http.Client, 10)

func getClient() *http.Client {
	select {
	case client := <-clientPool:
		return client
	default:
		return &http.Client{Timeout: 30 * time.Second}
	}
}

func putClient(client *http.Client) {
	select {
	case clientPool <- client:
	default:
		// pool is full, discard
	}
}

func init() {
	for i := 0; i < 10; i++ {
		clientPool <- &http.Client{Timeout: 30 * time.Second}
	}
}

// MessageRecord tracks an AI conversation in the database
func (s *AIService) MessageRecord(ctx context.Context, storage *Storage, userID, workspaceID, message string, conversationID string) (*AIMessage, *AIConversation, error) {
	var conv *AIConversation
	var err error

	if conversationID != "" {
		conv, err = storage.GetConversation(ctx, conversationID)
		if err != nil {
			s.logger.Warn("conversation not found, creating new", zap.String("id", conversationID))
			conv = nil
		}
	}

	if conv == nil {
		conv = &AIConversation{
			WorkspaceID: workspaceID,
			UserID:      userID,
			Title:       "New conversation",
		}
		if err := storage.CreateConversation(ctx, conv); err != nil {
			s.logger.Error("failed to create conversation", zap.Error(err))
			return nil, nil, fmt.Errorf("ai.MessageRecord: create conversation: %w", err)
		}
	}

	userMsg := &AIMessage{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        message,
	}
	if err := storage.CreateMessage(ctx, userMsg); err != nil {
		s.logger.Error("failed to create user message", zap.Error(err))
		return nil, nil, fmt.Errorf("ai.MessageRecord: create user message: %w", err)
	}

	conv.MessageCount++
	_ = storage.UpdateConversation(ctx, conv)

	return userMsg, conv, nil
}

// RecordResponse stores an AI response in the database
func (s *AIService) RecordResponse(ctx context.Context, storage *Storage, conversationID, response string, tokensUsed int) (*AIMessage, error) {
	msg := &AIMessage{
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        response,
		TokensUsed:     tokensUsed,
	}
	if err := storage.CreateMessage(ctx, msg); err != nil {
		s.logger.Error("failed to create assistant message", zap.Error(err))
		return nil, fmt.Errorf("ai.RecordResponse: %w", err)
	}
	return msg, nil
}
