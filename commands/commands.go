package commands

import (
	"gobot/models"
)

var AllCommands = map[string]models.Command{
	HelpCommand.Name:       HelpCommand,
	PingCommand.Name:       PingCommand,
	OwnersListCommand.Name: OwnersListCommand,
	MemberInfoCommand.Name: MemberInfoCommand,
	PlayCommand.Name:       PlayCommand,
	StopCommand.Name:       StopCommand,
}
