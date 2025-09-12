package model

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/FlameInTheDark/disai/internal/mcp"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/ollama"
)

const maxToolCalls = 10

// StatusCallback is called to report the current status of the Chat operation.
type StatusCallback func(status string)

// Model wraps a Genkit instance and MCP manager to handle chat requests.
type Model struct {
	name      string
	g         *genkit.Genkit
	mcp       *mcp.Client
	ToolNames map[string]string

	systemTpl *template.Template
	userTpl   *template.Template
}

// statusTool wraps an ai.Tool to report usage via the status callback.
type statusTool struct {
	ai.Tool
	displayName string
	status      StatusCallback
	turn        *int
}

func (t *statusTool) RunRaw(ctx context.Context, input any) (any, error) {
	name := t.displayName
	if name == "" {
		raw := t.Tool.Name()
		if parts := strings.SplitN(raw, "_", 2); len(parts) == 2 {
			name = parts[1]
		} else {
			name = raw
		}
	}
	if t.status != nil && t.turn != nil {
		t.status(fmt.Sprintf("🔧 Turn %d: %s", *t.turn, name))
		(*t.turn)++
	}
	return t.Tool.RunRaw(ctx, input)
}

// NewModel initialises Genkit with the Ollama plugin and connects MCP servers.
// Only the first Ollama server URL is used as Genkit's Ollama plugin supports a
// single server.
func NewModel(modelName string, servers map[string]string, mcpc *mcp.Client, system, user string, toolNames map[string]string) *Model {
	var serverURL string
	for _, url := range servers {
		serverURL = url
		break
	}
	ctx := context.Background()
	o := &ollama.Ollama{ServerAddress: serverURL}
	g := genkit.Init(ctx, genkit.WithPlugins(o), genkit.WithDefaultModel("ollama/"+modelName))
	o.DefineModel(g, ollama.ModelDefinition{Name: modelName, Type: "chat"}, &ai.ModelOptions{
		Supports: &ai.ModelSupports{
			Multiturn:  true,
			SystemRole: true,
			Tools:      true,
		},
	})

	m := &Model{
		name:      modelName,
		g:         g,
		mcp:       mcpc,
		ToolNames: toolNames,
	}
	m.LoadTemplate(system, user)
	return m
}

// Chat sends a message to the model without status updates.
func (m *Model) Chat(ctx context.Context, message string, args map[string]any) (string, error) {
	return m.ChatWithStatus(ctx, message, args, nil)
}

// ChatWithStatus generates a response using Genkit and reports progress via the callback.
func (m *Model) ChatWithStatus(ctx context.Context, message string, args map[string]any, status StatusCallback) (string, error) {
	if status != nil {
		status("📝 Preparing message templates...")
	}

	system, err := m.ExecuteSystemTemplate(args)
	if err != nil {
		return "", err
	}
	user, err := m.ExecuteUserTemplate(message, args)
	if err != nil {
		return "", err
	}

	if status != nil {
		status("🔧 Loading tools...")
	}
	tools, err := m.mcp.GetTools(ctx, m.g)
	if err != nil {
		return "", err
	}

	var turn int
	if status != nil {
		turn = 1
		wrapped := make([]ai.Tool, len(tools))
		for i, t := range tools {
			toolName := t.Name()
			display := m.ToolNames[toolName]
			if display == "" {
				if parts := strings.SplitN(toolName, "_", 2); len(parts) == 2 {
					display = m.ToolNames[parts[1]]
				}
			}
			wrapped[i] = &statusTool{Tool: t, displayName: display, status: status, turn: &turn}
		}
		tools = wrapped
	}

	if status != nil {
		status("🤖 AI is thinking...")
	}

	refs := make([]ai.ToolRef, len(tools))
	for i, t := range tools {
		refs[i] = t
	}

	resp, err := genkit.GenerateText(ctx, m.g,
		ai.WithModelName("ollama/"+m.name),
		ai.WithMessages(
			ai.NewSystemTextMessage(system),
			ai.NewUserTextMessage(user),
		),
		ai.WithTools(refs...),
		ai.WithMaxTurns(maxToolCalls),
	)
	if err != nil {
		return "", err
	}

	if status != nil {
		status("✨ Formatting response...")
	}
	return resp, nil
}
