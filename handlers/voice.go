/*******************************************************************************
 * This is very experimental code and probably a long way from perfect or
 * ideal.  Please provide feed back on areas that would improve performance
 *
 */

// Package dgvoice provides opus encoding and audio file playback for the
// Discordgo package.
package handlers

import (
	"ascension/models"
	"ascension/utils/arrays"
	"ascension/utils/checks"
	"bufio"
	"encoding/binary"
	"fmt"
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

	frameDuration time.Duration = 20 * time.Millisecond // DCA frame size

)

var (
	speakers    map[uint32]*gopus.Decoder
	opusEncoder *gopus.Encoder
	mu          sync.Mutex
)

// OnError gets called by dgvoice when an error is encountered.
// By default logs to STDERR
var OnError = func(prefix string, str string, err error) {
	prefix = prefix + ": " + str

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
		OnError("[Music]", "NewEncoder Error", err)
		return
	}

	for {

		// read pcm from chan, exit if channel is closed.
		recv, ok := <-pcm
		if !ok {
			OnError("[Music]", "PCM Channel closed", nil)
			return
		}
		// try encoding pcm frame with Opus
		opus, err := opusEncoder.Encode(recv, frameSize, maxBytes)
		if err != nil {
			OnError("[Music]", "Encoding Error", err)
			return
		}

		if v.Ready == false || v.OpusSend == nil {
			// OnError("[Music]",fmt.Sprintf("Discordgo not ready for opus packets. %+v : %+v", v.Ready, v.OpusSend), nil)
			// Sending errors here might not be suited
			return
		}
		// send encoded opus data to the sendOpus channel
		v.OpusSend <- opus
	}
}

// SendDCA will receive on the provied channel then send that to Discordgo
func SendDCA(v *discordgo.VoiceConnection, dca <-chan []byte) {
	if dca == nil {
		return
	}

	for {
		dcaData, ok := <-dca
		// read dca from chan, exit if channel is closed.
		if !ok {
			log.Println("[DCA] Channel closed")
			return
		}
		if v.Ready == false || v.OpusSend == nil {
			// fmt.Println(fmt.Sprintf("Discordgo not ready for opus packets. %+v : %+v", v.Ready, v.OpusSend), nil)
			// Sending errors here might not be suited
			return
		}
		// send encoded opus data to the sendOpus channel
		v.OpusSend <- dcaData
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
			OnError("[Music]", fmt.Sprintf("Discordgo not to receive opus packets. %+v : %+v", v.Ready, v.OpusSend), nil)
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
				OnError("[Music]", "error creating opus decoder", err)
				continue
			}
		}

		p.PCM, err = speakers[p.SSRC].Decode(p.Opus, 960, false)
		if err != nil {
			OnError("[Music]", "Error decoding opus data", err)
			continue
		}

		c <- p
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
		log.Println(err)
		ctx.Send("Error joining the voice channel")
		return nil
	}
	return v
}

// Helper: build frame index offsets
func buildFrameIndex(file *os.File) ([]int64, error) {
	var offsets []int64
	var frameLen int16

	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	for {
		pos, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			break
		}
		offsets = append(offsets, pos)

		err = binary.Read(file, binary.LittleEndian, &frameLen)
		if err != nil {
			break
		}

		_, err = file.Seek(int64(frameLen), io.SeekCurrent)
		if err != nil {
			break
		}
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	return offsets, nil
}

// This plays the next song in the queue
func playNextSongInQueue(v *discordgo.VoiceConnection, ctx *models.Context, stop <-chan bool, skip <-chan bool, seek <-chan int) {
	if len(ctx.Client.SongQueue) >= 1 {
		// Get first SongInfo in Queue and play it
		fmt.Println(ctx.Client.SongQueue)
		fmt.Println(ctx.Client.SongQueue[0])

		var song *models.SongInfo = ctx.Client.SongQueue[0]
		PlayFromWS(v, ctx, song, stop, skip, seek)
	}
}

