package manifest

type RunSorter struct {
	names    []string
	manifest Manifest
}

func (rs RunSorter) Len() int {
	return len(rs.names)
}

func (rs RunSorter) Less(i, j int) bool {
	for _, link := range rs.manifest[rs.names[j]].Links {
		if link == rs.names[i] {
			return true
		}
	}

	return false
}

func (rs RunSorter) Swap(i, j int) {
	rs.names[i], rs.names[j] = rs.names[j], rs.names[i]
}
