package router

func createInterface(name, ip string) error {
	if err := execute("ifconfig", name, "create"); err != nil {
		return err
	}

	return nil
}

func destroyInterface(name string) error {
	return execute("ifconfig", name, "destroy")
}
