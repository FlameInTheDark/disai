package mcp

import (
	"context"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	gmcp "github.com/firebase/genkit/go/plugins/mcp"
)

// Client wraps a Genkit MCP manager for interacting with MCP servers.
// It connects to all configured servers and exposes their tools to Genkit.
type Client struct {
	clients []*gmcp.GenkitMCPClient
}

// NewClient creates MCP clients for all provided servers using the streamable HTTP transport.
func NewClient(servers map[string]string) *Client {
	var cls []*gmcp.GenkitMCPClient
	for name, url := range servers {
		cl, err := gmcp.NewGenkitMCPClient(gmcp.MCPClientOptions{
			Name:           name,
			StreamableHTTP: &gmcp.StreamableHTTPConfig{BaseURL: url},
		})
		if err != nil {
			panic(err)
		}
		cls = append(cls, cl)
	}
	return &Client{clients: cls}
}

// GetTools aggregates all active tools from connected MCP servers and returns them.
func (c *Client) GetTools(ctx context.Context, g *genkit.Genkit) ([]ai.Tool, error) {
	var tools []ai.Tool
	for _, cl := range c.clients {
		t, err := cl.GetActiveTools(ctx, g)
		if err != nil {
			return nil, err
		}
		if t != nil {
			tools = append(tools, t...)
		}
	}
	return tools, nil
}
