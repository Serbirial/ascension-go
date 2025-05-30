package commands

import (
	"ascension/models"
	"strconv"
)

var OwnersListCommand = models.Command{
	Name:     "owners",
	Desc:     "List all owners.",
	Aliases:  []string{"o"},
	Checks:   []func(*models.Context) error{},
	Callback: listCommand,
}

func listCommand(ctx *models.Context, args map[string]string) {
	toSend := ""
	for i := 0; i < len(ctx.Client.Owners); i++ {
		toSend += strconv.Itoa(ctx.Client.Owners[i]) + "\n"
	}
	ctx.Send(toSend)
}
