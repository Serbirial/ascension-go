package commands

import (
	"gobot/models"
)

var AllCommands = map[string]models.Command{
	HelpCommand.Name:       HelpCommand,
	PingCommand.Name:       PingCommand,
	OwnersListCommand.Name: OwnersListCommand,

	PlayCommand.Name:      PlayCommand,
	StopCommand.Name:      StopCommand,
	MusicInfoCommand.Name: MusicInfoCommand,
	JoinCommand.Name:      JoinCommand,
}
