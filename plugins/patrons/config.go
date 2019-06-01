package patrons

type Config struct {
	PatreonCampaignID          string `envconfig:"PATREON_CAMPAIGN_ID"`
	PatreonCreatorsAccessToken string `envconfig:"PATREON_CREATORS_ACCESS_TOKEN"`
}
