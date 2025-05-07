package embeds

import (
	"github.com/bwmarrin/discordgo"
)

type Field struct {
	Name   string
	Value  string
	Inline bool
}

type Embed struct {
	Title        string
	Description  string
	Color        int
	Fields       []Field
	Footer       discordgo.MessageEmbedFooter
	DiscordEmbed *discordgo.MessageEmbed
}

func (embed *Embed) AddFooter(text string) {
	embed.Footer = discordgo.MessageEmbedFooter{Text: text}

}
func (embed *Embed) AddField(name string, value string, inline bool) {
	embed.Fields = append(embed.Fields, Field{
		Name:   name,
		Value:  value,
		Inline: inline,
	})
}

func (embed *Embed) CreateDiscordEmbed() *discordgo.MessageEmbed {
	var fields = []*discordgo.MessageEmbedField{}

	for i := range embed.Fields {
		var field Field = embed.Fields[i]

		createdField := &discordgo.MessageEmbedField{
			Name:   field.Name,
			Value:  field.Value,
			Inline: field.Inline,
		}
		fields = append(fields, createdField)
	}

	createdEmbed := &discordgo.MessageEmbed{
		Title:       embed.Title,
		Description: embed.Description,
		Color:       embed.Color,
		Fields:      fields,
		Footer:      &embed.Footer,
	}
	embed.DiscordEmbed = createdEmbed
	return createdEmbed
}
