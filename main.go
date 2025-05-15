package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"gobot/commands"
	"gobot/error"
	"gobot/handlers"
	"gobot/models"
	"gobot/utils/fs"

	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/net/websocket"
)

// Define a struct to hold the CLI arguments
type CommandLineConfig struct {
	BotTokenFilePath string
	BotPrefix        string

	UseDCA bool
}

func parseFlags() CommandLineConfig {
	var cfg CommandLineConfig

	// Bind command-line flags to struct fields
	flag.StringVar(&cfg.BotTokenFilePath, "token", "token.txt", "Path to txt file containing the token. Defaults to `token.txt`.")
	flag.StringVar(&cfg.BotPrefix, "prefix", "a!", "The prefix the bot uses for commands. Defaults to `a!`.")
	flag.BoolVar(&cfg.UseDCA, "useDCA", false, "Tells the bot to use DCA audio only (WILL BYPASS USING EXTERNAL SERVER)")

	// Parse the flags
	log.Println("[CLI] Parsing arguments.")
	flag.Parse()
	log.Println("[CLI] Bot Token File: " + cfg.BotTokenFilePath)
	log.Println("[CLI] Bot Prefix: " + cfg.BotPrefix)
	log.Println("[CLI] Using DCA: " + strconv.FormatBool(cfg.UseDCA))

	return cfg
}

var config = parseFlags()

func startProfiler() {
	log.Println("[PROFILER] Starting pprof server at :6060")
	log.Println("[PROFILER]", http.ListenAndServe("0.0.0.0:6060", nil))
}

func startWS() {
	http.Handle("/ws", websocket.Handler(handlers.HandleWebSocket))

	fmt.Println("WebSocket server running on :8182")
	log.Fatal(http.ListenAndServe("localhost:8182", nil))
}

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
	var seekChannel = make(chan int)

	var Bot = models.LanaBot{Session: session, StopChannel: stopChannel, SkipChannel: skipChannel, SeekChannel: seekChannel, Token: token, Owners: owners, Prefix: prefix, Commands: commandList}
	Bot.AddCommands(commands.AllCommands)
	session.Identify.Intents = models.Intents

	session.AddHandler(Bot.ProcessMessage)

	log.Println("[BOT] Starting...")

	err = session.Open()
	error.ErrorCheckPanic(err)

	//Bot.ConnectToWS("ws://localhost:8182/ws", "http://localhost/")

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
	go startProfiler()
	//go startWS()
	go startBot()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("[MAIN] Waiting for exit signal.")
	<-sc
}
