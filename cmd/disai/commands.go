package main

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

func (a *App) createCommands() {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "chat",
			Description: "Ask AI to do something",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "message for AI",
					Required:    true,
				},
			},
			IntegrationTypes: &[]discordgo.ApplicationIntegrationType{
				discordgo.ApplicationIntegrationGuildInstall,
				discordgo.ApplicationIntegrationUserInstall,
			},
			Contexts: &[]discordgo.InteractionContextType{
				discordgo.InteractionContextGuild,
				discordgo.InteractionContextBotDM,
				discordgo.InteractionContextPrivateChannel,
			},
		},
	}

	for _, command := range commands {
		_, err := a.s.ApplicationCommandCreate(a.s.State.User.ID, "", command)
		if err != nil {
			slog.Error("Unable to create command: ", slog.String("command", command.Name))
		}
	}
}

func (a *App) registerHandlers() {
	a.handlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"chat": a.chatHandler,
	}

	a.s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := a.handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}
