package exec

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const logFormat = "--pretty=format:%H|%h|%an|%ar|%s"

var (
	defaultBaseBranches = []string{"origin/main", "origin/master", "origin/develop"}
	errNoBaseCommit     = errors.New("no upstream or default remote branch found")
)

// GitLog executes `git log` and returns the last 20 commits.
func GitLog(ctx context.Context) ([]GitCommit, error) {
	return executeAndParseLog(ctx, "-20")
}

// GetSquashableCommits returns commits that haven't been pushed.
// If no remote is found, it falls back to listing recent commits from the log.
func GetSquashableCommits(ctx context.Context) ([]GitCommit, error) {
	baseCommit, err := findBaseCommit(ctx)
	if err != nil {
		// If the error is specifically that no base was found, use the fallback.
		if errors.Is(err, errNoBaseCommit) {
			return getCommitsFromLogFallback(ctx)
		}
		// For any other error, return it.
		return nil, fmt.Errorf("could not find base commit: %w", err)
	}

	return executeAndParseLog(ctx, fmt.Sprintf("%s..HEAD", baseCommit))
}

// SquashCommits performs a soft reset and creates a new squashed commit.
func SquashCommits(ctx context.Context, commitHash, message string) error {
	if !CommitExists(ctx, commitHash) {
		return fmt.Errorf("commit hash '%s' does not exist", commitHash)
	}

	resetCmd := exec.CommandContext(ctx, "git", "reset", "--soft", commitHash+"^")
	if output, err := resetCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to git reset: %w\n%s", err, output)
	}

	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create squashed commit: %w\n%s", err, output)
	}

	return nil
}

// CommitExists checks if a commit hash is valid.
func CommitExists(ctx context.Context, commitHash string) bool {
	cmd := exec.CommandContext(ctx, "git", "cat-file", "-e", commitHash)
	return cmd.Run() == nil
}

// getCommitsFromLogFallback implements the original code's behavior for when no remote is found.
// It returns all but the oldest of the last 50 commits.
func getCommitsFromLogFallback(ctx context.Context) ([]GitCommit, error) {
	allCommits, err := executeAndParseLog(ctx, "-50")
	if err != nil {
		return nil, fmt.Errorf("fallback failed to get git log: %w", err)
	}

	if len(allCommits) > 1 {
		// The original logic returned all commits except the last one in the list (the oldest).
		return allCommits[:len(allCommits)-1], nil
	}

	return []GitCommit{}, nil
}

// findBaseCommit tries to find a remote branch to compare against.
func findBaseCommit(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	for _, branch := range defaultBaseBranches {
		checkCmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", branch)
		if checkCmd.Run() == nil {
			return branch, nil
		}
	}

	return "", errNoBaseCommit
}

// executeAndParseLog is a helper to run and parse git log commands.
func executeAndParseLog(ctx context.Context, args ...string) ([]GitCommit, error) {
	cmdArgs := append([]string{"log", logFormat}, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)

	output, err := cmd.Output()
	if err != nil {
		if len(output) == 0 {
			return []GitCommit{}, nil // Not an error, just no commits in range.
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []GitCommit{}, nil
	}

	commits := make([]GitCommit, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}
		commits = append(commits, GitCommit{
			Hash: parts[0], HashShort: parts[1], Author: parts[2], Time: parts[3], Comment: parts[4],
		})
	}
	return commits, nil
}
