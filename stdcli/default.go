package stdcli

func Default(value, def string) string {
	if value == "" {
		return def
	} else {
		return value
	}
}
