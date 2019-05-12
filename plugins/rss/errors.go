package rss

import (
	"strings"
)

// nolint: gosimple
func isAcceptableError(err error) bool {
	if strings.Contains(
		err.Error(),
		"Failed to detect feed type",
	) {
		return true
	}

	return false
}
