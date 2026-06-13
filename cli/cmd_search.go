package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// searchCmd returns the `search <query>` command.
func (a *App) searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search Coding Horror posts by title or summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			n := a.effectiveLimit(20)
			a.progressf("searching posts for %q...", query)
			posts, err := a.client.Search(cmd.Context(), query, n)
			if err != nil {
				return mapFetchErr(err)
			}
			if len(posts) == 0 {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "no posts matched %q\n", query)
				return codeError(exitNoData, nil)
			}
			return a.renderOrEmpty(posts, len(posts))
		},
	}
}
