package weversemedia

import (
	"github.com/Seklfreak/geverse"
)

func extractMediaURLs(items []geverse.Photo) []string {
	result := make([]string, len(items))

	for i, item := range items {
		result[i] = item.OrgImgURL
	}

	return result
}

func splitURLs(items []string) [][]string {
	chunkSize := 5
	var mediaURLsLeft [][]string

	// divide > 5 links into chunks of five
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize

		if end > len(items) {
			end = len(items)
		}

		mediaURLsLeft = append(mediaURLsLeft, items[i:end])
	}

	return mediaURLsLeft
}
