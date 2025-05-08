package commands

import (
	"gobot/models"
)

var QueueCommand = models.Command{
	Name:     "queue",
	Desc:     "Shows the queue.",
	Aliases:  []string{"pl"},
	Checks:   []func(*models.Context) error{},
	Callback: queueCommand,
}

func queueCommand(ctx *models.Context, args map[string]string) {
	var msg string

	for _, songInfo := range ctx.Client.SongQueue {
		msg += songInfo.Title + " - " + songInfo.Uploader + "\n"
	}

	ctx.Send(msg)
}
