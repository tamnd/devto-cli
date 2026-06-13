package cli

import (
	"errors"

	"github.com/tamnd/devto-cli/devto"
)

func isNotFound(err error) bool {
	return errors.Is(err, devto.ErrNotFound)
}
