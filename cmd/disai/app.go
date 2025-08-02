package main

import (
	"log/slog"

	"github.com/FlameInTheDark/disai/internal/mcp"
	"github.com/FlameInTheDark/disai/internal/model"
	"github.com/bwmarrin/discordgo"

	"github.com/FlameInTheDark/disai/internal/config"
)

type App struct {
	s *discordgo.Session

	model *model.Model

	handlers map[string]func(s *discordgo.Session, m *discordgo.InteractionCreate)
}

func NewApp(cfg config.Config) *App {
	mcpClient := mcp.NewClient(cfg.MCPServers)
	modelClient := model.NewModel(cfg.Model, cfg.OllamaServers, mcpClient, cfg.Templates.System, cfg.Templates.User)

	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		panic(err)
	}

	return &App{
		s:     s,
		model: modelClient,
	}
}

func (a *App) Run() error {
	err := a.s.Open()
	if err != nil {
		return err
	}
	user, err := a.s.User("@me")
	if err != nil {
		return err
	}
	slog.Info("Logged in", slog.String("username", user.Username), slog.String("discriminator", user.Discriminator))
	return nil
}
