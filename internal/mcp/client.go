package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-viper/mapstructure/v2"
	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"
)

type Client struct {
	mcps map[string]*mcp.Client
}

type Tool struct {
	Name string
	Item ToolItem
	c    *mcp.Client
}

func (t *Tool) Call(args map[string]any) ([]string, error) {
	resp, err := t.c.CallTool(context.Background(), t.Name, args)
	if err != nil {
		slog.Error("Unable to call tool", slog.String("error", err.Error()), slog.String("tool", t.Name))
		return nil, err
	}
	var responses = make([]string, len(resp.Content))

	for i, r := range resp.Content {
		if r != nil {
			switch r.Type {
			case "text":
				if r.TextContent != nil {
					responses[i] = r.TextContent.Text
				}
			default:
				responses[i] = fmt.Sprintf("unknown response type: ", r.Type)
			}
		}
	}
	return responses, nil
}

func NewClient(mcpServers map[string]string) *Client {
	var c = &Client{
		mcps: make(map[string]*mcp.Client),
	}
	for s, url := range mcpServers {
		transport := http.NewHTTPClientTransport("/mcp")
		transport.WithBaseURL(url)
		client := mcp.NewClient(transport)
		init, err := client.Initialize(context.Background())
		if err != nil {
			slog.Error("Unable to initialize client", slog.String("error", err.Error()), slog.String("mcp", s))
			continue
		}
		slog.Info("MCP server initialized", slog.String("name", init.ServerInfo.Name), slog.String("version", init.ServerInfo.Version))
		c.mcps[s] = client
	}
	return c
}

func (c *Client) GetTools() map[string]*Tool {
	var tools = make(map[string]*Tool)
	for n, client := range c.mcps {
		tl, err := client.ListTools(context.Background(), nil)
		if err != nil {
			slog.Warn("Unable to get tools", slog.String("error", err.Error()), slog.String("mcp", n))
			continue
		}

		for _, t := range tl.Tools {
			mapped, ok := t.InputSchema.(map[string]any)
			if !ok {
				slog.Warn("Unable to get schema", slog.String("error", "invalid schema"), slog.String("mcp", n))
				continue
			}

			var schema InputSchema

			mapstructure.Decode(mapped, &schema)
			tools[t.Name] = &Tool{
				Name: t.Name,
				Item: ToolItem{
					Type: "function",
					Function: ToolFunction{
						Name:        t.Name,
						Description: t.Description,
						Parameters: ToolParameters{
							Type:       schema.Type,
							Properties: schema.Properties,
						},
						Required: schema.Required,
					},
				},
				c: client,
			}
		}
	}
	return tools
}

type InputSchema struct {
	Schema     string                  `json:"$schema"`
	Type       string                  `json:"type"`
	Required   []string                `json:"required"`
	Properties map[string]ToolProperty `json:"properties"`
}

type ToolItem struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Parameters  ToolParameters `json:"parameters"`
	Required    []string       `json:"required"`
}

type ToolParameters struct {
	Type       string                  `json:"type"`
	Properties map[string]ToolProperty `json:"properties"`
}

type ToolProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

func ListTools(tools map[string]*Tool) {
	var toolList []ToolItem
	for _, t := range tools {
		toolList = append(toolList, t.Item)
	}
}
