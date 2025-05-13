/*******************************************************************************
 * This is very experimental code and probably a long way from perfect or
 * ideal.  Please provide feed back on areas that would improve performance
 *
 */

// Package dgvoice provides opus encoding and audio file playback for the
// Discordgo package.
package handlers

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"gobot/models"
	"gobot/utils/checks"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

// NOTE: This API is not final and these are likely to change.

// Technically the below settings can be adjusted however that poses
// a lot of other problems that are not handled well at this time.
// These below values seem to provide the best overall performance
const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

var (
	speakers    map[uint32]*gopus.Decoder
	opusEncoder *gopus.Encoder
	mu          sync.Mutex
)

// OnError gets called by dgvoice when an error is encountered.
// By default logs to STDERR
var OnError = func(str string, err error) {
	prefix := "Music: " + str

	if err != nil {
		fmt.Println(prefix + ": " + err.Error())
	} else {
		fmt.Println(prefix)
	}
}

// SendPCM will receive on the provied channel encode
// received PCM data into Opus then send that to Discordgo
// TODO: download as opus or convert to opus so i can cut out usage of gopus opus encoding
func SendPCM(v *discordgo.VoiceConnection, pcm <-chan []int16) {
	if pcm == nil {
		return
	}

	var err error

	opusEncoder, err = gopus.NewEncoder(frameRate, channels, gopus.Audio)

	if err != nil {
		OnError("NewEncoder Error", err)
		return
	}

	for {

		// read pcm from chan, exit if channel is closed.
		recv, ok := <-pcm
		if !ok {
			OnError("PCM Channel closed", nil)
			return
		}
		// try encoding pcm frame with Opus
		opus, err := opusEncoder.Encode(recv, frameSize, maxBytes)
		if err != nil {
			OnError("Encoding Error", err)
			return
		}

		if v.Ready == false || v.OpusSend == nil {
			// OnError(fmt.Sprintf("Discordgo not ready for opus packets. %+v : %+v", v.Ready, v.OpusSend), nil)
			// Sending errors here might not be suited
			return
		}
		// send encoded opus data to the sendOpus channel
		v.OpusSend <- opus
	}
}

// SendDCA will receive on the provied channel then send that to Discordgo
func SendDCA(v *discordgo.VoiceConnection, dca <-chan []byte, stop <-chan bool) {
	if dca == nil {
		return
	}

	for {
		select {
		case <-stop:
			log.Println("[DCA] Stop recognized")
			break
		case <-time.After(10 * time.Millisecond):
		case dca, ok := <-dca:
			// read dca from chan, exit if channel is closed.
			if !ok {
				fmt.Println("[DCA] Channel closed")
				return
			}
			if v.Ready == false || v.OpusSend == nil {
				// fmt.Println(fmt.Sprintf("Discordgo not ready for opus packets. %+v : %+v", v.Ready, v.OpusSend), nil)
				// Sending errors here might not be suited
				return
			}
			// send encoded opus data to the sendOpus channel
			v.OpusSend <- dca
		}
	}
}

// ReceivePCM will receive on the the Discordgo OpusRecv channel and decode
// the opus audio into PCM then send it on the provided channel.
func ReceivePCM(v *discordgo.VoiceConnection, c chan *discordgo.Packet) {
	if c == nil {
		return
	}

	var err error

	for {
		if v.Ready == false || v.OpusRecv == nil {
			OnError(fmt.Sprintf("Discordgo not to receive opus packets. %+v : %+v", v.Ready, v.OpusSend), nil)
			return
		}

		p, ok := <-v.OpusRecv
		if !ok {
			return
		}

		if speakers == nil {
			speakers = make(map[uint32]*gopus.Decoder)
		}

		_, ok = speakers[p.SSRC]
		if !ok {
			speakers[p.SSRC], err = gopus.NewDecoder(48000, 2)
			if err != nil {
				OnError("error creating opus decoder", err)
				continue
			}
		}

		p.PCM, err = speakers[p.SSRC].Decode(p.Opus, 960, false)
		if err != nil {
			OnError("Error decoding opus data", err)
			continue
		}

		c <- p
	}
}

func removeSongFromQueue(ctx *models.Context) []*models.SongInfo {
	// Remove current song from queue
	var temp []*models.SongInfo
	for i := 0; i < len(ctx.Client.SongQueue); i++ {
		if i >= 1 {
			temp = append(temp, ctx.Client.SongQueue[i])
		}
	}
	// Replace queue with updated one
	return temp
}

// This plays the next song in the queue
func playNextSongInQueue(v *discordgo.VoiceConnection, ctx *models.Context, stop <-chan bool, skip <-chan bool) {
	if len(ctx.Client.SongQueue) >= 1 {
		// Get first SongInfo in Queue and play it
		fmt.Println(ctx.Client.SongQueue)
		fmt.Println(ctx.Client.SongQueue[0])

		var song *models.SongInfo = ctx.Client.SongQueue[0]
		PlayDCAFile(v, ctx, song, song.FilePath, stop, skip)
	}
}

