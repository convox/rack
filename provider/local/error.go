package local

func (p *Provider) ErrorFormat(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
