package commands

import (
	"gobot/models"
	"gobot/utils/checks"
)

var HelpCommand = models.Command{
	Name:     "help",
	Desc:     "List all commands.",
	Aliases:  []string{"h"},
	Args:     map[string]string{"command_name": "The name of the command you want the help message for."},
	Checks:   []func(*models.Context) error{},
	Callback: helpCommand,
}

func helpCommand(ctx *models.Context, args map[string]string) {
	toSend := ""
	if checks.HasArgs(args) {
		var command = ctx.Client.Commands[args["command_name"]]
		toSend += "**" + command.Name + "**: " + command.Desc + "\n"
		if len(command.Args) > 0 {
			toSend += "\nArgs:\n"

			for name, desc := range command.Args {
				toSend += "		**" + name + "**: " + desc + "\n"

			}
		}

		ctx.Send(toSend)
		return
	}
	for name, command := range ctx.Client.Commands {
		toSend += "`" + name + "`: " + command.Desc + "\n"
	}
	ctx.Send(toSend)
}
