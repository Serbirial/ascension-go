package commands

import (
	"ascension/models"
	"ascension/utils/checks"
)

var LoopCommand = models.Command{
	Name:     "loop",
	Desc:     "Loops the currently playing song.", // TODO: add looping of the whole queue
	Aliases:  []string{"lo"},
	Checks:   []func(*models.Context) error{},
	Callback: loopCommand,
}

func loopCommand(ctx *models.Context, args map[string]string) {
	err := checks.UserInVoice(ctx)
	if err != nil {
		ctx.Send("You are not in a Voice Channel.")
		return
	}
	loop, exists := ctx.Client.IsLooping[ctx.GuildID]
	if !exists { // Map is empty, set to true
		ctx.Client.SetLoopingBool(ctx.GuildID, true)
		ctx.Send("Now looping.")
		return
	} else if loop { // Looping is true, set to false
		ctx.Client.SetLoopingBool(ctx.GuildID, false)
		ctx.Send("No longer looping")
		return
	} else if !loop { // Looping is false, set to true
		ctx.Client.SetLoopingBool(ctx.GuildID, true)
		ctx.Send("Now looping.")
		return
	}
}
