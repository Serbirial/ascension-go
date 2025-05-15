package checks

import (
	"ascension/models"
	"errors"
	"strconv"
	"strings"
)

func NeedsArgs(ctx *models.Context) error {
	var neededArgCount int = 0
	for range ctx.CurrentCommand.Args {
		neededArgCount++
	}
	var args []string = strings.Fields(ctx.ArgsRaw)
	if neededArgCount > len(args) {
		return errors.New("Command is missing args, there are **" + strconv.Itoa(neededArgCount) + "** required args but you gave **" + strconv.Itoa(len(args)) + "**.")
	}

	return nil
}

func HasArgs(args map[string]string) bool {
	var actualArgs []string

	for _, providedArg := range args {
		if providedArg != "" { // Prevent empty args from being passed.
			actualArgs = append(actualArgs, providedArg)
		}
	}

	if len(actualArgs) > 0 {
		return true
	} else {
		return false
	}
}
