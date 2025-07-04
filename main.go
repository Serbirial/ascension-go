package main

// TODO:
// Make queue for downloads, which a goroutine reads from, downloads, and adds to queue all by itself to prevent any form of race horse

import (
	"encoding/json"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"ascension/commands"
	"ascension/error"
	"ascension/handlers"
	"ascension/models"
	"ascension/utils/fs"

	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/net/websocket"
)

// Define a struct to hold the CLI arguments
type CommandLineConfig struct {
	ConfigFile       string
	Standalone       bool
	BotTokenFilePath string
	BotPrefix        string

	UseDCA        bool
	WSOnly        bool
	StartProfiler bool

	DownloaderOnly           bool
	DetachedDownloaderServer bool
	RemoteDownloaderURL      string

	RemoteWSURL    string
	RemoteWSOrigin string

	SpotifyID     string
	SpotifySecret string
}

const spotifySecret = ""
const spotifyID = ""

var wsURL string = "ws://localhost:8182/ws"
var downloaderURL string = "localhost:8183/"

var wsOrigin string = "http://localhost/"

func parseFlags() CommandLineConfig {
	var cfg CommandLineConfig

	// Bind command-line flags to struct fields
	flag.StringVar(&cfg.ConfigFile, "config-file", "", "Path to a JSON file to load config values from (overrides other flags if present: see examples)")

	flag.BoolVar(&cfg.Standalone, "standalone", true, "Set to false if clustering across multiple devices, this will tell the bot to run its own WS/Downloader server internally.")

	flag.StringVar(&cfg.BotTokenFilePath, "token", "token.txt", "Path to txt file containing the token. Defaults to `token.txt`.")
	flag.StringVar(&cfg.BotPrefix, "prefix", "a!", "The prefix the bot uses for commands. Defaults to `a!`.")
	flag.BoolVar(&cfg.UseDCA, "useDCA", false, "Tells the bot to use DCA audio only (Bypasses usage of WS server)")
	flag.BoolVar(&cfg.WSOnly, "ws-only", false, "Tells the program only launch the WS server")
	flag.BoolVar(&cfg.DownloaderOnly, "downloader-only", false, "Tells the program only launch the Downloader/IO server")

	flag.BoolVar(&cfg.StartProfiler, "profiler", false, "Flag that enables the profiler")

	flag.StringVar(&cfg.RemoteWSURL, "ws-url", wsURL, "The URL the bot uses for connecting to WS. Defaults to `ws://localhost:8182/ws`.")
	flag.StringVar(&cfg.RemoteWSOrigin, "ws-origin", wsOrigin, "The Origin the bot uses for connecting to remote WS. Defaults to `http://localhost/`.")

	flag.BoolVar(&cfg.DetachedDownloaderServer, "remote-downloader", false, "Tells the bot/ws server to use a remote/detached downloader server. This will require knowledge of bridging device IO, The device running the bot/music server needs to be able to access files on the server running the downloader.")
	flag.StringVar(&cfg.RemoteDownloaderURL, "downloader-url", downloaderURL, "The URL the bot/ws server uses for connecting to a remote/detached downloader server. Defaults to `localhost:8183`.")

	flag.StringVar(&cfg.SpotifyID, "spotify-id", spotifyID, "The spotify client ID, required for spotify.")
	flag.StringVar(&cfg.SpotifySecret, "spotify-secret", spotifySecret, "Spotify client secret, required for spotify.")

	// Parse the flags
	log.Println("[CLI] Parsing arguments.")
	flag.Parse()
	if cfg.ConfigFile != "" {
		log.Println("[CLI] Loading config from JSON file:", cfg.ConfigFile)
		data, err := os.ReadFile(cfg.ConfigFile)
		if err != nil {
			log.Fatalf("[CLI] Failed to read config file: %v", err)
		}
		var jsonCfg CommandLineConfig
		if err := json.Unmarshal(data, &jsonCfg); err != nil {
			log.Fatalf("[CLI] Failed to parse config file: %v", err)
		}
		cfg = jsonCfg
	}
	log.Println("[CLI] Bot Token File: " + cfg.BotTokenFilePath)
	log.Println("[CLI] Spotify ID: " + cfg.SpotifyID)
	log.Println("[CLI] Spotify Secret: " + cfg.SpotifySecret)

	log.Println("[CLI] Bot Prefix: " + cfg.BotPrefix)
	log.Println("[CLI] Using DCA: " + strconv.FormatBool(cfg.UseDCA))
	log.Println("[CLI] WS Only: " + strconv.FormatBool(cfg.WSOnly))
	log.Println("[CLI] Remote WS URL: " + cfg.RemoteWSURL)
	log.Println("[CLI] Remote WS UROriginL: " + cfg.RemoteWSOrigin)
	log.Println("[CLI] Use detached IO/downloader server: " + strconv.FormatBool(cfg.DetachedDownloaderServer))

	return cfg
}

