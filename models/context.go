package models

import (
	"ascension/error"

	"github.com/bwmarrin/discordgo"
)

type Context struct {
	Client         *Ascension
	CurrentCommand Command
	Author         *discordgo.User
	ArgsRaw        string
	ChannelID      string
	GuildID        string
}

func (ctx Context) Send(content string) *discordgo.Message {

	message, err := ctx.Client.Session.ChannelMessageSend(ctx.ChannelID, content)
	error.ErrorCheckPanic(err)
	return message
}

func (ctx Context) SendEmbed(embed *discordgo.MessageEmbed) *discordgo.Message {
	message, err := ctx.Client.Session.ChannelMessageSendEmbed(ctx.ChannelID, embed)
	error.ErrorCheckPanic(err)
	return message
}
