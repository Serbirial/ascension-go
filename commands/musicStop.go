package commands

import (
	"gobot/models"
	"gobot/utils/checks"
)

var StopCommand = models.Command{
	Name:          "play",
	Desc:          "Plays a song from youtube.",
	Aliases:       []string{"pl"},
	Args:          map[string]string{"url": "The name or link of the song you want to play."},
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
	err := checks.InVoice(ctx)
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
