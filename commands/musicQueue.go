package commands

import (
	"ascension/models"
	"strconv"
)

var QueueCommand = models.Command{
	Name:     "queue",
	Desc:     "Shows the queue.",
	Aliases:  []string{"pl"},
	Checks:   []func(*models.Context) error{},
	Callback: queueCommand,
}

func queueCommand(ctx *models.Context, args map[string]string) {
	var msg string = "Queue:"

	for _, songInfo := range ctx.Client.SongQueue[ctx.GuildID].Queue {
		msg += "	" + songInfo.Title + " - " + songInfo.Uploader + "\n"
	}

	ctx.Send(msg)
	ctx.Send("IsPlaying: " + strconv.FormatBool(ctx.Client.IsPlaying[ctx.GuildID]))
}
