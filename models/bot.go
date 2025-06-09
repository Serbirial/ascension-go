package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/net/websocket"
)

const (
	Intents = discordgo.IntentsGuilds |

		//discordgo.IntentsDirectMessages |
		// discordgo.IntentsGuildBans |
		// discordgo.IntentsGuildEmojis |
		// discordgo.IntentsGuildIntegrations |
		// discordgo.IntentsGuildInvites |
		// discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildMessages |
		// discordgo.IntentsGuildPresences |
		discordgo.IntentsGuildVoiceStates
)

type SongInfoWs struct {
	FilePath string `json:"FilePath"`
	Title    string `json:"Title"`
	Uploader string `json:"Uploader"`
	ID       string `json:"ID"`
}

type Ascension struct {
	Session    *discordgo.Session
	Websockets map[string]*websocket.Conn

	StopChannels map[string]chan bool
	SkipChannels map[string]chan bool
	SeekChannels map[string]chan int

	SongQueue     map[string]*SongQueue
	DownloadQueue *DownloadQueue

	IsPlaying     map[string]bool
	IsLooping     map[string]bool
	IsDownloading map[string]bool
	Token         string
	Owners        []int
	Prefix        string
	Commands      map[string]Command

	DetachedDownloader bool
	DownloaderUrl      string
	WsUrl              string
	WsOrigin           string

	SpotifyID     string
	SpotifySecret string
}

// Useful when when clustering, will need to bridge IO of device running the bot and/or music server depending on setup.
func (bot *Ascension) SendDownloadDetached(url string) (*SongInfo, error) {
	type DownloadRequest struct {
		URL string `json:"url"`
	}

	reqBody := DownloadRequest{URL: url}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to marshal request JSON:", err)
		return nil, err
	}

	resp, err := http.Post(bot.DownloaderUrl+"/download", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("[DETACHED-DOWnLOADER] Failed to POST to detached downloader server:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[DETACHED-DOWNLOADER] Download server responded with status: %d\n", resp.StatusCode)
		return nil, errors.New("download server returned non-OK status")
	}

	var song SongInfo
	if err := json.NewDecoder(resp.Body).Decode(&song); err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to decode SongInfo from response:", err)
		return nil, err
	}

	log.Printf("[DETACHED-DOWNLOADER] Successfully downloaded and received info: %s\n", song.Title)
	return &song, nil
}

// SendSearchRequest sends a search query to the detached downloader and returns the YouTube video ID.
func (bot *Ascension) SendSearchRequest(query string) (string, error) {
	type SearchRequest struct {
		Query string `json:"query"`
	}
	reqBody := SearchRequest{Query: query}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to marshal search request JSON:", err)
		return "", err
	}

	resp, err := http.Post(bot.DownloaderUrl+"/search", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to POST search request:", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[DETACHED-DOWNLOADER] Search server responded with status: %d\n", resp.StatusCode)
		return "", errors.New("Download server returned non-OK status")
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to decode search response:", err)
		return "", err
	}

	log.Printf("[DETACHED-DOWNLOADER] Search successful, got video ID: %s\n", result.ID)
	return result.ID, nil
}

// SendGetRelatedRequest sends a related videos request to the detached downloader and returns a slice of video URLs.
func (bot *Ascension) SendGetRelatedRequest(videoID string, limit int) ([]string, error) {
	type RelatedRequest struct {
		ID    string `json:"id"`
		Limit int    `json:"limit"`
	}
	reqBody := RelatedRequest{
		ID:    videoID,
		Limit: limit,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to marshal related request JSON:", err)
		return nil, err
	}

	resp, err := http.Post(bot.DownloaderUrl+"/related", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to POST related request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[DETACHED-DOWNLOADER] Download server responded with status: %d\n", resp.StatusCode)
		return nil, errors.New("download server returned non-OK status")
	}

	var related []string
	if err := json.NewDecoder(resp.Body).Decode(&related); err != nil {
		log.Println("[DETACHED-DOWNLOADER] Failed to decode related response:", err)
		return nil, err
	}

	log.Printf("[DETACHED-DOWNLOADER] Retrieved %d related videos\n", len(related))
	return related, nil
}

func (bot *Ascension) ConnectToWS(url string, origin string, identifier string) *websocket.Conn {
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		panic("CANT CONNECT TO WS SERVER: " + err.Error())
	}
	bot.Websockets[identifier] = ws
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From:       me.Username,
		Identifier: identifier,
		URL:        "",
		Stop:       false,
		Seek:       -1,
		Download:   false,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("JSON Marshal error:", err)
	}
	err = websocket.Message.Send(ws, jsonData)
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND NAME!")
	}
	return ws
}
func (bot *Ascension) CreateTempWS(url string, origin string, identifier string) *websocket.Conn {
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		panic("CANT CONNECT TO WS SERVER: " + err.Error())
	}
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From:       me.Username,
		URL:        "",
		Stop:       false,
		Seek:       -1,
		Download:   false,
		Identifier: identifier,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("JSON Marshal error:", err)
	}
	err = websocket.Message.Send(ws, jsonData)
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND NAME!")
	}
	return ws

}

