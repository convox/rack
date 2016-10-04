package structs

type Registry struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Registries []Registry
