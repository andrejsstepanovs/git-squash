package exec

import (
	"fmt"
)

type GitCommit struct {
	Hash      string
	HashShort string
	Comment   string
	Author    string
	Time      string
}

func (c *GitCommit) String() string {
	comment := c.Comment
	if len(comment) > 50 {
		comment = comment[:50] + "..."
	}
	return fmt.Sprintf("%s %s %s %s", c.HashShort, c.Author, c.Time, comment)
}
