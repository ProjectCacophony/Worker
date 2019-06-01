package patrons

import (
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/plugins/common"
)

func (p *Plugin) getPatrons(run *common.Run) ([]*Patron, error) {
	members, err := p.client.GetCampaignMembers(run.Context())
	if err != nil {
		return nil, errors.Wrap(err, "error querying for campaign members")
	}

	var patrons []*Patron // nolint: prealloc
	for _, member := range members {
		if member.ID == "" {
			continue
		}

		patrons = append(patrons, &Patron{
			PatreonUserID: member.ID,
			FirstName:     member.FirstName,
			VanityName:    member.Vanity,
			FullName:      member.FullName,
			PatronStatus:  member.PatronStatus,
			DiscordUserID: member.DiscordUserID,
		})
	}

	return patrons, nil
}
