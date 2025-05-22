package models

import (
	"encoding/json"
	"log"
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

type LanaBot struct {
	Session    *discordgo.Session
	Websockets map[string]*websocket.Conn

	StopChannels map[string]chan bool
	SkipChannels map[string]chan bool
	SeekChannels map[string]chan int

	SongQueue map[string][]*SongInfo

	IsPlaying     map[string]bool
	IsDownloading map[string]bool
	Token         string
	Owners        []int
	Prefix        string
	Commands      map[string]Command

	WsUrl    string
	WsOrigin string
}

func (bot *LanaBot) ConnectToWS(url string, origin string, identifier string) *websocket.Conn {
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
func (bot *LanaBot) CreateTempWS(url string, origin string, identifier string) *websocket.Conn {
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

func (bot *LanaBot) SendDownloadToWS(url string, identifier string) (*SongInfo, error) {
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

func (bot *LanaBot) SendPlayToWS(url string, identifier string) (*SongInfo, error) {
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

func (bot *LanaBot) SendStopToWS(identifier string) {
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

func (bot *LanaBot) SendSeekToWS(seek int, identifier string) {
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

func (bot *LanaBot) SendDONEToWS(identifier string) {
	ws := bot.Websockets[identifier]

	err := websocket.Message.Send(ws, []byte("DONE")) // Send DONE so the bot knows everything is OK and DONE
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND DONE!")
	}
}
func (bot *LanaBot) CloseWebsocket(identifier string) {
	ws := bot.Websockets[identifier]
	ws.Close()

}

func (bot *LanaBot) AddToQueue(guildID string, song *SongInfo) {
	bot.SongQueue[guildID] = append(bot.SongQueue[guildID], song)
}
func (bot *LanaBot) SetQueue(guildID string, queue []*SongInfo) {
	bot.SongQueue[guildID] = queue
}

func (bot *LanaBot) SetPlayingBool(guildID string, toSet bool) {
	bot.IsPlaying[guildID] = toSet
}
func (bot *LanaBot) SetDownloadingBool(guildID string, toSet bool) {
	bot.IsDownloading[guildID] = toSet
}

func (bot *LanaBot) matchArgsToCommand(ctx *Context, argsRaw string) map[string]string {

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

func (bot *LanaBot) ProcessMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
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

func (bot *LanaBot) AddCommands(commands map[string]Command) {
	for name, command := range commands {
		log.Println("[BOT] Adding command: " + name)
		bot.Commands[name] = command
	}
}

//
//func (bot LanaBot) AddIntents(session *discordgo.Session) {
//	session.Identify.Intents = discordgo.IntentsGuildMessages
//}
