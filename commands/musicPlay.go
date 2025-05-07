package commands

import (
	"fmt"

	"gobot/handlers"
	"gobot/models"
	"gobot/utils/checks"
	"gobot/utils/fs"
)

const AUDIO_FOLDER string = "audio_temp"

var PlayCommand = models.Command{
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

func playCommand(ctx *models.Context, args map[string]string) {
	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err := checks.InVoice(ctx)
	if err != nil {
		_, err := ctx.Client.Session.ChannelVoiceJoin(ctx.GuildID, ctx.ChannelID, false, true)
		if err != nil {
			fmt.Println(err)
			ctx.Send("Error joining the voice channel")
			return
		}
	}
	voice, err := checks.GetVoiceChannel(ctx)
	if err != nil {
		fmt.Println(err)
		ctx.Send("Error getting the voice channel")
		return
	}

	filepath, err := fs.DownloadYoutubeURLToFile(args["url"], AUDIO_FOLDER)
	if err != nil {
		fmt.Println(err)
		ctx.Send("Error with DownloadURL function.")
		return
	}

	// ctx.Client.Session.UpdateCustomStatus("Playing: " + file)

	handlers.PlayAudioFile(voice, filepath, ctx.Client.StopChannel)

	// Close connections
	// voice.Close()

	return
}