func startCleanupProcess(v *discordgo.VoiceConnection, ctx *models.Context, stop <-chan bool, skip <-chan bool, seek <-chan int) {
	log.Println("[Music] Cleanup process started")
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
		log.Println("[Music] Queue is not empty, playing next song")
		// FIXME
		// Give the bot 2 seconds to prevent audio overlap
		time.Sleep(2 * time.Second)
		// Play the next song
		playNextSongInQueue(v, ctx, stop, skip, seek)
	} else if len(ctx.Client.SongQueue) == 0 { // Queue was empty
		log.Println("[Music] Queue is empty, waiting for activity")
		// Wait 60s to see if activity happens
		var tries int = 0
		for {
			if tries >= 300 { // 300s
				break
			}
			time.Sleep(1 * time.Second)
			if len(ctx.Client.SongQueue) >= 1 { // queue is no longer empty
				log.Println("[Music] Activity in queue")
				break
			}
			tries++
		}
		if len(ctx.Client.SongQueue) == 0 { // Disconnect after the 300s if the queue is still empty
			if !ctx.Client.IsDownloading { // Only disconnect if not currently downloading
				log.Println("[Music] Disconnecting because no activity and empty queue")
				v.Disconnect()
				return
			}
		}
	}
}

func clearStatusAndRemoveCurrentSongFromQueue(ctx *models.Context) {
	ctx.Client.Session.UpdateCustomStatus("")
	temp := arrays.RemoveFirstSong(ctx.Client.SongQueue)
	ctx.Client.SetQueue(temp)
}

