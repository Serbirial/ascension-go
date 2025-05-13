package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gobot/commands"
	"gobot/error"
	"gobot/models"
	"gobot/utils/fs"

	"github.com/bwmarrin/discordgo"
)

// Define a struct to hold the CLI arguments
type CommandLineConfig struct {
	BotTokenFilePath string
	BotPrefix        string
}

func parseFlags() CommandLineConfig {
	var cfg CommandLineConfig

	// Bind command-line flags to struct fields
	flag.StringVar(&cfg.BotTokenFilePath, "token", "token.txt", "Path to txt file containing the token. Defaults to `token.txt`.")
	flag.StringVar(&cfg.BotTokenFilePath, "prefix", "a!", "The prefix the bot uses for commands. Defaults to `a!`.")

	// Parse the flags
	log.Println("[CLI] Parsing arguments.")
	flag.Parse()
	log.Println("[CLI] Bot Token File: " + cfg.BotTokenFilePath)
	log.Println("[CLI] Bot Prefix: " + cfg.BotPrefix)

	return cfg
}

var config = parseFlags()

func startBot() {
	var prefix = config.BotPrefix
	var owners = make([]int, 0)
	var token = fs.ReadFileWhole(config.BotTokenFilePath)
	var commandList = make(map[string]models.Command)

	// TODO Un-Hardcode this
	owners = append(owners, 1270040138948411442)
	//

	session, err := discordgo.New("Bot " + token)
	error.ErrorCheckPanic(err)

	var stopChannel = make(chan bool)
	var skipChannel = make(chan bool)

	var Bot = models.LanaBot{Session: session, StopChannel: stopChannel, SkipChannel: skipChannel, Token: token, Owners: owners, Prefix: prefix, Commands: commandList}
	Bot.AddCommands(commands.AllCommands)
	session.Identify.Intents = models.Intents

	session.AddHandler(Bot.ProcessMessage)

	log.Println("[BOT] Starting...")

	err = session.Open()
	error.ErrorCheckPanic(err)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("[BOT] Started.")
	<-sc

	// Cleanly close down the Discord session.
	log.Println("[BOT] Closing...")
	session.Close()
	log.Println("[BOT] Closed.")
}

func main() {
	go startBot()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("[MAIN] Waiting for exit signal.")
	<-sc
}
