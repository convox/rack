package helpers

func CoalesceInt(ii ...int) int {
	for _, i := range ii {
		if i != 0 {
			return i
		}
	}
	return 0
}

func CoalesceString(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}
