package structs

type Capacity struct {
	ClusterMemory  int64 `json:"cluster-memory"`
	InstanceMemory int64 `json:"instance-memory"`
	ProcessCount   int64 `json:"process-count"`
	ProcessMemory  int64 `json:"process-memory"`
	ProcessWidth   int64 `json:"process-width"`
}