var config = parseFlags()

func startProfiler() {
	log.Println("[PROFILER] Starting pprof server at :6060")
	log.Println("[PROFILER]", http.ListenAndServe("0.0.0.0:6060", nil))
}

func startWS() { // FIXME: recognize usage of detached downloader, meaning the IO is also detached
	if config.DetachedDownloaderServer {
		handlers.DownloaderIsDetached = true // Let handler know that
		handlers.DownloaderURL = config.RemoteDownloaderURL
	}
	http.Handle("/ws", websocket.Handler(handlers.HandleWebSocket))

	log.Println("[WS] Server running on :8182")
	log.Fatal(http.ListenAndServe("0.0.0.0:8182", nil))
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func startDownloaderServer() {
	http.HandleFunc("/download", handlers.HandleDownloader)
	http.HandleFunc("/search", handlers.HandleDownloaderSearch)
	http.HandleFunc("/related", handlers.HandleDownloaderGetRelated)

	log.Println("[DOWNLOADER] Running on :8183")
	log.Fatal(http.ListenAndServe("0.0.0.0:8183", nil))
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

	var stopChannels = make(map[string]chan bool)
	var skipChannels = make(map[string]chan bool)
	var seekChannels = make(map[string]chan int)

	var isPlaying = make(map[string]bool)
	var isLooping = make(map[string]bool)
	var isDownloading = make(map[string]bool)
	var songQueues = make(map[string]*models.SongQueue)

	var websockets = make(map[string]*websocket.Conn)

	// Initialize download queue and start downloader loop
	downloadQueue := &models.DownloadQueue{
		SongQueues: songQueues,
		Queue:      []*models.DownloadQueueRequest{},
	}
	downloadQueue.StartDownloader()

	var Bot = models.Ascension{Session: session, Websockets: websockets, StopChannels: stopChannels, SkipChannels: skipChannels, SeekChannels: seekChannels, SongQueue: songQueues, DownloadQueue: downloadQueue, IsPlaying: isPlaying, IsLooping: isLooping, IsDownloading: isDownloading, Token: token, Owners: owners, Prefix: prefix, Commands: commandList, WsUrl: config.RemoteWSURL, WsOrigin: config.RemoteWSOrigin, DetachedDownloader: config.DetachedDownloaderServer, DownloaderUrl: config.RemoteDownloaderURL, SpotifyID: config.SpotifyID, SpotifySecret: config.SpotifySecret}
	Bot.AddCommands(commands.AllCommands)
	session.Identify.Intents = models.Intents

	session.AddHandler(Bot.ProcessMessage)

	log.Println("[BOT] Starting...")

	err = session.Open()
	error.ErrorCheckPanic(err)

	for _, guild := range session.State.Guilds {
		err := session.ChannelVoiceJoinManual(guild.ID, "", false, false) // empty channel ID = disconnect
		if err != nil {
			log.Printf("Error disconnecting from voice in guild %s: %v", guild.ID, err)
		}
	}

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

	if config.WSOnly { // Launch WS server if WS only
		go startWS()
	} else if config.DownloaderOnly { // Or launch the Downloader server if its Downloader only
		go startDownloaderServer()
	}

	if !config.WSOnly && !config.DownloaderOnly { // Dont launch the bot if in WS/Downloader server mode
		go startBot()
		if config.Standalone {
			go startWS()
			go startDownloaderServer()
			log.Println("[INFO] Running internal WS/Downloader servers.")
		} else if !config.Standalone {
			log.Println("[CRITICAL] ATTENTION! YOU WILL NEED TO RUN THE MUSIC SERVER ALONGSIDE THE BOT!")

		}

	}
	if config.StartProfiler { // Launch the profiler if enabled
		go startProfiler()
	}
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("[MAIN] Waiting for exit signal.")
	<-sc
}
