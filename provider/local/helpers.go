package local

func coalesce(strings ...string) string {
	for _, s := range strings {
		if s != "" {
			return s
		}
	}

	return ""
}

func coalescei(ints ...int) int {
	for _, i := range ints {
		if i > 0 {
			return i
		}
	}

	return 0
}

func ci(i *int64, def int64) int64 {
	if i != nil {
		return *i
	}
	return def
}

func cs(s *string, def string) string {
	if s != nil {
		return *s
	}
	return def
}

func diff(all []string, remove []string) []string {
	f := []string{}

	for _, a := range all {
		found := false

		for _, r := range remove {
			if a == r {
				found = true
				break
			}
		}

		if !found {
			f = append(f, a)
		}
	}

	return f
}
