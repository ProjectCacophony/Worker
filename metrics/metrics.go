package metrics

import "expvar"

var (
	// GallFeedPosts counts all gall posts posted
	GallFeedPosts = expvar.NewInt("gall_feed_posts")
	// FeedFeedPosts counts all feed posts posted
	FeedFeedPosts = expvar.NewInt("feed_feed_posts")
)
