package commands

import (
	"gobot/models"
	"strconv"
)

var TestCommand = models.Command{
	Name:     "test",
	Desc:     "testing.",
	Aliases:  []string{"pl"},
	Checks:   []func(*models.Context) error{},
	Callback: testCommand,
}

func testCommand(ctx *models.Context, args map[string]string) {
	ctx.Send("IsPlaying: " + strconv.FormatBool(ctx.Client.IsPlaying))
	ctx.Client.SetPlayingBool(!ctx.Client.IsPlaying)
	ctx.Send("IsPlaying: " + strconv.FormatBool(ctx.Client.IsPlaying))

}
