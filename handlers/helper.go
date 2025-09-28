package handlers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/andrejsstepanovs/git-squash/exec"
	"github.com/c-bata/go-prompt"
	"github.com/fatih/color"
)

// ErrAbortedByUser is returned when the user aborts input (Ctrl+C or Ctrl+D)
var ErrAbortedByUser = errors.New("aborted by user")

type handler struct{}

func (h *handler) handleCommitSelection(ctx context.Context, commits []exec.GitCommit, selectedCommit string, max bool) (string, error) {
	if selectedCommit == "" {
		if max {
			if len(commits) > 0 {
				selectedCommit = commits[len(commits)-1].Hash
			}
		} else {
			header := color.New(color.FgHiBlue, color.Bold).SprintFunc()
			fmt.Printf("\n%s\n", header("Squashable commits:"))

			var err error
			commit, err := selectCommit(ctx, commits)
			if err != nil {
				return "", fmt.Errorf("failed to select commit: %w", err)
			}
			selectedCommit = commit.Hash
		}
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

func (h *handler) handleCommitMessage(commitMessage string, selectedCommitHash string, commits []exec.GitCommit) (string, error) {
	if commitMessage == "" {
		var selectedCommit *exec.GitCommit
		for _, c := range commits {
			if c.Hash == selectedCommitHash || c.HashShort == selectedCommitHash {
				selectedCommit = &c
				break
			}
		}

		abort := func() (string, error) {
			fmt.Fprintln(os.Stderr, "\nAborted by user.")
			return "", ErrAbortedByUser
		}

		if selectedCommit != nil {
			livePrefix := fmt.Sprintf("Commit message [%s]: ", selectedCommit.Comment)
			// Set up signal handling for Ctrl+C
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(sigChan)

			inputCh := make(chan string, 1)
			errCh := make(chan error, 1)

			go func() {
				defer close(inputCh)
				defer close(errCh)
				defer func() {
					if r := recover(); r != nil {
						errCh <- ErrAbortedByUser
					}
				}()
				msg := prompt.Input(
					livePrefix,
					func(d prompt.Document) []prompt.Suggest { return nil },
					prompt.OptionInitialBufferText(selectedCommit.Comment),
					prompt.OptionLivePrefix(func() (string, bool) { return livePrefix, true }),
					prompt.OptionAddKeyBind(prompt.KeyBind{
						Key: prompt.ControlC,
						Fn: func(*prompt.Buffer) {
							panic("abort")
						},
					}),
					prompt.OptionAddKeyBind(prompt.KeyBind{
						Key: prompt.ControlD,
						Fn: func(*prompt.Buffer) {
							panic("abort")
						},
					}),
				)
				inputCh <- msg
			}()

			select {
			case <-sigChan:
				return abort()
			case msg := <-inputCh:
				if msg == "" {
					return abort()
				}
				commitMessage = msg
			case <-errCh:
				return abort()
			}
		} else {
			fmt.Print("Commit message: ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				commitMessage = scanner.Text()
			} else {
				if err := scanner.Err(); err != nil && err != io.EOF {
					return "", fmt.Errorf("failed to read commit message: %w", err)
				}
				return abort()
			}
		}
	}

	commitMessage = strings.TrimSpace(commitMessage)
	if commitMessage == "" {
		return "", fmt.Errorf("commit message cannot be empty")
	}

	return commitMessage, nil
}
