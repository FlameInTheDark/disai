package main

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (a *App) chatHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(i.ApplicationCommandData().Options) == 0 {
		slog.Warn("No message provided")
		return
	}

	err := a.thinkingResponse(s, i)
	if err != nil {
		return
	}

	start := time.Now()
	resp, err := a.model.Chat(i.ApplicationCommandData().Options[0].StringValue(), map[string]any{
		"UserId":   i.Member.User.ID,
		"Username": i.Member.User.Username,
	})
	if err != nil {
		slog.Error("Unable to chat", slog.String("error", err.Error()))
		respEdit := &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Author: &discordgo.MessageEmbedAuthor{
						Name: "Error",
					},
				},
			},
		}

		_, err = a.s.InteractionResponseEdit(i.Interaction, respEdit)
		if err != nil {
			slog.Error("Unable to send response", slog.String("error", err.Error()))
			return
		}
		return
	}
	end := time.Now()

	respEdit := &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Author: &discordgo.MessageEmbedAuthor{
					Name: "Chat: " + i.ApplicationCommandData().Options[0].StringValue(),
				},
				Description: CropText(strings.TrimSpace(strings.Split(resp, "</think>")[1])),
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Response time: %.2fs", end.Sub(start).Seconds()),
				},
			},
		},
	}

	_, err = a.s.InteractionResponseEdit(i.Interaction, respEdit)
	if err != nil {
		slog.Error("Unable to send response", slog.String("error", err.Error()))
		return
	}
}

func (a *App) thinkingResponse(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsLoading,
			Embeds: []*discordgo.MessageEmbed{
				{
					Author: &discordgo.MessageEmbedAuthor{
						Name: "Thinking...",
					},
				},
			},
		}}
	err := s.InteractionRespond(i.Interaction, resp)
	if err != nil {
		slog.Error("Unable to send response", slog.String("error", err.Error()))
		return err
	}
	return nil
}
