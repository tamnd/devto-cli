package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// periodToTop maps --period flag values to the DEV API top= param.
var periodToTop = map[string]int{
	"week":  7,
	"month": 30,
	"year":  365,
	"all":   0,
}

func (a *App) topCmd() *cobra.Command {
	var period string
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Top articles on DEV Community",
		Long: `Fetch the top articles on DEV Community over a chosen time window.

--period controls the window: week (past 7 days), month (past 30 days),
year (past 365 days), or all (all time).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			top, ok := periodToTop[period]
			if !ok {
				return codeError(exitUsage, fmt.Errorf("unknown period %q: choose week, month, year, or all", period))
			}
			n := a.effectiveLimit(30)
			a.progressf("fetching top %d articles (period: %s)...", n, period)
			articles, err := a.client.Articles(cmd.Context(), top, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(articles, len(articles))
		},
	}
	cmd.Flags().StringVar(&period, "period", "week", "time window: week|month|year|all")
	return cmd
}
