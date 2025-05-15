package commands

import (
	"gobot/models"
	"gobot/utils/checks"
	"strconv"
	"time"
)

var SeekCommand = models.Command{
	Name:    "seek",
	Desc:    "Seeks forward in the currently playing song.",
	Aliases: []string{"se"},
	Args:    map[string]string{"time": "The ammount of seconds to seek."},

	Checks:   []func(*models.Context) error{},
	Callback: seekCommand,
}

func seekCommand(ctx *models.Context, args map[string]string) {
	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err := checks.UserInVoice(ctx)
	if err != nil {
		ctx.Send("You are not in a Voice Channel.")
		return
	}
	seekTime, err := strconv.Atoi(args["time"])
	if err != nil {
		ctx.Send("Couldnt not turn given arg into a number.")
		return
	}

	ctx.Send("Sending seek...")
	select {
	case ctx.Client.SeekChannel <- seekTime:
		ctx.Send("Done.")
		return

	case <-time.After(5 * time.Second):
		ctx.Send("Took too long to send the seek signal.")
		return
	}
}
