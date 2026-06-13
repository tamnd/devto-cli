package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) tagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag <tag>",
		Short: "Articles tagged with a DEV tag",
		Long: `Fetch the most recent articles tagged with <tag> on DEV Community.

<tag> is the tag slug, e.g. go, python, webdev.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]
			n := a.effectiveLimit(30)
			a.progressf("fetching %d articles tagged %q...", n, tag)
			articles, err := a.client.TagArticles(cmd.Context(), tag, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(articles, len(articles))
		},
	}
	return cmd
}
