package router

func createInterface(name, ip string) error {
	if err := execute("ip", "link", "add", "link", "docker0", "name", name, "type", "vlan", "id", "2"); err != nil {
		return err
	}

	return nil
}

func destroyInterface(name string) error {
	return execute("ip", "link", "delete", name)
}
