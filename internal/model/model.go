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
			tool := t
			def := t.Definition()
			rawName := t.Name()
			display := m.ToolNames[rawName]
			if display == "" {
				if parts := strings.SplitN(rawName, "_", 2); len(parts) == 2 {
					display = m.ToolNames[parts[1]]
				}
				if display == "" {
					if parts := strings.SplitN(rawName, "_", 2); len(parts) == 2 {
						display = parts[1]
					} else {
						display = rawName
					}
				}
			}
			disp := display

			wrapped[i] = ai.NewToolWithInputSchema[any](
				def.Name,
				def.Description,
				def.InputSchema,
				func(tc *ai.ToolContext, input any) (any, error) {
					if status != nil {
						status(fmt.Sprintf("🔧 Turn %d: %s", turn, disp))
						turn++
					}
					return tool.RunRaw(tc.Context, input)
				},
			)
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
