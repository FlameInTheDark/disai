package model

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"text/template"

	"resty.dev/v3"

	"github.com/FlameInTheDark/disai/internal/mcp"
)

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

type Model struct {
	Name      string
	Servers   []string
	systemTpl *template.Template
	userTpl   *template.Template

	mcp *mcp.Client
}

func NewModel(modelName string, servers []string, mcpc *mcp.Client, system, user string) *Model {
	m := &Model{
		Name:    modelName,
		Servers: servers,
		mcp:     mcpc,
	}
	m.LoadTemplate(system, user)
	return m
}

func (m *Model) SelectServer() (string, error) {
	c := resty.New()
	defer c.Close()
	for _, server := range m.Servers {
		res, err := c.R().SetBody(ModelInfoRequest{Model: m.Name}).Post(server + "/api/show")
		if err != nil {
			slog.Warn("Unable to connect to server", slog.String("server", server), slog.String("error", err.Error()))
			continue
		}
		if res.StatusCode() != 200 {
			continue
		}
		return server, nil
	}
	return "", errors.New("no server available")
}

func (m *Model) Chat(message string, args map[string]any) (string, error) {
	var req ChatRequest
	system, err := m.ExecuteSystemTemplate(args)
	if err != nil {
		return "", err
	}
	user, err := m.ExecuteUserTemplate(message, args)
	if err != nil {
		return "", err
	}

	req.Model = m.Name
	req.Stream = false
	req.Messages = append(req.Messages, ChatMessage{Role: "system", Content: system})
	req.Messages = append(req.Messages, ChatMessage{Role: "user", Content: user})

	tools := m.mcp.GetTools()
	for _, tool := range tools {
		req.Tools = append(req.Tools, tool.Item)
	}

	server, err := m.SelectServer()
	if err != nil {
		return "", err
	}
	c := resty.New()
	defer c.Close()
	for {
		resp, err := c.R().SetBody(req).Post(server + "/api/chat")
		if err != nil || resp.StatusCode() != 200 {
			slog.Warn("Ollama call error", slog.String("error", err.Error()), slog.Int("status", resp.StatusCode()))
			return "", err
		}
		var respBody ChatResponse
		err = json.Unmarshal(resp.Bytes(), &respBody)
		if err != nil {
			slog.Warn("Unable to parse response", slog.String("error", err.Error()), slog.String("response", string(resp.Bytes())))
			return "", err
		}
		req.Messages = append(req.Messages, respBody.Message)
		if len(respBody.Message.ToolCalls) > 0 {
			for _, tc := range respBody.Message.ToolCalls {
				tresp, err := tools[tc.Function.Name].Call(tc.Function.Arguments)
				if err != nil {
					slog.Warn("Unable to call tool", slog.String("error", err.Error()), slog.String("tool", tc.Function.Name))
					req.Messages = append(req.Messages, ChatMessage{Role: "tool", Content: "Tool '" + tc.Function.Name + "' call error: " + err.Error()})
					continue
				}
				req.Messages = append(req.Messages, ChatMessage{Role: "tool", Content: strings.Join(tresp, "\n")})
			}
		} else {
			return respBody.Message.Content, nil
		}
	}
}
