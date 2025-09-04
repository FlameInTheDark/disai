package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	"resty.dev/v3"

	"log/slog"

	"github.com/FlameInTheDark/disai/internal/mcp"
)

const (
	// Timeout for all HTTP requests to the LLM server.
	requestTimeout = 2 * time.Minute
	// Maximum number of consecutive tool calls allowed in a single chat turn.
	maxToolCalls = 10
)

// ModelInfoRequest/Response and ModelDetails are unchanged.
type ModelInfoRequest struct {
	Model string `json:"model"`
}

type ModelInfoResponse struct {
	Details ModelDetails `json:"details"`
}

type ModelDetails struct {
	Family            string `json:"family"`
	Format            string `json:"format"`
	ParameterSize     string `json:"parameter_size"`
	QuantizationLevel string `json:"quantization_level"`
}

// OllamaServer represents a named Ollama server
type OllamaServer struct {
	Name string
	URL  string
}

// Model represents a chat client that can discover a server, send templated
// messages, and automatically invoke tools returned by the LLM.
type Model struct {
	Name      string
	Servers   []OllamaServer
	systemTpl *template.Template
	userTpl   *template.Template
	ToolNames map[string]string

	mcp    *mcp.Client
	client *resty.Client // reusable HTTP client

	// serverLocks holds a channel‑based semaphore for each server URL.
	// The channel is buffered to 1; a send blocks until the server is free.
	serverLocks map[string]chan struct{}
}

// NewModel creates a new Model instance and prepares the HTTP client.
func NewModel(modelName string, servers map[string]string, mcpc *mcp.Client, system, user string, toolNames map[string]string) *Model {
	// Convert map to slice of OllamaServer
	var ollamaServers []OllamaServer
	for name, url := range servers {
		ollamaServers = append(ollamaServers, OllamaServer{
			Name: name,
			URL:  url,
		})
	}

	m := &Model{
		Name:        modelName,
		Servers:     ollamaServers,
		mcp:         mcpc,
		serverLocks: make(map[string]chan struct{}),
		ToolNames:   toolNames,
	}
	m.LoadTemplate(system, user)

	// Initialise a single Resty client with a timeout.
	m.client = resty.New().
		SetTimeout(requestTimeout).
		SetRetryCount(0) // no automatic retries – we handle errors explicitly

	// Create a semaphore channel for each server URL.
	for _, srv := range ollamaServers {
		ch := make(chan struct{}, 1)
		ch <- struct{}{}
		m.serverLocks[srv.URL] = ch
	}

	return m
}