func startCleanupProcess(v *discordgo.VoiceConnection, ctx *models.Context, stop <-chan bool, skip <-chan bool) {
	fmt.Println("[Music] Cleanup process started")
	// Stop speaking
	err := checks.BotInVoice(ctx)
	if err != nil {
		v = recoverBotLeftChannel(ctx) // This should only error when the bot leaves pre-maturely
		if v == nil {
			return
		}
	}
	err = v.Speaking(false)
	if err != nil {
		log.Fatalf("Error while setting speaking: %s", err)
	}
	// Remove current song from queue and replace it with the updated one while clearing status
	clearStatusAndRemoveCurrentSongFromQueue(ctx)
	// Set Playing to false
	ctx.Client.SetPlayingBool(false)
	// Check if Queue is empty
	if len(ctx.Client.SongQueue) >= 1 {
		fmt.Println("[Music] Queue is not empty, playing next song")
		// Play the next song
		playNextSongInQueue(v, ctx, stop, skip)
	} else if len(ctx.Client.SongQueue) == 0 { // Queue was empty
		fmt.Println("[Music] Queue is empty, waiting for activity")
		// Wait 60s to see if activity happens
		var tries int = 0
		for {
			if tries >= 300 {
				break
			}
			time.Sleep(1 * time.Second)
			if len(ctx.Client.SongQueue) >= 1 {
				fmt.Println("[Music] Activity in queue")
				break
			}
			tries++
		}
		if len(ctx.Client.SongQueue) == 0 {
			// No activity, Disconnect
			fmt.Println("[Music] Disconnecting because no activity and empty queue")
			v.Disconnect()
			return
		}
	}
}

func clearStatusAndRemoveCurrentSongFromQueue(ctx *models.Context) {
	ctx.Client.Session.UpdateCustomStatus("")
	temp := removeSongFromQueue(ctx)
	ctx.Client.SetQueue(temp)
}

// PlayAudioFile will play the given filename to the already connected
// Discord voice server/channel.  voice websocket and udp socket
// must already be setup before this will work.
func PlayAudioFile(v *discordgo.VoiceConnection, ctx *models.Context, songInfo *models.SongInfo, filename string, stop <-chan bool, skip <-chan bool) {
	// Send "playing" message to the channel
	ctx.Send("Playing: " + songInfo.Title + " - " + songInfo.Uploader)
	// Set status
	ctx.Client.Session.UpdateCustomStatus("Playing: " + songInfo.Title)
	ctx.Client.SetPlayingBool(true)

	// Create a shell command "object" to run.
	run := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		OnError("StdoutPipe Error", err)
		return
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384*20)

	// Starts the ffmpeg command
	err = run.Start()
	if err != nil {
		OnError("RunStart Error", err)
		return
	}

	// prevent memory leak from residual ffmpeg streams
	defer run.Process.Kill()

	//when stop is sent, kill ffmpeg
	go func() {
		signal := <-stop
		fmt.Println("[Music] Received signal")
		if signal == true {
			// Remove the 'Playing X' status
			ctx.Client.Session.UpdateCustomStatus("")
			fmt.Println("[Music] Stop signal sent")
			// Remove current song from queue
			var temp []*models.SongInfo
			for i := 0; i < len(ctx.Client.SongQueue); i++ {
				if i >= 1 {
					temp = append(temp, ctx.Client.SongQueue[i])
				}
			}
			// Replace queue with updated one
			ctx.Client.SetQueue(temp)
			// Kill ffmpeg
			err = run.Process.Kill()
			fmt.Println("[Music] FFMPEG killed")
		}

	}()

	//when skip is sent, do the cleanup process so the next song can be played
	go func() {
		signal := <-skip
		fmt.Println("[Music] Received signal")
		if signal == true {
			// Remove the 'Playing X' status
			ctx.Client.Session.UpdateCustomStatus("")
			fmt.Println("[Music] Skip signal sent")
			err = run.Process.Kill()
			fmt.Println("[Music] FFMPEG killed")
			startCleanupProcess(v, ctx, stop, skip)
		}
	}()

	// Send "speaking" packet over the voice websocket
	err = v.Speaking(true)
	if err != nil {
		OnError("Couldn't set speaking", err)
	}

	// Send not "speaking" packet over the websocket when we finish and start the cleanup
	defer func() {
		// Remove the 'Playing X' status
		ctx.Client.Session.UpdateCustomStatus("")
		err = checks.BotInVoice(ctx)
		if err != nil {
			v = recoverBotLeftChannel(ctx) // This should only error when already not speaking
			if v == nil {
				return
			}
		}
		err = v.Speaking(false)
		if err != nil {
			log.Fatalf("Error while setting speaking: %s", err)

		}

		startCleanupProcess(v, ctx, stop, skip)
	}()

	send := make(chan []int16, 2)
	defer close(send)

	close := make(chan bool)
	go func() {
		SendPCM(v, send)
		close <- true
	}()

	for {
		// read data from ffmpeg stdout
		var data []int16 = make([]int16, frameSize*channels)
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &data)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			OnError("error reading from ffmpeg stdout", err)
			return
		}

		// Send received PCM to the sendPCM channel
		select {
		case send <- data:
		case <-close:
			return
		}
	}
}

