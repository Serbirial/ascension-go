package checks

import (
	"errors"
	"fmt"
	"gobot/models"

	"github.com/bwmarrin/discordgo"
)

func BotInVoice(ctx *models.Context) error {
	guild, err := ctx.Client.Session.State.Guild(ctx.GuildID)
	if err != nil {
		fmt.Println("Error fetching guild:", err)
		return err
	}
	// Iterate through the voice states to find the user
	for _, voiceState := range guild.VoiceStates {
		if voiceState.UserID == ctx.Client.Session.State.User.ID {
			return nil
		}
	}

	return errors.New("Not in voice channel.")
}

func UserInVoice(ctx *models.Context) error {
	guild, err := ctx.Client.Session.State.Guild(ctx.GuildID)
	if err != nil {
		fmt.Println("Error fetching guild:", err)
		return err
	}
	// Iterate through the voice states to find the user
	for _, voiceState := range guild.VoiceStates {
		if voiceState.UserID == ctx.Author.ID {
			return nil
		}
	}

	return errors.New("Not in voice channel.")
}

func GetBotVoiceChannel(ctx *models.Context) (*discordgo.VoiceConnection, error) {
	err := BotInVoice(ctx)
	if err != nil {
		return nil, errors.New("Bot not in voice channel.")
	}
	return ctx.Client.Session.VoiceConnections[ctx.GuildID], nil
}

func GetUserVoiceChannel(ctx *models.Context) (string, error) {
	err := UserInVoice(ctx)
	if err != nil {
		return "", errors.New("User not in voice channel.")
	}

	guild, err := ctx.Client.Session.State.Guild(ctx.GuildID)
	if err != nil {
		fmt.Println("Error fetching guild:", err)
		return "", err
	}
	// Iterate through the voice states to find the user
	for _, voiceState := range guild.VoiceStates {
		if voiceState.UserID == ctx.Author.ID {
			return voiceState.ChannelID, nil
		}
	}
	return "", errors.New("Could not find channel")
}
