package commands

import (
	"ascension/models"
	"ascension/utils/checks"
	"time"
)

var SkipCommand = models.Command{
	Name:     "skip",
	Desc:     "Skips the currently playing song.",
	Aliases:  []string{"sk"},
	Checks:   []func(*models.Context) error{},
	Callback: skipCommand,
}

func skipCommand(ctx *models.Context, args map[string]string) {
	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err := checks.UserInVoice(ctx)
	if err != nil {
		ctx.Send("You are not in a Voice Channel.")
		return
	}

	ctx.Send("Sending skip...")
	select {
	case ctx.Client.SkipChannel <- true:
		ctx.Send("Done.")
		return

	case <-time.After(5 * time.Second):
		ctx.Send("Took too long to send the skip signal.")
		return
	}
}
