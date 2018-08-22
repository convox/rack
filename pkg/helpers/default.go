package helpers

import "time"

func DefaultBool(v *bool, def bool) bool {
	if v == nil {
		return def
	}

	return *v
}

func DefaultDuration(v *time.Duration, def time.Duration) time.Duration {
	if v == nil {
		return def
	}

	return *v
}

func DefaultInt(v *int, def int) int {
	if v == nil {
		return def
	}

	return *v
}

func DefaultInt32(v *int32, def int32) int32 {
	if v == nil {
		return def
	}

	return *v
}

func DefaultString(v *string, def string) string {
	if v == nil {
		return def
	}

	return *v
}
