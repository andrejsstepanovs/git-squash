package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrejsstepanovs/git-squash/exec"
	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
	"golang.org/x/term"
)

// MainHandler retrieves the current user absolute directory and prints it
// Then calls GitLog to show last 20 commits
func MainHandler(ctx context.Context, selectedCommit, commitMessage string, max bool) error {
	// Show current directory
	dir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan(dir))

	// Get squashable commits
	commits, err := exec.GetSquashableCommits(ctx)
	if err != nil {
		return fmt.Errorf("failed to get squashable commits: %w", err)
	}

	if len(commits) <= 1 {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Println(yellow("No squashable commits found"))
		return nil
	}

	h := &handler{}
	selectedCommit, err = h.handleCommitSelection(ctx, commits, selectedCommit, max)
	if err != nil {
		return err
	}

	count, err := h.validateSelectedCommit(ctx, commits, selectedCommit)
	if err != nil {
		return err
	}

	commitMessage, err = h.handleCommitMessage(commitMessage, selectedCommit, commits)
	if err != nil {
		return err
	}

	// Perform squash
	err = exec.SquashCommits(ctx, selectedCommit, commitMessage)
	if err != nil {
		return fmt.Errorf("failed to squash commits: %w", err)
	}

	success := color.New(color.FgHiGreen, color.Bold).SprintFunc()
	fmt.Printf("%d %s\n", count, success("commits successfully squashed"))

	return nil
}

// formatCommitLine formats a commit line with hash, author, time, and message
func formatCommitLine(commitLine exec.GitCommit) string {
	hash := commitLine.HashShort
	author := commitLine.Author
	time := commitLine.Time
	message := commitLine.Comment

	// Trim message to 50 characters and ensure it's a single line
	message = strings.TrimSpace(message)
	if len(message) > 50 {
		message = message[:50] + "..."
	}
	// Replace newlines with spaces
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")

	// Format the commit line with colors
	hashColor := color.New(color.FgYellow).SprintFunc()
	authorColor := color.New(color.FgGreen).SprintFunc()
	timeColor := color.New(color.FgCyan).SprintFunc()
	messageColor := color.New(color.FgWhite).SprintFunc()

	return fmt.Sprintf("%s %s %s %s",
		hashColor(hash),
		authorColor(author),
		timeColor(time),
		messageColor(message))
}

// selectCommit allows user to interactively select a commit using arrow keys (scrollable)
func selectCommit(ctx context.Context, commits []exec.GitCommit) (exec.GitCommit, error) {
	if len(commits) == 0 {
		return exec.GitCommit{}, fmt.Errorf("no commits to select")
	}

	// Get terminal size
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		height = 20 // fallback if cannot detect
	}

	// Reserve 4 lines: instructions + header + padding
	visibleCount := height - 4
	if visibleCount < 3 {
		visibleCount = 3 // minimum to still work
	}

	if err := keyboard.Open(); err != nil {
		return exec.GitCommit{}, fmt.Errorf("failed to open keyboard: %w", err)
	}
	defer keyboard.Close()

	selectedIndex := 0
	startIndex := 0

	printCommitsWithSelectionWindow(commits, selectedIndex, startIndex, visibleCount)

	keyboardChan := make(chan keyboard.KeyEvent)
	errorChan := make(chan error)

	go func() {
		defer close(keyboardChan)
		for {
			char, key, err := keyboard.GetKey()
			if err != nil {
				select {
				case errorChan <- err:
				case <-ctx.Done():
				}
				return
			}
			select {
			case keyboardChan <- keyboard.KeyEvent{Rune: char, Key: key}:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			clearLines(visibleCount + 4)
			return exec.GitCommit{}, fmt.Errorf("operation cancelled: %w", ctx.Err())
		case err := <-errorChan:
			clearLines(visibleCount + 4)
			return exec.GitCommit{}, fmt.Errorf("keyboard error: %w", err)
		case event := <-keyboardChan:
			clearLines(visibleCount + 4)
			switch event.Key {
			case keyboard.KeyArrowUp:
				if selectedIndex > 0 {
					selectedIndex--
					if selectedIndex < startIndex {
						startIndex--
					}
				}
			case keyboard.KeyArrowDown:
				if selectedIndex < len(commits)-1 {
					selectedIndex++
					if selectedIndex >= startIndex+visibleCount {
						startIndex++
					}
				}
			case keyboard.KeyEnter, keyboard.KeySpace:
				return commits[selectedIndex], nil
			case keyboard.KeyEsc, keyboard.KeyCtrlC, keyboard.KeyCtrlD:
				return exec.GitCommit{}, fmt.Errorf("selection cancelled")
			default:
				if event.Rune == 3 || event.Rune == 4 {
					return exec.GitCommit{}, fmt.Errorf("selection cancelled")
				}
			}
			printCommitsWithSelectionWindow(commits, selectedIndex, startIndex, visibleCount)
		}
	}
}

func printCommitsWithSelectionWindow(commits []exec.GitCommit, selectedIndex, startIndex, visibleCount int) {
	instruction := color.New(color.FgHiMagenta).SprintFunc()
	fmt.Printf("%s\n", instruction("Select a first included commit (use arrow keys, Enter to select, Esc to cancel):"))

	header := color.New(color.FgHiWhite, color.Bold).SprintFunc()
	fmt.Printf("%s\n", header("HASH   AUTHOR        TIME                 MESSAGE"))

	endIndex := startIndex + visibleCount
	if endIndex > len(commits) {
		endIndex = len(commits)
	}

	for i := startIndex; i < endIndex; i++ {
		formattedCommit := formatCommitLine(commits[i])
		if i == selectedIndex {
			selected := color.New(color.FgHiWhite, color.BgBlue).SprintFunc()
			fmt.Printf("> %s\n", selected(formattedCommit))
		} else {
			fmt.Printf("  %s\n", formattedCommit)
		}
	}
}

// clearLines clears the specified number of lines from the terminal
func clearLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[1A\033[K")
	}
}

func containsCommit(commits []exec.GitCommit, search string) (bool, int) {
	for i, c := range commits {
		if c.Hash == search || c.HashShort == search {
			return true, i + 1
		}
	}

	return false, 0
}
