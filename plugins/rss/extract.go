package rss

import (
	"github.com/mmcdole/gofeed"
)

func getThumbnailURL(post *gofeed.Item) string {
	if post.Image != nil && post.Image.URL != "" {
		return post.Image.URL
	}

	for key, extension := range post.Extensions {
		if key == "media" {
			for valueKey, value := range extension {
				if valueKey == "content" && len(value) > 0 {
					content := value[len(value)-1]
					for attrKey, attr := range content.Attrs {
						if attrKey == "url" && attr != "" {
							return attr
						}
					}
				}
			}
		}
	}

	return ""
}
