package generate

func Routes() ([]byte, error) {
	ms, err := Methods()
	if err != nil {
		return nil, err
	}

	params := map[string]interface{}{
		"Methods": ms,
	}

	data, err := renderTemplate("routes", params)
	if err != nil {
		return nil, err
	}

	return gofmt(data)
}