func recoverBotLeftChannel(ctx *models.Context) *discordgo.VoiceConnection {
	channelID, err := checks.GetUserVoiceChannel(ctx)
	if err != nil {
		ctx.Send("User left the voice channel")
		return nil
	}
	v, err := ctx.Client.Session.ChannelVoiceJoin(ctx.GuildID, channelID, false, true)
	if err != nil {
		fmt.Println(err)
		ctx.Send("Error joining the voice channel")
		return nil
	}
	return v
}

func PlayDCAFile(v *discordgo.VoiceConnection, ctx *models.Context, songInfo *models.SongInfo, filename string, stop <-chan bool, skip <-chan bool) {

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening dca file :", err)
	}
	defer file.Close()

	// Send "playing" message to the channel
	ctx.Send("Playing: " + songInfo.Title + " - " + songInfo.Uploader)
	// Set status
	ctx.Client.Session.UpdateCustomStatus("Playing: " + songInfo.Title)
	ctx.Client.SetPlayingBool(true)

	// Send "speaking" packet over the voice websocket
	err = checks.BotInVoice(ctx)
	if err != nil {
		v = recoverBotLeftChannel(ctx) // This should only error when already not speaking
		if v == nil {
			return
		}
	}
	err = v.Speaking(false)
	if err != nil {
		log.Fatalf("Error while setting speaking: %s", err)
	}

	// Send not "speaking" packet over the websocket when we finish and start the cleanup
	defer func() {
		// Remove the 'Playing X' status
		err := checks.BotInVoice(ctx)
		if err != nil {
			v = recoverBotLeftChannel(ctx) // This should only error when the bot leaves pre-maturely
			if v == nil {
				return
			}
		}
		ctx.Client.Session.UpdateCustomStatus("")
		err = v.Speaking(false)
		if err != nil {
			log.Fatalf("Error while setting speaking: %s", err)
		}
	}()

	send := make(chan []byte, 200) // 200 frames can be buffered for sending
	defer close(send)

	sendCloseChannel := make(chan bool, 1)
	defer close(sendCloseChannel)

	closeChannel := make(chan bool, 1)
	go func() {
		SendDCA(v, send, sendCloseChannel)
		closeChannel <- true
	}()

	var opuslen int16

	// File reader
	buffer := make(chan []byte, 100) // 100 frames can be buffered from the file

	//when stop is sent, set stop bool to true
	go func() {
		signal := <-stop
		fmt.Println("[Music] Received signal")
		if signal {
			fmt.Println("[Music] Stop signal recognized")
			closeChannel <- true
			fmt.Println("[Music] Buffer closed")
		}
	}()

	//when skip is sent, do the cleanup process so the next song can be played and set the stop bool
	go func() {
		signal := <-skip
		fmt.Println("[Music] Received signal")
		if signal {
			fmt.Println("[Music] Skip signal recognized")
			closeChannel <- true
			fmt.Println("[Music] Buffer closed")
		}
	}()

	go func() {
		for {
			err := binary.Read(file, binary.LittleEndian, &opuslen)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				fmt.Println("[Music] Error reading frame length:", err)
				break
			}

			data := make([]byte, opuslen)
			err = binary.Read(file, binary.LittleEndian, &data)
			if err != nil {
				fmt.Println("[Music] Error reading frame data:", err)
				break
			}

			buffer <- data // Fill the buffer
		}
		close(buffer) // Signal end of stream
	}()

	for {
		select {
		case <-closeChannel:
			fmt.Println("[Music] Close signal recognized")
			// Stop streaming
			sendCloseChannel <- true
			fmt.Println("[Music] DCA Streaming stopped")
			startCleanupProcess(v, ctx, stop, skip)
			return
		case data, ok := <-buffer:
			if !ok {
				// DCA stream ended
				fmt.Println("[Music] DCA buffer empty/closed, ending stream")
				startCleanupProcess(v, ctx, stop, skip)

				return
			}
			select {
			case send <- data:
			case <-closeChannel:
				fmt.Println("[Music] Close signal recognized during send")
				// Stop streaming
				sendCloseChannel <- true
				fmt.Println("[Music] DCA Streaming stopped")
				startCleanupProcess(v, ctx, stop, skip)
				return
			}
		}
	}
}
