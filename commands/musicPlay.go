package commands

import (
	"fmt"
	"log"

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
		channelID, _ := checks.GetUserVoiceChannel(ctx)
		_, err = ctx.Client.Session.ChannelVoiceJoin(ctx.GuildID, channelID, false, true)
		if err != nil {
			fmt.Println(err)
			ctx.Send("Error joining the voice channel")
			return
		}
	}

	// Start the WS connection for the guild and create everything needed
	_, exists := ctx.Client.Websockets[ctx.GuildID]
	if !exists {
		_ = ctx.Client.ConnectToWS(ctx.Client.WsUrl, ctx.Client.WsOrigin, ctx.GuildID)
		ctx.Client.SkipChannels[ctx.GuildID] = make(chan bool)
		ctx.Client.StopChannels[ctx.GuildID] = make(chan bool)
		ctx.Client.SeekChannels[ctx.GuildID] = make(chan int)

	}
	var stopWhenChecked bool = false
	// Parse spotify to youtube
	if handlers.ContainsSpotify(args["url"]) {
		stype, sid, err := handlers.ParseSpotifyURL(args["url"])
		if err != nil {
			ctx.Send("Error parsing spotify URL")
			return
		}
		token, err := handlers.GetSpotifyAccessToken(ctx.Client.SpotifyID, ctx.Client.SpotifySecret)
		if err != nil {
			ctx.Send("Error while getting Spotify Auth Token.")
			return
		}

		if stype == "track" {
			title, artist, err := handlers.GetTrackTitleAndArtist(sid, token)
			if err != nil {
				log.Println(err)
				ctx.Send(err.Error())
				return
			}
			query := fmt.Sprintf("%s - %s", title, artist)
			ctx.Send("Grabbing first result for `" + query + "` on youtube.")
			videoID, err := ctx.Client.SendSearchRequest(query)
			if err != nil {
				log.Println(err)
				ctx.Send(err.Error())
				return
			}
			args["url"] = fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
			// Continue as normal, will add the single youtube link to the download queue
		} else if stype == "playlist" {
			results, err := handlers.GetPlaylistTitlesAndArtists(sid, token)
			if err != nil {
				log.Println(err)
				ctx.Send(err.Error())
				return
			}

			ctx.Client.SetDownloadingBool(ctx.GuildID, true)
			var isFirst bool = true
			ctx.Send("Downloading...")

			for _, track := range results {
				query := fmt.Sprintf("%s - %s", track.Title, track.Artist)
				ctx.Send("Grabbing first result for `" + query + "` on YouTube.")
				videoID, err := ctx.Client.SendSearchRequest(query)
				if err != nil {
					log.Println(err)
					ctx.Send(err.Error())
					return
				}
				done := ctx.Client.DownloadQueue.Add(ctx, fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID), ctx.GuildID)
				if isFirst { // Start playing as soon as first download is done
					isFirst = false
					isDone := <-done
					if isDone {
						if !ctx.Client.IsPlaying[ctx.GuildID] {
							voice, err := checks.GetBotVoiceChannel(ctx)
							if err != nil {
								fmt.Println(err)
								ctx.Send("Error getting the voice channel")
								return
							}
							ctx.Client.SetPlayingBool(ctx.GuildID, true)
							ctx.Client.SendPlayToWS(args["url"], ctx.GuildID)
							go handlers.PlayFromWS(
								voice,
								ctx,
								ctx.Client.SongQueue[ctx.GuildID].Current(),
								ctx.Client.StopChannels[ctx.GuildID],
								ctx.Client.SkipChannels[ctx.GuildID],
								ctx.Client.SeekChannels[ctx.GuildID],
							)
						}
					} else if !isDone {
						ctx.Send("Error during downloading, doing nothing.")
					}
				}
			}
			ctx.Client.SetDownloadingBool(ctx.GuildID, false)
			stopWhenChecked = true
		}
	}
	if stopWhenChecked {
		return
	}

	// Download the youtube URL to a file)
	ctx.Send("Downloading...")
	ctx.Client.SetDownloadingBool(ctx.GuildID, true)
	done := ctx.Client.DownloadQueue.Add(ctx, args["url"], ctx.GuildID)
	isDone := <-done
	ctx.Client.SetDownloadingBool(ctx.GuildID, false)

	if isDone {
		if !ctx.Client.IsPlaying[ctx.GuildID] {
			voice, err := checks.GetBotVoiceChannel(ctx)
			if err != nil {
				fmt.Println(err)
				ctx.Send("Error getting the voice channel")
				return
			}
			ctx.Client.SetPlayingBool(ctx.GuildID, true)
			ctx.Client.SendPlayToWS(args["url"], ctx.GuildID)
			handlers.PlayFromWS(
				voice,
				ctx,
				ctx.Client.SongQueue[ctx.GuildID].Current(),
				ctx.Client.StopChannels[ctx.GuildID],
				ctx.Client.SkipChannels[ctx.GuildID],
				ctx.Client.SeekChannels[ctx.GuildID],
			)
		}
	} else if !isDone {
		ctx.Send("Error during downloading, doing nothing.")
	}

}
