package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/andrejsstepanovs/git-squash/exec"
	"github.com/fatih/color"
)

type handler struct{}

func (h *handler) handleCommitSelection(ctx context.Context, commits []exec.GitCommit, selectedCommit string) (string, error) {
	if selectedCommit == "" {
		header := color.New(color.FgHiBlue, color.Bold).SprintFunc()
		fmt.Printf("\n%s\n", header("Squashable commits:"))

		var err error
		commit, err := selectCommit(ctx, commits)
		if err != nil {
			return "", fmt.Errorf("failed to select commit: %w", err)
		}
		selectedCommit = commit.Hash
	}

	if selectedCommit == "" {
		return "", fmt.Errorf("no commit selected")
	}

	return selectedCommit, nil
}

func (h *handler) validateSelectedCommit(ctx context.Context, commits []exec.GitCommit, selectedCommit string) (int, error) {
	if !(len(selectedCommit) == 40 || len(selectedCommit) == 7) {
		return 0, fmt.Errorf("invalid commit hash length")
	}

	if !exec.CommitExists(ctx, selectedCommit) {
		return 0, fmt.Errorf("selected commit does not exist")
	}

	exists, count := containsCommit(commits, selectedCommit)
	if !exists {
		return 0, fmt.Errorf("selected commit is not in the list of squashable commits")
	}

	if count <= 1 {
		return 0, fmt.Errorf("not enough commits selected")
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	countStr := fmt.Sprintf("%d", count)
	fmt.Printf("\n%s %s %s\n", yellow(countStr), green("commits selected including:"), selectedCommit)

	return count, nil
}

func (h *handler) handleCommitMessage(commitMessage string) (string, error) {
	if commitMessage == "" {
		var err error
		// Prompt for new commit message
		fmt.Print("\nEnter commit message for the squashed commit: ")
		commitMessage, err = readLine()
		if err != nil {
			return "", fmt.Errorf("failed to read commit message: %w", err)
		}
	}

	commitMessage = strings.TrimSpace(commitMessage)
	if commitMessage == "" {
		return "", fmt.Errorf("commit message cannot be empty")
	}

	return commitMessage, nil
}
