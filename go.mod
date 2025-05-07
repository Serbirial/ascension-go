module gobot

go 1.24.1

require (
	github.com/bwmarrin/discordgo v0.28.1
	// comment this out for my own if needed
	github.com/wader/goutubedl v0.0.0-20250501160909-e491034be88d
	layeh.com/gopus v0.0.0-20210501142526-1ee02d434e32
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/serbirial/goutubedl v0.0.0-20250507180029-8a1ebcd5bcc3 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/wader/goutubedl v0.0.0-20250501160909-e491034be88d => github.com/serbirial/goutubedl v0.0.0-20250507165209-6776a45806ed
