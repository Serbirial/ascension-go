package models

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
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

type LanaBot struct {
	Session *discordgo.Session

	StopChannel chan bool
	SongQueue   []*SongInfo

	IsPlaying bool
	Token     string
	Owners    []int
	Prefix    string
	Commands  map[string]Command
}

func (bot LanaBot) matchArgsToCommand(ctx *Context, argsRaw string) map[string]string {

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

func (bot LanaBot) ProcessMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
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
				ctx := &Context{bot, command, message.Author, argsraw, message.ChannelID, message.GuildID}

				// Execute all the checks
				for i := 0; i < len(command.Checks); i++ {
					err := command.Checks[i](ctx)
					if err != nil {
						ctx.Send(err.Error())
						fmt.Println("Check not passed")
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

func (bot LanaBot) AddCommands(commands map[string]Command) {
	for name, command := range commands {
		fmt.Println("Adding command: " + name)
		bot.Commands[name] = command
	}
}

//
//func (bot LanaBot) AddIntents(session *discordgo.Session) {
//	session.Identify.Intents = discordgo.IntentsGuildMessages
//}
