package cmd

import (
	"github.com/andrejsstepanovs/git-squash/handlers"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git-squash",
		Short: "A CLI tool to squash git commits",
		Long:  `git-squash is a CLI tool that helps you squash multiple git commits into a single commit.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			hash, err := cmd.Flags().GetString("hash")
			if err != nil {
				return err
			}
			message, err := cmd.Flags().GetString("message")
			if err != nil {
				return err
			}
			max, err := cmd.Flags().GetBool("max")
			if err != nil {
				return err
			}

			return handlers.MainHandler(cmd.Context(), hash, message, max)
		},
	}

	cmd.Flags().StringP("hash", "a", "", "Optional commit hash to squash from")
	cmd.Flags().StringP("message", "m", "", "Optional commit message for the squashed commit")
	cmd.Flags().BoolP("max", "", false, "Auto-select the oldest selectable commit")

	return cmd
}
