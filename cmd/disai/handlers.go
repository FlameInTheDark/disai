package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (a *App) chatHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(i.ApplicationCommandData().Options) == 0 {
		slog.Warn("No message provided")
		a.errorResponse(s, i)
		return
	}
	if len(strings.TrimSpace(i.ApplicationCommandData().Options[0].StringValue())) == 0 {
		slog.Warn("No message provided")
		a.errorResponse(s, i)
		return
	}

	err := a.thinkingResponse(s, i)
	if err != nil {
		return
	}

	start := time.Now()
	resp, err := a.model.Chat(url.QueryEscape(strings.ReplaceAll(i.ApplicationCommandData().Options[0].StringValue(), "\t", "    ")), map[string]any{
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

	var result string
	if data := ExtractAfterLastThinkTag(resp); len(data) > 0 {
		result = CropText(data, 4096)
	} else {
		result = "AI was thinking too hard so it provided no response... Try again later."
	}

	respEdit := &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Author: &discordgo.MessageEmbedAuthor{
					Name: "Chat: " + CropText(i.ApplicationCommandData().Options[0].StringValue(), 240),
				},
				Description: result,
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

func (a *App) errorResponse(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsLoading,
			Embeds: []*discordgo.MessageEmbed{
				{
					Author: &discordgo.MessageEmbedAuthor{
						Name: "Error",
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