// PlayAudioFile will play the given filename to the already connected
// Discord voice server/channel.  voice websocket and udp socket
// must already be setup before this will work.
func PlayAudioFile(v *discordgo.VoiceConnection, ctx *models.Context, songInfo *models.SongInfo, filename string, stop <-chan bool, skip <-chan bool, seek <-chan int) {
	// Send "playing" message to the channel
	ctx.Send("Playing: " + songInfo.Title + " - " + songInfo.Uploader)
	// Set status
	ctx.Client.Session.UpdateCustomStatus("Playing: " + songInfo.Title)
	ctx.Client.SetPlayingBool(true)

	// Create a shell command "object" to run.
	run := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	ffmpegout, err := run.StdoutPipe()
	if err != nil {
		OnError("[Music]", "StdoutPipe Error", err)
		return
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384*20)

	// Starts the ffmpeg command
	err = run.Start()
	if err != nil {
		OnError("[Music]", "RunStart Error", err)
		return
	}

	// prevent memory leak from residual ffmpeg streams
	defer run.Process.Kill()

	//when stop is sent, kill ffmpeg
	go func() {
		signal := <-stop
		log.Println("[Music] Received signal")
		if signal == true {
			// Remove the 'Playing X' status
			ctx.Client.Session.UpdateCustomStatus("")
			log.Println("[Music] Stop signal sent")
			// Remove current song from queue
			temp := arrays.RemoveFirstSong(ctx.Client.SongQueue)
			// Replace queue with updated one
			ctx.Client.SetQueue(temp)
			// Kill ffmpeg
			err = run.Process.Kill()
			log.Println("[Music] FFMPEG killed")
		}

	}()

	//when skip is sent, do the cleanup process so the next song can be played
	go func() {
		signal := <-skip
		log.Println("[Music] Received signal")
		if signal == true {
			// Remove the 'Playing X' status
			ctx.Client.Session.UpdateCustomStatus("")
			log.Println("[Music] Skip signal sent")
			err = run.Process.Kill()
			log.Println("[Music] FFMPEG killed")
			startCleanupProcess(v, ctx, stop, skip, seek)
		}
	}()

	// Send "speaking" packet over the voice websocket
	err = v.Speaking(true)
	if err != nil {
		OnError("[Music]", "Couldn't set speaking", err)
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

		startCleanupProcess(v, ctx, stop, skip, seek)
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
			OnError("[Music]", "error reading from ffmpeg stdout", err)
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

func PlayDCAFile(v *discordgo.VoiceConnection, ctx *models.Context, songInfo *models.SongInfo, filename string, stop <-chan bool, skip <-chan bool, seek <-chan int) {

	file, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening dca file :", err)
		return
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
	err = v.Speaking(true)
	if err != nil {
		log.Fatalf("Error while setting speaking: %s", err)
	}

	// Send not "speaking" packet over the websocket when we finish and start the cleanup
	defer func() {
		// Remove the 'Playing X' status
		err := checks.BotInVoice(ctx)
		if err != nil {
			return // Bot already left
		}
		ctx.Client.Session.UpdateCustomStatus("")
		err = v.Speaking(false)
		if err != nil {
			log.Fatalf("Error while setting speaking: %s", err)
		}
	}()

	send := make(chan []byte, 20) // 20 frames can be buffered for sending
	// setting the buffer too high for `send` MIGHT cause audio overlap when playing the next song in queue
	defer close(send)

	closeChannel := make(chan bool, 1)
	go func() {
		SendDCA(v, send)
		// Code is not needed, once `send` is closed it will recognize that and close itself.
		//closeChannel <- true
	}()
	defer close(closeChannel)

	// File reader
	buffer := make(chan []byte, 200) // 200 frames can be buffered from the file

	// Handle stop and skip signals
	go func() {
		select {
		case signal, ok := <-stop:
			if ok && signal {
				closeChannel <- true
			}
		case signal, ok := <-skip:
			if ok && signal {
				closeChannel <- true
			}
		}
	}()

	const frameRateDCA = int(time.Second / frameDuration) // 50 frames per second
	var (
		currentFrame int = 0
		smu          sync.Mutex
	)

	frameIndex, err := buildFrameIndex(file)
	if err != nil {
		log.Println("[Music] Error building frame index:", err)
		return
	}
	if len(frameIndex) == 0 {
		log.Println("[Music] Frame index empty, cannot play")
		return
	}

	// Frame reader goroutine
	go func() {
		defer close(buffer)

		for {
			select {
			case seconds := <-seek:
				// Drain buffer using labeled block
			drain:
				for {
					select {
					case <-buffer:
						// drain element
					default:
						break drain // exit draining loop
					}
				}
				smu.Lock()
				frameDelta := int(seconds * frameRateDCA)
				targetFrame := currentFrame + frameDelta
				if targetFrame < 0 {
					targetFrame = 0
				}
				if targetFrame >= len(frameIndex) {
					targetFrame = len(frameIndex) - 1
				}

				_, err := file.Seek(frameIndex[targetFrame], io.SeekStart)
				if err != nil {
					log.Println("[Music] Seek error:", err)
					smu.Unlock()
					return
				}
				currentFrame = targetFrame
				smu.Unlock()

			default:
				// Continue reading current frame
				smu.Lock()
				pos, _ := file.Seek(0, io.SeekCurrent)
				_ = pos
				smu.Unlock()

				var opuslen int16
				err := binary.Read(file, binary.LittleEndian, &opuslen)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return // end of file
				}
				if err != nil {
					log.Println("[Music] Error reading frame length:", err)
					return
				}

				data := make([]byte, opuslen)
				err = binary.Read(file, binary.LittleEndian, &data)
				if err != nil {
					log.Println("[Music] Error reading frame data:", err)
					return
				}

				buffer <- data
				smu.Lock()
				currentFrame++
				smu.Unlock()
			}
		}
	}()

	for {
		select {
		case <-closeChannel:
			log.Println("[Music] Close signal recognized")
			// Stop streaming
			close(send)
			log.Println("[Music] DCA Streaming stopped")
			startCleanupProcess(v, ctx, stop, skip, seek)
			return
		case data, ok := <-buffer:
			if !ok {
				// DCA stream ended
				log.Println("[Music] DCA buffer empty, ending stream")

				startCleanupProcess(v, ctx, stop, skip, seek)
				return
			}
			select {
			case send <- data:
			case <-closeChannel:
				log.Println("[Music] Close signal recognized during send")
				// Stop streaming
				log.Println("[Music] DCA Stream channel closed")
				startCleanupProcess(v, ctx, stop, skip, seek)
				return
			}
		}
	}
}

// TODO Play audio from remote server through WS
func PlayFromWS(v *discordgo.VoiceConnection, ctx *models.Context, songInfo *models.SongInfo, stop <-chan bool, skip <-chan bool, seek <-chan int) {

	// Send "playing" message to the channel
	ctx.Send("Playing: " + songInfo.Title + " - " + songInfo.Uploader)
	// Set status
	ctx.Client.Session.UpdateCustomStatus("Playing: " + songInfo.Title)
	ctx.Client.SetPlayingBool(true)

	// Send "speaking" packet over the voice websocket
	err := checks.BotInVoice(ctx)
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

	send := make(chan []byte, 20) // 20 frames can be buffered for sending
	// setting the buffer too high for `send` MIGHT cause audio overlap when playing the next song in queue
	defer close(send)

	closeChannel := make(chan bool, 1)
	go func() {
		SendDCA(v, send)
		closeChannel <- true
	}()
	defer close(closeChannel)

	wsBuffer := make(chan []byte, 60)                       // 60 frames can be buffered from WS
	defer close(wsBuffer)                                   // Close buffer
	wsStop := make(chan bool, 1)                            // Signal for quitting the WS receiver
	defer close(wsStop)                                     // Close WS stop
	defer func() { wsStop <- true }()                       // Stop the WS receiver once done
	go RecvByteData(ctx.Client.WebSocket, wsBuffer, wsStop) // Start receiving PCM data from WS

	// Handle stop and skip signals
	go func() {
		select {
		case signal, ok := <-stop:
			if ok && signal {
				closeChannel <- true
			}
		case signal, ok := <-skip:
			if ok && signal {
				closeChannel <- true
			}
		case seekNum, ok := <-seek:
			if ok {
			drain:
				for {
					select {
					case <-wsBuffer:
						// drain element
					default:
						break drain // exit draining loop
					}
				}
				ctx.Client.SendSeekToWS(seekNum)
			}
		}
	}()

	for {
		select {
		case <-closeChannel:
			log.Println("[Music] Close signal recognized")
			// Stop WS/Stream
			close(send)
			wsStop <- true
			log.Println("[Music] WS recv stopped")
			log.Println("[Music] Sending stop to WS server")
			ctx.Client.SendStopToWS()
			log.Println("[Music] Sent stop to WS server")

			startCleanupProcess(v, ctx, stop, skip, seek)
			return
		case data, ok := <-wsBuffer:
			if !ok {
				// WS stream channel closed
				wsStop <- true
				log.Println("[Music] WS buffer closed, ending stream")
				startCleanupProcess(v, ctx, stop, skip, seek)
				return
			} else if string(data) == "EOF" {
				// WS sent EOF, stop recv and start cleanup
				wsStop <- true
				log.Println("[Music] WS sent EOF, ending stream")
				startCleanupProcess(v, ctx, stop, skip, seek)
				return
			}
			select {
			case send <- data:
			case <-closeChannel:
				log.Println("[Music] Close signal recognized during send")
				// Stop WS/Stream
				close(send)
				wsStop <- true
				log.Println("[Music] WS recv stopped")
				log.Println("[Music] Sending stop to WS server")
				ctx.Client.SendStopToWS()
				log.Println("[Music] Sent stop to WS server")
				log.Println("[Music] WS Streaming stopped")
				startCleanupProcess(v, ctx, stop, skip, seek)
				return
			}
		}
	}
}
