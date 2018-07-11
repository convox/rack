package stdcli

func New(name, version string) *Engine {
	e := &Engine{
		Name:    name,
		Reader:  DefaultReader,
		Version: version,
		Writer:  DefaultWriter,
	}

	e.Command("help", "list commands", Help, CommandOptions{
		Validate: ArgsBetween(0, 1),
	})

	return e
}
