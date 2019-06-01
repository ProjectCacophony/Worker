package patrons

func (p *Plugin) upsertPatrons(patrons []*Patron) error {
	var err error

	for _, patron := range patrons {
		err = upsertPatron(p.db, patron)
		if err != nil {
			return err
		}
	}

	return nil
}