func (bot *Ascension) SendDownloadToWS(url string, identifier string) (*SongInfo, error) {

	ws := bot.CreateTempWS(bot.WsUrl, bot.WsOrigin, identifier) // Create a new WS connection for communicating with the server
	defer ws.Close()
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From:       me.Username,
		URL:        url,
		Stop:       false,
		Seek:       -1,
		Download:   true,
		Identifier: identifier,
	}
	jsonDataSend, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("JSON Marshal error:", err)
	}
	err = websocket.Message.Send(ws, jsonDataSend)
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND URL!")
	}
	var jsonDataRecv []byte
	if err := websocket.Message.Receive(ws, &jsonDataRecv); err != nil {
		log.Fatalf("Failed to receive: %v", err)
		return nil, err
	}

	var song SongInfo
	if err := json.Unmarshal(jsonDataRecv, &song); err != nil {
		log.Fatalf("Failed to decode JSON: %v", err)
		return nil, err
	}
	return &song, nil

}

func (bot *Ascension) SendPlayToWS(url string, identifier string) (*SongInfo, error) {
	ws := bot.CreateTempWS(bot.WsUrl, bot.WsOrigin, identifier) // Create a new WS connection for communicating with the server
	defer ws.Close()
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From:       me.Username,
		URL:        url,
		Stop:       false,
		Seek:       -1,
		Download:   false,
		Identifier: identifier,
	}
	jsonDataSend, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("JSON Marshal error:", err)
	}
	err = websocket.Message.Send(ws, jsonDataSend)
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND URL!")
	}
	var jsonDataRecv []byte
	if err := websocket.Message.Receive(ws, &jsonDataRecv); err != nil {
		log.Fatalf("Failed to receive: %v", err)
		return nil, err
	}

	var song SongInfo
	if err := json.Unmarshal(jsonDataRecv, &song); err != nil {
		log.Fatalf("Failed to decode JSON: %v", err)
		return nil, err
	}
	return &song, nil

}

func (bot *Ascension) SendStopToWS(identifier string) {
	ws := bot.CreateTempWS(bot.WsUrl, bot.WsOrigin, identifier) // Create a new WS connection for communicating with the server
	defer ws.Close()
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From:       me.Username,
		URL:        "",
		Stop:       true,
		Seek:       -1,
		Download:   false,
		Identifier: identifier,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("JSON Marshal error:", err)
	}
	err = websocket.Message.Send(ws, jsonData)
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND STOP!")
	}
}

func (bot *Ascension) SendSeekToWS(seek int, identifier string) {
	ws := bot.CreateTempWS(bot.WsUrl, bot.WsOrigin, identifier) // Create a new WS connection for communicating with the server
	defer ws.Close()
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From:       me.Username,
		URL:        "",
		Stop:       false,
		Seek:       seek,
		Download:   false,
		Identifier: identifier,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("JSON Marshal error:", err)
	}
	err = websocket.Message.Send(ws, jsonData)
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND SEEK!")
	}
}

func (bot *Ascension) SendDONEToWS(identifier string) {
	ws := bot.Websockets[identifier]

	err := websocket.Message.Send(ws, []byte("DONE")) // Send DONE so the bot knows everything is OK and DONE
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND DONE!")
	}
}
func (bot *Ascension) CloseWebsocket(identifier string) {
	ws := bot.Websockets[identifier]
	ws.Close()

}

func (bot *Ascension) AddToQueue(guildID string, song *SongInfo) {
	bot.SongQueue[guildID].Add(song)
}

func (bot *Ascension) SetPlayingBool(guildID string, toSet bool) {
	bot.IsPlaying[guildID] = toSet
}
func (bot *Ascension) SetDownloadingBool(guildID string, toSet bool) {
	bot.IsDownloading[guildID] = toSet
}
func (bot *Ascension) SetLoopingBool(guildID string, toSet bool) {
	bot.SongQueue[guildID].Loop = toSet
}

func (bot *Ascension) matchArgsToCommand(ctx *Context, argsRaw string) map[string]string {

	// Maybe use SplitAfterN
	var argsSplit = strings.SplitN(argsRaw, " ", len(ctx.CurrentCommand.Args)+1)
	var args map[string]string = make(map[string]string)

	if len(argsSplit) == 1 { // There are no args
		return args
	}

	var i int = 1

	for argName := range ctx.CurrentCommand.Args {
		args[argName] = argsSplit[i]
		i++
	}

	return args
}

func (bot *Ascension) ProcessMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID {
		return
	}

	if strings.HasPrefix(message.Content, bot.Prefix) {
		var listOfWords = strings.Fields(message.Content)
		var possibleCommandString, ok = strings.CutPrefix(listOfWords[0], bot.Prefix)
		if ok {
			command, exists := bot.Commands[possibleCommandString]
			if exists {
				var _, argsraw, _ = strings.Cut(message.Content, possibleCommandString)
				// Create the context
				botPointer := &bot
				ctx := &Context{*botPointer, command, message.Author, argsraw, message.ChannelID, message.GuildID}

				// Execute all the checks
				for i := 0; i < len(command.Checks); i++ {
					err := command.Checks[i](ctx)
					if err != nil {
						ctx.Send(err.Error())
						log.Fatalln("[Bot] Command Check not passed")
						return
					}
				}
				// TODO Check if the channel and command is NSFW
				argsProcessed := bot.matchArgsToCommand(ctx, argsraw)
				command.Callback(ctx, argsProcessed)
			}
		}
	}
}

func (bot *Ascension) AddCommands(commands map[string]Command) {
	for name, command := range commands {
		log.Println("[BOT] Adding command: " + name)
		bot.Commands[name] = command
	}
}

//
//func (bot Ascension) AddIntents(session *discordgo.Session) {
//	session.Identify.Intents = discordgo.IntentsGuildMessages
//}
