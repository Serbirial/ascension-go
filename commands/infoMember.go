package commands

import (
	"fmt"
	"gobot/models"
	"gobot/utils/checks"
	"gobot/utils/embeds"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
)

var MemberInfoCommand = models.Command{
	Name:          "userinfo",
	Desc:          "Gets user information and displays it.",
	Aliases:       []string{"ui"},
	Args:          map[string]string{"user": "The name or ID of the user you want to display information about."},
	Subcommands:   []string{""},
	Parentcommand: "none",
	Checks:        []func(*models.Context) error{checks.NeedsArgs},
	Callback:      memberInfoCommand,
	Nsfw:          false,
	Endpoint:      "string",
}

func memberInfoCommand(ctx *models.Context, args map[string]string) {
	embed := &embeds.Embed{
		Title:       "",
		Description: "User Information",
		Color:       0x00ff00,
	}
	guild, err := ctx.Client.Session.State.Guild(ctx.GuildID)
	if err != nil {
		fmt.Println(err)
		return
	}
	membersAll := guild.Members
	memberSearch, err := ctx.Client.Session.GuildMembersSearch(ctx.GuildID, args["user"], 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(memberSearch) == 0 {
		ctx.Send("No user found.")
		return
	}
	var member *discordgo.Member = memberSearch[0]

	var joinedAtDates map[*discordgo.Member]time.Time = make(map[*discordgo.Member]time.Time)
	var joinedAtKeys []*discordgo.Member

	for i := range membersAll {
		joinedAtDates[membersAll[i]] = membersAll[i].JoinedAt
	}
	sort.SliceStable(joinedAtKeys, func(i, j int) bool {
		return joinedAtDates[joinedAtKeys[i]].Second() < joinedAtDates[joinedAtKeys[j]].Second()
	})
	fmt.Println(joinedAtKeys)

	embed.AddField("Nick", member.Nick, false)
	embed.AddField("Member No.", "placeholder", false)
	embed.AddFooter("Footer.")

	ctx.SendEmbed(embed.CreateDiscordEmbed())
}
