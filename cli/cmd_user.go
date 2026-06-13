package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/devto-cli/devto"
)

func (a *App) userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user <username>",
		Short: "Show a DEV user profile and their articles",
		Long: `Fetch a DEV Community user profile and their published articles.

Prints the user record first, then the article list.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			a.progressf("fetching user %q...", username)
			user, err := a.client.User(cmd.Context(), username)
			if err != nil {
				return mapFetchErr(err)
			}
			if err := a.render([]devto.User{user}); err != nil {
				return err
			}
			n := a.effectiveLimit(30)
			a.progressf("fetching %d articles by %q...", n, username)
			articles, err := a.client.UserArticles(cmd.Context(), username, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(articles, len(articles))
		},
	}
	return cmd
}
