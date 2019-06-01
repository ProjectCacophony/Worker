package patrons

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type PatreonAPI struct {
	client              *http.Client
	creatorsAccessToken string
	campaignID          string
}

func NewPatreonAPI(
	client *http.Client,
	creatorsAccessToken string,
	campaignID string,
) *PatreonAPI {
	return &PatreonAPI{
		client:              client,
		creatorsAccessToken: creatorsAccessToken,
		campaignID:          campaignID,
	}
}

func (p *PatreonAPI) makeRequest(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+p.creatorsAccessToken)
	req = req.WithContext(ctx)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf(
			"received unexpected status from Patreon API: %s", resp.Status,
		)
	}

	return ioutil.ReadAll(resp.Body)
}

func (p *PatreonAPI) GetCampaignMembers(
	ctx context.Context,
) ([]*CampaignMember, error) {
	endpoint, err := url.Parse(
		apiBase + "api/oauth2/v2/campaigns/" + p.campaignID + "/members",
	)
	if err != nil {
		return nil, err
	}

	query := endpoint.Query()
	query.Set("include", "user,currently_entitled_tiers")
	query.Set("fields[member]", "patron_status")
	query.Set("fields[tier]", "title,discord_role_ids")
	query.Set("fields[user]", "first_name,full_name,vanity,social_connections")
	endpoint.RawQuery = query.Encode()

	raw, err := p.makeRequest(
		ctx,
		endpoint.String(),
	)
	if err != nil {
		return nil, err
	}

	var campaignMembersList []*campaignMembersResponse
	var campaignMembers *campaignMembersResponse
	err = json.Unmarshal(raw, &campaignMembers)
	if err != nil {
		return nil, err
	}

	campaignMembersList = append(campaignMembersList, campaignMembers)
	next := campaignMembers.Links.Next

	for {
		if next == "" {
			break
		}

		raw, err = p.makeRequest(
			ctx,
			next,
		)
		if err != nil {
			return nil, err
		}

		var campaignMembers *campaignMembersResponse
		err = json.Unmarshal(raw, &campaignMembers)
		if err != nil {
			return nil, err
		}

		campaignMembersList = append(campaignMembersList, campaignMembers)
		next = campaignMembers.Links.Next
	}

	return parseCampaignMembersResponse(campaignMembersList), nil
}

func parseCampaignMembersResponse(resps []*campaignMembersResponse) []*CampaignMember {
	var result []*CampaignMember

	for _, resp := range resps {
		for _, data := range resp.Data {
			member := &CampaignMember{
				ID:           data.Relationships.User.Data.ID,
				PatronStatus: data.Attributes.PatronStatus,
			}

			for _, included := range resp.Included {
				if included.Type != "user" {
					continue
				}

				if included.ID != member.ID {
					continue
				}

				if included.Attributes.FirstName != "" {
					member.FirstName = included.Attributes.FirstName
				}
				if included.Attributes.FullName != "" {
					member.FirstName = included.Attributes.FullName
				}
				if included.Attributes.Vanity != "" {
					member.FirstName = included.Attributes.Vanity
				}
				if included.Attributes.SocialConnections.Discord.UserID != "" {
					member.DiscordUserID = included.Attributes.SocialConnections.Discord.UserID
				}
			}

			result = append(result, member)
		}
	}

	return result
}

type CampaignMember struct {
	ID            string
	Vanity        string
	FirstName     string
	FullName      string
	PatronStatus  string
	DiscordUserID string
}

type campaignMembersResponse struct {
	Data []struct {
		Attributes struct {
			PatronStatus string `json:"patron_status"`
		} `json:"attributes"`
		ID            string `json:"id"`
		Relationships struct {
			CurrentlyEntitledTiers struct {
				Data []struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
			} `json:"currently_entitled_tiers"`
			User struct {
				Data struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
				Links struct {
					Related string `json:"related"`
				} `json:"links"`
			} `json:"user"`
		} `json:"relationships"`
		Type string `json:"type"`
	} `json:"data"`
	Included []struct {
		Attributes struct {
			Title             string   `json:"title"`
			FirstName         string   `json:"first_name"`
			FullName          string   `json:"full_name"`
			DiscordRoleIds    []string `json:"discord_role_ids"`
			SocialConnections struct {
				Deviantart interface{} `json:"deviantart"`
				Discord    struct {
					URL    string `json:"url"`
					UserID string `json:"user_id"`
				} `json:"discord"`
				Facebook  interface{} `json:"facebook"`
				Instagram interface{} `json:"instagram"`
				Reddit    interface{} `json:"reddit"`
				Spotify   interface{} `json:"spotify"`
				Twitch    interface{} `json:"twitch"`
				Twitter   interface{} `json:"twitter"`
				Youtube   interface{} `json:"youtube"`
			} `json:"social_connections"`
			Vanity string `json:"vanity"`
		} `json:"attributes,omitempty"`
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"included"`
	Links struct {
		Next string `json:"next"`
	} `json:"links"`
	Meta struct {
		Pagination struct {
			Cursors struct {
				Next string `json:"next"`
			} `json:"cursors"`
			Total int `json:"total"`
		} `json:"pagination"`
	} `json:"meta"`
}
