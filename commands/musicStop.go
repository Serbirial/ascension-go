package commands

import (
	"gobot/models"
	"gobot/utils/checks"
	"time"
)

var StopCommand = models.Command{
	Name:     "stop",
	Desc:     "Stops the currently playing song.",
	Aliases:  []string{"pl"},
	Checks:   []func(*models.Context) error{},
	Callback: stopCommand,
}

func stopCommand(ctx *models.Context, args map[string]string) {
	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err := checks.UserInVoice(ctx)
	if err != nil {
		ctx.Send("You are not in a Voice Channel.")
		return
	}

	ctx.Send("Sending stop...")
	select {
	case ctx.Client.StopChannel <- true:
		ctx.Send("Done.")
		return

	case <-time.After(5 * time.Second):
		ctx.Send("Took too long to quit player.")
		return
	}
}
