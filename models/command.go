package models

type Command struct {
	Name    string
	Desc    string
	Aliases []string

	Args map[string]string
	//ArgsDesc string

	Subcommands   []string // Names of all commands that are under this if it happens to be a parent command
	Parentcommand string

	Checks   []func(ctx *Context) error
	Callback func(ctx *Context, args map[string]string)

	Nsfw bool

	// These bits are for lana
	Endpoint string
}
