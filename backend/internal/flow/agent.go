package flow

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

// AnthropicRequest is the Claude API request
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []AnthropicMessage `json:"messages"`
	Tools     []AnthropicTool    `json:"tools,omitempty"`
	Stream    bool               `json:"stream"`
}

// AnthropicMessage is a chat message
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicTool defines a callable tool
type AnthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// StreamEvent is an SSE event from Claude
type StreamEvent struct {
	Type  string `json:"type"`
	Delta *Delta `json:"delta,omitempty"`
}

// Delta holds streaming text
type Delta struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// AgentService handles AI conversations
type AgentService struct {
	svc    *Service
	client *http.Client
	logger *zap.Logger
}

// streamChat streams AI response to client
func (s *Service) streamChat(
	ctx context.Context,
	w *bufio.Writer,
	workspaceID string,
	userID string,
	input ChatInput,
) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		writeSSE(w, "error", "AI not configured")
		return
	}

	messages := buildMessages(input)
	systemPrompt := buildSystemPrompt(workspaceID)

	req := AnthropicRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages:  messages,
		Tools:     getZipDeskTools(),
		Stream:    true,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		writeSSE(w, "error", "failed to build request")
		return
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader(payload),
	)
	if err != nil {
		writeSSE(w, "error", err.Error())
		return
	}

	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}

	resp, err := client.Do(httpReq)
	if err != nil {
		writeSSE(w, "error", err.Error())
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		if len(line) > 6 && line[:6] == "data: " {
			data := line[6:]
			if data == "[DONE]" {
				break
			}

			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if event.Type == "content_block_delta" &&
				event.Delta != nil &&
				event.Delta.Type == "text_delta" {
				writeSSE(w, "text", event.Delta.Text)
				w.Flush()
			}
		}
	}

	writeSSE(w, "done", "")
	w.Flush()

	go s.saveConversation(context.Background(), workspaceID, userID, input)
}

// buildMessages builds Claude message array
func buildMessages(input ChatInput) []AnthropicMessage {
	var messages []AnthropicMessage

	for _, msg := range input.History {
		messages = append(messages, AnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	messages = append(messages, AnthropicMessage{
		Role:    "user",
		Content: input.Message,
	})

	return messages
}

// buildSystemPrompt creates context-aware prompt
func buildSystemPrompt(workspaceID string) string {
	return fmt.Sprintf(`You are ZipDesk AI, an intelligent business assistant.

You help users manage their business tools through natural conversation.

WORKSPACE: %s

YOUR CAPABILITIES:
You can help users:
- Create and manage short links (ZipDesk Links)
- Build and publish forms (ZipDesk Forms)
- Create documents and PDFs (ZipDesk Docs)
- Manage email contacts and campaigns (ZipDesk Mail)
- Manage CRM contacts and deals (ZipDesk CRM)
- Set up automation flows (ZipDesk Flow)
- View analytics and health status

STYLE:
- Be concise and action-oriented
- Confirm what you did after each action
- Suggest logical next steps
- Use plain language, not technical jargon

When a user asks you to create something,
do it immediately with sensible defaults.
Do not ask unnecessary clarifying questions.
Make smart assumptions from context.`,
		workspaceID,
	)
}

// getZipDeskTools returns Claude tool definitions
func getZipDeskTools() []AnthropicTool {
	return []AnthropicTool{
		{
			Name:        "create_form",
			Description: "Create a new ZipDesk form with fields",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string", "description": "Form title"},
					"description": map[string]any{"type": "string", "description": "Form description"},
					"fields": map[string]any{
						"type":        "array",
						"description": "Form fields",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"type": map[string]any{
									"type": "string",
									"enum": []string{"text", "email", "phone", "number", "dropdown", "mcq", "rating", "long_text"},
								},
								"label":    map[string]any{"type": "string"},
								"required": map[string]any{"type": "boolean"},
							},
						},
					},
				},
				"required": []string{"title"},
			},
		},
		{
			Name:        "create_link",
			Description: "Create a short trackable link",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url":   map[string]any{"type": "string", "description": "URL to shorten"},
					"title": map[string]any{"type": "string", "description": "Link title"},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "create_flow",
			Description: "Create an automation flow connecting tools",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":    map[string]any{"type": "string", "description": "Flow name"},
					"trigger": map[string]any{"type": "string", "description": "Trigger event type", "enum": []string{"form.submitted", "link.clicked", "mail.contact_added"}},
					"action":  map[string]any{"type": "string", "description": "Action to take", "enum": []string{"mail.add_contact", "mail.send_email", "system.notify", "system.webhook"}},
				},
				"required": []string{"name", "trigger", "action"},
			},
		},
		{
			Name:        "get_analytics",
			Description: "Get analytics and statistics",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"resource": map[string]any{"type": "string", "enum": []string{"forms", "links", "mail", "health"}},
				},
				"required": []string{"resource"},
			},
		},
		{
			Name:        "get_health",
			Description: "Check system health and quota status",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

// saveConversation persists chat to DB
func (s *Service) saveConversation(
	ctx context.Context,
	workspaceID string,
	userID string,
	input ChatInput,
) {
	convID := input.ConversationID
	if convID == "" {
		conv := &AIConversation{
			WorkspaceID: workspaceID,
			UserID:      userID,
			Title:       truncate(input.Message, 50),
		}
		if err := s.repo.CreateConversation(ctx, conv); err != nil {
			s.logger.Warn("failed to save conversation", zap.Error(err))
			return
		}
		convID = conv.ID
	}

	msg := &AIMessage{
		ConversationID: convID,
		Role:           "user",
		Content:        input.Message,
		ToolCalls:      []ToolCall{},
		ToolResults:    []ToolResult{},
	}

	_ = s.repo.SaveMessage(ctx, msg)
}

// writeSSE writes an SSE event to writer
func writeSSE(w *bufio.Writer, event string, data string) {
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// truncate shortens a string
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

var _ = io.EOF
