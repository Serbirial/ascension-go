package commands

import (
	"gobot/models"
	"gobot/utils/checks"
)

var StopCommand = models.Command{
	Name:          "stop",
	Desc:          "Stops the currently playing song.",
	Aliases:       []string{"pl"},
	Args:          nil,
	Subcommands:   []string{""},
	Parentcommand: "none",
	Checks:        []func(*models.Context) error{},
	Callback:      playCommand,
	Nsfw:          false,
	Endpoint:      "string",
}

func stopCommand(ctx *models.Context, args map[string]string) {
	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err := checks.UserInVoice(ctx)
	if err != nil {
		ctx.Send("Not in voice or playing.")
		return
	}

	ctx.Send("Sending stop...")
	channel := ctx.Client.StopChannel
	channel <- true
	ctx.Send("Done.")

	return
}
