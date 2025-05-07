package commands

import (
	"gobot/models"
	"time"
)

var PingCommand = models.Command{
	Name:     "ping",
	Desc:     "Shows the bots ping.",
	Aliases:  []string{"p"},
	Callback: pingCommand,
}

func pingCommand(ctx *models.Context, args map[string]string) {
	var latency time.Duration = ctx.Client.Session.HeartbeatLatency().Round(time.Millisecond)
	ctx.Send(latency.String())
}
