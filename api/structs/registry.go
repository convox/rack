package structs

type Registry struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Registries []Registry

func (r Registries) Len() int           { return len(r) }
func (r Registries) Less(i, j int) bool { return r[i].Server < r[j].Server }
func (r Registries) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
