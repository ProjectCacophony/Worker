package rss

import (
	"strings"
)

func isAcceptableError(err error) bool {
	if strings.Contains(
		err.Error(),
		"Failed to detect feed type",
	) {
		return true
	}

	return false
}
