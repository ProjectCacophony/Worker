package rss

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mmcdole/gofeed"
)

func getFeed(client *http.Client, parser *gofeed.Parser, feedURL string) (*gofeed.Feed, error) {
	parsedFeedURL, err := url.Parse(feedURL)
	if err != nil {
		return nil, err
	}

	// add cache busting
	newQueries := parsedFeedURL.Query()
	newQueries.Set("_", strconv.FormatInt(time.Now().Unix(), 10))
	parsedFeedURL.RawQuery = newQueries.Encode()

	// download feed page
	resp, err := client.Get(parsedFeedURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parser.Parse(resp.Body)
}