// SelectServer queries each configured server until one responds with a
// successful /api/show request. It blocks until a server becomes available.
// Returns the selected server (name and URL).
func (m *Model) SelectServer(ctx context.Context) (OllamaServer, error) {
	for {
		// Try to acquire a lock on any server.
		for _, server := range m.Servers {
			select {
			case <-m.serverLocks[server.URL]:
				// Acquired lock – now check if the server is alive.
				res, err := m.client.R().
					SetContext(ctx).
					SetBody(ModelInfoRequest{Model: m.Name}).
					Post(server.URL + "/api/show")
				if err != nil || res.StatusCode() != 200 {
					// Release the lock and try the next server.
					m.serverLocks[server.URL] <- struct{}{}
					continue
				}
				// Server is alive and locked – return it.
				return server, nil
			default:
				// Server currently busy; skip to next.
			}
		}
		// No free server found – wait a bit before retrying.
		select {
		case <-ctx.Done():
			return OllamaServer{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// StatusCallback is called to report the current status of the Chat operation.
type StatusCallback func(status string)

// Chat sends a templated user message to the LLM, handles tool calls, and
// returns the final assistant response.
func (m *Model) Chat(ctx context.Context, message string, args map[string]any) (string, error) {
	return m.ChatWithStatus(ctx, message, args, nil)
}

// ChatWithStatus sends a templated user message to the LLM with status reporting.
func (m *Model) ChatWithStatus(ctx context.Context, message string, args map[string]any, statusCallback StatusCallback) (string, error) {
	if m.mcp == nil {
		return "", errors.New("mcp client not initialized")
	}

	req, err := m.prepareInitialRequest(message, args, statusCallback)
	if err != nil {
		return "", err
	}

	if statusCallback != nil {
		statusCallback("🔍 Finding available Ollama server...")
	}
	server, err := m.SelectServer(ctx)
	if err != nil {
		return "", err
	}
	defer func() { m.serverLocks[server.URL] <- struct{}{} }()

	if statusCallback != nil {
		statusCallback(fmt.Sprintf("🤖 AI is thinking... (using %s)", server.Name))
	}

	return m.performChat(ctx, server, req, statusCallback)
}

// performChat handles the chat loop, including tool calls.
func (m *Model) performChat(ctx context.Context, server OllamaServer, req ChatRequest, statusCallback StatusCallback) (string, error) {

	toolCallCount := 0
	for {
		respBody, err := m.postChat(ctx, server.URL, req)
		if err != nil {
			return "", err
		}

		req.Messages = append(req.Messages, respBody.Message)

		if len(respBody.Message.ToolCalls) == 0 {
			if statusCallback != nil {
				statusCallback("✨ Formatting response...")
			}
			return respBody.Message.Content, nil
		}

		if toolCallCount >= maxToolCalls {
			return "", errors.New("maximum tool call depth exceeded")
		}
		toolCallCount++

		req, err = m.handleToolCalls(req, respBody.Message.ToolCalls, statusCallback)
		if err != nil {
			return "", err
		}

		if statusCallback != nil {
			statusCallback(fmt.Sprintf("🤖 AI is processing tool results... (using %s)", server.Name))
		}
	}
}

// prepareInitialRequest creates the initial ChatRequest from templates and tools.
func (m *Model) prepareInitialRequest(message string, args map[string]any, statusCallback StatusCallback) (ChatRequest, error) {
	if statusCallback != nil {
		statusCallback("📝 Preparing message templates...")
	}

	system, err := m.ExecuteSystemTemplate(args)
	if err != nil {
		return ChatRequest{}, err
	}
	user, err := m.ExecuteUserTemplate(message, args)
	if err != nil {
		return ChatRequest{}, err
	}

	req := ChatRequest{
		Model:  m.Name,
		Stream: false,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}

	for _, tool := range m.mcp.GetTools() {
		if tool != nil {
			req.Tools = append(req.Tools, tool.Item)
		}
	}

	return req, nil
}

// handleToolCalls processes tool calls from the model's response.
func (m *Model) handleToolCalls(req ChatRequest, toolCalls []ToolCall, statusCallback StatusCallback) (ChatRequest, error) {
	for _, tc := range toolCalls {
		tool, ok := m.mcp.GetTools()[tc.Function.Name]
		if !ok || tool == nil {
			slog.Warn("Tool not found", "tool", tc.Function.Name)
			req.Messages = append(req.Messages, ChatMessage{
				Role:    "tool",
				Content: "Tool '" + tc.Function.Name + "' not found",
			})
			continue
		}

		if statusCallback != nil {
			toolName := tc.Function.Name
			if displayName, ok := m.ToolNames[toolName]; ok {
				toolName = displayName
			}
			statusCallback(toolName)
		}

		tresp, err := tool.Call(tc.Function.Arguments)
		if err != nil {
			slog.Warn("Unable to call tool", "error", err.Error(), "tool", tc.Function.Name)
			req.Messages = append(req.Messages, ChatMessage{
				Role:    "tool",
				Content: "Tool '" + tc.Function.Name + "' call error: " + err.Error(),
			})
			continue
		}
		req.Messages = append(req.Messages, ChatMessage{
			Role:    "tool",
			Content: strings.Join(tresp, "\n"),
		})
	}
	return req, nil
}

// postChat performs the actual POST request to /api/chat and unmarshals the
// response into a ChatResponse. It returns a descriptive error if anything
// goes wrong.
func (m *Model) postChat(ctx context.Context, server string, req ChatRequest) (*ChatResponse, error) {
	resp, err := m.client.R().
		SetContext(ctx).
		SetBody(req).
		Post(server + "/api/chat")
	if err != nil {
		slog.Error("Ollama call error", "error", err)
		return nil, fmt.Errorf("ollama call failed: %w", err)
	}
	if resp.StatusCode() != 200 {
		slog.Error("Ollama call returned non-200 status", "status", resp.StatusCode(), "body", string(resp.Bytes()))
		return nil, fmt.Errorf("server returned status %s", resp.Status())
	}

	var respBody ChatResponse
	if err := json.Unmarshal(resp.Bytes(), &respBody); err != nil {
		slog.Error("Unable to parse response", "error", err, "response", string(resp.Bytes()))
		return nil, fmt.Errorf("failed to parse response body: %w", err)
	}
	return &respBody, nil
}
