package weversemoment

import (
	"context"
	"sync"

	"github.com/Seklfreak/geverse"
)

var (
	artistCaches     map[int64]map[int64][]geverse.ArtistPost
	artistCachesLock sync.Mutex
)

func resetCache() {
	artistCachesLock.Lock()
	defer artistCachesLock.Unlock()

	artistCaches = nil
}

func get(ctx context.Context, geverseClient *geverse.Geverse, communityID, artistID int64) ([]geverse.ArtistPost, error) {
	artistCachesLock.Lock()
	defer artistCachesLock.Unlock()

	if artistCaches == nil {
		artistCaches = make(map[int64]map[int64][]geverse.ArtistPost)
	}
	if artistCaches[communityID] == nil {
		artistCaches[communityID] = make(map[int64][]geverse.ArtistPost)
	}

	if artistCaches[communityID][artistID] != nil {

		return artistCaches[communityID][artistID], nil
	}

	artistPosts, err := geverseClient.GetArtistMoments(ctx, communityID, artistID)
	if err != nil {
		return nil, err
	}

	artistCaches[communityID][artistID] = artistPosts.Posts
	return artistPosts.Posts, nil
}
