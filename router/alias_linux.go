package router

func createAlias(iface, ip string) error {
	if err := execute("ip", "addr", "add", ip, "dev", iface); err != nil {
		return err
	}

	return nil
}
