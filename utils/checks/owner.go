package checks

import (
	"errors"
	"gobot/models"
	"strconv"
)

func IsOwner(ctx *models.Context) error {
	for _, ownerID := range ctx.Client.Owners {
		if strconv.Itoa(ownerID) == ctx.Author.ID {
			return nil
		}
	}

	return errors.New("Owner only command.")
}
