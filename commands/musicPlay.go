package commands

import (
	"fmt"
	"time"

	"ascension/handlers"
	"ascension/models"
	"ascension/utils/checks"
)

const AUDIO_FOLDER string = "audio_temp"

var PlayCommand = models.Command{
	Name:     "play",
	Desc:     "Plays a song from youtube.",
	Aliases:  []string{"pl"},
	Args:     map[string]string{"url": "The name or link of the song you want to play."},
	Checks:   []func(*models.Context) error{},
	Callback: playCommand,
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

	// Start the WS connection for the guild and create everything needed
	_, exists := ctx.Client.Websockets[ctx.GuildID]
	if !exists {
		_ = ctx.Client.ConnectToWS(ctx.Client.WsUrl, ctx.Client.WsOrigin, ctx.GuildID)
		ctx.Client.SkipChannels[ctx.GuildID] = make(chan bool)
		ctx.Client.StopChannels[ctx.GuildID] = make(chan bool)
		ctx.Client.SeekChannels[ctx.GuildID] = make(chan int)

	}

	// If the bot is currently downloading, wait for download to finish before starting next download.
	for {
		if ctx.Client.IsDownloading {
			// keep looping until IsDownloading is false
			time.Sleep(1 * time.Second)
		} else if !ctx.Client.IsDownloading {
			// exit the loop and download the song
			break
		}
	}

	// Set the downloading bool to true
	ctx.Client.SetDownloadingBool(true)
	// Download the youtube URL to a file
	ctx.Send("Downloading...")
	//songInfo, err := fs.DownloadYoutubeURLToFile(args["url"], AUDIO_FOLDER)
	songInfo, err := ctx.Client.SendDownloadToWS(args["url"], ctx.GuildID)
	if err != nil {
		fmt.Println(err)
		ctx.Send("Error with DownloadURL function.")
		return
	}
	// Set the downloading bool back to false
	ctx.Client.SetDownloadingBool(false)

	// Add the song to the queue
	ctx.Client.AddToQueue(songInfo)

	ctx.Send("Added `" + songInfo.Title + "` to queue")

	// Nothing is playing: start playing song instantly.
	if ctx.Client.IsPlaying == false {
		ctx.Client.SetPlayingBool(true)                   // Set playing
		ctx.Client.SendPlayToWS(args["url"], ctx.GuildID) // Notify the WS server to start playing the song
		handlers.PlayFromWS(voice, ctx, songInfo, ctx.Client.StopChannels[ctx.GuildID], ctx.Client.SkipChannels[ctx.GuildID], ctx.Client.SeekChannels[ctx.GuildID])

	}
}
