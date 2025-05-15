package models

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/net/websocket"
)

const (
	Intents = discordgo.IntentsDirectMessages |
		discordgo.IntentsGuildBans |
		discordgo.IntentsGuildEmojis |
		discordgo.IntentsGuildIntegrations |
		discordgo.IntentsGuildInvites |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildPresences |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsGuilds
)

type SongInfoWs struct {
	FilePath string `json:"FilePath"`
	Title    string `json:"Title"`
	Uploader string `json:"Uploader"`
	ID       string `json:"ID"`
}

type LanaBot struct {
	Session   *discordgo.Session
	WebSocket *websocket.Conn

	StopChannel chan bool
	SkipChannel chan bool
	SeekChannel chan int

	SongQueue []*SongInfo

	IsPlaying     bool
	IsDownloading bool
	Token         string
	Owners        []int
	Prefix        string
	Commands      map[string]Command
}

func (bot *LanaBot) ConnectToWS(url string, origin string) {
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		panic("CANT CONNECT TO WS SERVER: " + err.Error())
	}
	bot.WebSocket = ws
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From: me.Username,
		URL:  "",
		Stop: false,
		Seek: 0,
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

}

func (bot *LanaBot) SendDownloadToWS(url string) (*SongInfo, error) {
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From: me.Username,
		URL:  url,
		Stop: false,
		Seek: 0,
	}
	jsonDataSend, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("JSON Marshal error:", err)
	}
	err = websocket.Message.Send(bot.WebSocket, jsonDataSend)
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND URL!")
	}
	var jsonDataRecv []byte
	if err := websocket.Message.Receive(bot.WebSocket, &jsonDataRecv); err != nil {
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
func (bot *LanaBot) SendStopToWS() {
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From: me.Username,
		URL:  "",
		Stop: true,
		Seek: 0,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("JSON Marshal error:", err)
	}
	err = websocket.Message.Send(bot.WebSocket, jsonData)
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND URL!")
	}

}
func (bot *LanaBot) SendSeekToWS(seek int) {
	me, err := bot.Session.User("@me")
	if err != nil {
		panic("error getting self")
	}
	msg := Message{
		From: me.Username,
		URL:  "",
		Stop: false,
		Seek: seek,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("JSON Marshal error:", err)
	}
	err = websocket.Message.Send(bot.WebSocket, jsonData)
	if err != nil {
		log.Fatal("Error while establishing connection:", err)
		panic("CANT CONNECT TO WS! CANT SEND SEEK!")
	}

}

func (bot *LanaBot) AddToQueue(song *SongInfo) {
	bot.SongQueue = append(bot.SongQueue, song)
}
func (bot *LanaBot) SetQueue(queue []*SongInfo) {
	bot.SongQueue = queue
}

func (bot *LanaBot) SetPlayingBool(toSet bool) {
	bot.IsPlaying = toSet
}
func (bot *LanaBot) SetDownloadingBool(toSet bool) {
	bot.IsDownloading = toSet
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
