package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gobot/commands"
	"gobot/error"
	"gobot/models"
	"gobot/utils/fs"

	"github.com/bwmarrin/discordgo"
)

func main() {
	const prefix = "a!"
	var owners = make([]int, 0)
	var token = fs.ReadFileWhole("token.txt")
	var commandList = make(map[string]models.Command)

	// TODO Un-Hardcode this
	owners = append(owners, 1270040138948411442)
	//

	session, err := discordgo.New("Bot " + token)
	error.ErrorCheckPanic(err)

	var songQueue []*models.SongInfo
	var isPlaying bool = false

	var Bot = models.LanaBot{Session: session, StopChannel: make(chan bool), SongQueue: songQueue, IsPlaying: isPlaying, Token: token, Owners: owners, Prefix: prefix, Commands: commandList}
	Bot.AddCommands(commands.AllCommands)
	session.Identify.Intents = models.Intents

	session.AddHandler(Bot.ProcessMessage)

	fmt.Println("Starting bot...")

	err = session.Open()
	error.ErrorCheckPanic(err)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	fmt.Println("Bot closing...")
	session.Close()
	fmt.Println("Bot closed.")

}
