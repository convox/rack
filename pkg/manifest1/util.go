package manifest1

func coalesce(values ...string) string {
	for _, s := range values {
		if s != "" {
			return s
		}
	}

	return ""
}
