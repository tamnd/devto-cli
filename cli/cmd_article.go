package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tamnd/devto-cli/devto"
)

func (a *App) articleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "article <id>",
		Short: "Fetch a single DEV article by ID",
		Long: `Fetch a single DEV Community article by its integer ID.

The id is the numeric article id, visible in the dev.to URL or from
the output of devto top / devto tag.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return codeError(exitUsage, fmt.Errorf("id must be an integer, got %q", args[0]))
			}
			a.progressf("fetching article %d...", id)
			article, err := a.client.ArticleByID(cmd.Context(), id)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]devto.Article{article})
		},
	}
	return cmd
}
