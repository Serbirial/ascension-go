package commands

import (
	"fmt"

	"ascension/models"
	"ascension/utils/checks"
)

var JoinCommand = models.Command{
	Name:     "join",
	Desc:     "Joins the voice channel.",
	Aliases:  []string{"j"},
	Checks:   []func(*models.Context) error{},
	Callback: joinCommand,
}

func joinCommand(ctx *models.Context, args map[string]string) {
	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err := checks.UserInVoice(ctx)
	if err != nil {
		ctx.Send("You are not in a Voice Channel.")
		return
	}

	err = checks.BotInVoice(ctx)
	if err != nil {
		channelID, err := checks.GetUserVoiceChannel(ctx)
		_, err = ctx.Client.Session.ChannelVoiceJoin(ctx.GuildID, channelID, false, true)
		if err != nil {
			fmt.Println(err)
			ctx.Send("Error joining the voice channel")
			return
		}
	}
	return
}
