package commands

import (
	"fmt"

	"gobot/models"
	"gobot/utils/checks"
)

var JoinCommand = models.Command{
	Name:          "join",
	Desc:          "Joins the voice channel.",
	Aliases:       []string{"j"},
	Args:          nil,
	Subcommands:   []string{""},
	Parentcommand: "none",
	Checks:        []func(*models.Context) error{},
	Callback:      playCommand,
	Nsfw:          false,
	Endpoint:      "string",
}

func joinCommand(ctx *models.Context, args map[string]string) {
	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err := checks.InVoice(ctx)
	if err != nil {
		_, err := ctx.Client.Session.ChannelVoiceJoin(ctx.GuildID, ctx.ChannelID, false, true)
		if err != nil {
			fmt.Println(err)
			ctx.Send("Error joining the voice channel")
			return
		}
	}
	return
}
