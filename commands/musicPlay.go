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
	err := checks.UserInVoice(ctx)
	if err != nil {
		ctx.Send("You are not in a Voice Channel.")
		return
	}
	// Connect to voice channel.
	// NOTE: Setting mute to false, deaf to true.
	err = checks.BotInVoice(ctx)
	if err != nil {
		channelID, err := checks.GetUserVoiceChannel(ctx)
		_, err = ctx.Client.Session.ChannelVoiceJoin(ctx.GuildID, channelID, false, true)
		if err != nil {
			fmt.Println(err)
			ctx.Send("Error joining the voice channel")
			return
		}
	}
	voice, err := checks.GetBotVoiceChannel(ctx)
	if err != nil {
		fmt.Println(err)
		ctx.Send("Error getting the voice channel")
		return
	}

	// Download the youtube URL to a file
	ctx.Send("Downloading...")
	songInfo, err := fs.DownloadYoutubeURLToFile(args["url"], AUDIO_FOLDER)
	if err != nil {
		fmt.Println(err)
		ctx.Send("Error with DownloadURL function.")
		return
	}
	ctx.Send("Done.")

	// Add the song to the queue
	ctx.Client.SongQueue = append(ctx.Client.SongQueue, songInfo)

	// ctx.Client.Session.UpdateCustomStatus("Playing: " + file)

	// Nothing is playing: start playing song instantly.
	if !ctx.Client.IsPlaying {
		ctx.Send("Playing: " + songInfo.Title)
		handlers.PlayAudioFile(voice, ctx, songInfo.FilePath, ctx.Client.StopChannel)
	}

	// Close connections
	// voice.Close()

	return
}
