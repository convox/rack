package router

func createAlias(iface, ip string) error {
	if err := execute("ifconfig", iface, "alias", ip, "netmask", "255.255.255.255"); err != nil {
		return err
	}

	return nil
}
