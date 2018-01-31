package sdk

func coalesce(strings ...string) string {
	for _, s := range strings {
		if s != "" {
			return s
		}
	}

	return ""
}
