package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	embedAuthorThinking = "Thinking..."
	embedAuthorError    = "Error"
)

var emojiRegex = regexp.MustCompile(`^\p{So}`)

// createEmbed builds a Discord embed with optional description and footer.
func createEmbed(author, description, footerText string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name: author,
		},
	}
	if description != "" {
		embed.Description = description
	}
	if footerText != "" {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: footerText,
		}
	}
	return embed
}

// sendInteractionResponse sends an initial response (e.g., “Thinking…” or “Error”).
func sendInteractionResponse(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, flags discordgo.MessageFlags) error {
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:  flags,
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	}
	if err := s.InteractionRespond(i.Interaction, resp); err != nil {
		slog.Error("Unable to send response", slog.String("error", err.Error()))
		return err
	}
	return nil
}

// sendInteractionResponseEdit updates a previously sent response.
func sendInteractionResponseEdit(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) error {
	respEdit := &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, respEdit); err != nil {
		slog.Error("Unable to send response", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (a *App) chatHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Validate input once
	if len(i.ApplicationCommandData().Options) == 0 {
		slog.Warn("No message provided")
		a.errorResponse(s, i)
		return
	}
	userInput := strings.TrimSpace(i.ApplicationCommandData().Options[0].StringValue())
	if userInput == "" {
		slog.Warn("No message provided")
		a.errorResponse(s, i)
		return
	}

	// Send “Thinking…” response
	if err := a.thinkingResponse(s, i); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	start := time.Now()

	escapedInput := url.QueryEscape(strings.ReplaceAll(userInput, "\t", "    "))

	// Status history to track all actions
	var statusHistory []string

	// Status callback to update Discord embed with accumulated history
	statusCallback := func(status string) {
		// Add new status to the history
		statusHistory = append(statusHistory, status)

		// Create a display history
		var displayHistory []string
		for i, s := range statusHistory {
			if i < len(statusHistory)-1 {
				// Previous status, mark as completed
				displayHistory = append(displayHistory, emojiRegex.ReplaceAllString(s, "✅"))
			} else {
				// Current status, display as is
				displayHistory = append(displayHistory, s)
			}
		}

		elapsed := time.Since(start).Seconds()
		footer := fmt.Sprintf("Elapsed: %.1fs", elapsed)

		// Create description with all status history
		description := strings.Join(displayHistory, "\n")

		// Show user request in embed title during processing
		processingTitle := fmt.Sprintf("Processing: %s", CropText(userInput, 200))
		statusEmbed := createEmbed(processingTitle, description, footer)
		if err := sendInteractionResponseEdit(s, i, statusEmbed); err != nil {
			slog.Warn("Unable to update status", slog.String("error", err.Error()))
		}
	}

	resp, err := a.model.ChatWithStatus(ctx, escapedInput, map[string]any{
		"UserId":   i.Member.User.ID,
		"Username": i.Member.User.Username,
	}, statusCallback)
	if err != nil {
		slog.Error("Unable to chat", slog.String("error", err.Error()))
		errorEmbed := createEmbed(embedAuthorError, "", "")
		if err := sendInteractionResponseEdit(s, i, errorEmbed); err != nil {
			return
		}
		return
	}
	end := time.Now()

	// Mark final status as completed
	if len(statusHistory) > 0 {
		lastIdx := len(statusHistory) - 1
		statusHistory[lastIdx] = emojiRegex.ReplaceAllString(statusHistory[lastIdx], "✅")
	}

	var result string
	if data := ExtractAfterLastThinkTag(resp); len(data) > 0 {
		result = CropText(data, 4096)
	} else {
		result = "AI was thinking too hard so it provided no response... Try again later."
	}

	// Create clean final response without process history
	chatEmbed := createEmbed(
		fmt.Sprintf("Chat: %s", CropText(userInput, 240)),
		CropText(result, 4096), // Full space for AI response
		fmt.Sprintf("Response time: %.2fs", end.Sub(start).Seconds()),
	)

	if err := sendInteractionResponseEdit(s, i, chatEmbed); err != nil {
		return
	}
}

func (a *App) thinkingResponse(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	thinkingEmbed := createEmbed(embedAuthorThinking, "", "")
	return sendInteractionResponse(s, i, thinkingEmbed, discordgo.MessageFlagsLoading)
}

func (a *App) errorResponse(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	errorEmbed := createEmbed(embedAuthorError, "", "")
	return sendInteractionResponse(s, i, errorEmbed, discordgo.MessageFlagsLoading)
}
