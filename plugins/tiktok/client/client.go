package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type Client struct {
	HttpClient *http.Client
	BaseURL    string
}

func NewClient(httpClient *http.Client, baseURL string) *Client {
	return &Client{HttpClient: httpClient, BaseURL: baseURL}
}

func (c *Client) Posts(ctx context.Context, username string) ([]*Post, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%susers/%s", c.BaseURL, username), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failure creating tiktok api request")
	}

	resp, err := c.HttpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "failure performing tiktok api request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrapf(err, "received unexpected status code from tiktok api: %d", resp.StatusCode)
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failure reading tiktok api response body")
	}

	var postsCol postsCollector
	err = json.Unmarshal(respData, &postsCol)
	if err != nil {
		return nil, errors.Wrap(err, "failure parsing tiktok api response body")
	}

	for i := range postsCol.Collector {
		postTimeStamp, err := strconv.ParseInt(postsCol.Collector[i].CreateTime, 10, 64)
		if err != nil {
			continue
		}

		postsCol.Collector[i].CreateTimeParsed = time.Unix(postTimeStamp, 0)
	}

	return postsCol.Collector, nil
}

type postsCollector struct {
	Collector []*Post `json:"collector"`
}

type Post struct {
	ID                  string `json:"id"`
	Text                string `json:"text"`
	CreateTime          string `json:"createTime"`
	CreateTimeParsed    time.Time
	AuthorMeta          PostAuthor `json:"authorMeta"`
	Covers              PostCovers `json:"covers"`
	ImageURL            string     `json:"imageUrl"`
	WebVideoURL         string     `json:"webVideoUrl"`
	VideoURL            string     `json:"videoUrl"`
	VideoURLNoWaterMark string     `json:"videoUrlNoWaterMark"`
	ShareCount          int        `json:"shareCount"`
	PlayCount           int        `json:"playCount"`
	CommentCount        int        `json:"commentCount"`
}

type PostAuthor struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NickName  string `json:"nickName"`
	Following int    `json:"following"`
	Fans      int    `json:"fans"`
	Heart     string `json:"heart"`
	Video     int    `json:"video"`
	Verified  bool   `json:"verified"`
	Private   bool   `json:"private"`
	Signature string `json:"signature"`
	Avatar    string `json:"avatar"`
}

type PostCovers struct {
	Default string `json:"default"`
	Origin  string `json:"origin"`
	Dynamic string `json:"dynamic"`
}
