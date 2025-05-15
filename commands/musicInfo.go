package commands

import (
	"ascension/models"
	"ascension/utils/checks"
)

var MusicInfoCommand = models.Command{
	Name:     "nowplaying",
	Desc:     "Shows the info about the currently playing song.",
	Aliases:  []string{"np"},
	Checks:   []func(*models.Context) error{},
	Callback: musicInfoCommand,
}

func musicInfoCommand(ctx *models.Context, args map[string]string) {
	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err := checks.UserInVoice(ctx)
	if err != nil {
		ctx.Send("You are not in a Voice Channel.")
		return
	}

	var song *models.SongInfo = ctx.Client.SongQueue[ctx.GuildID][0]

	ctx.Send(song.Title + " - " + song.Uploader)

}
