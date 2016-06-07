package structs

type Capacity struct {
	ClusterMemory  int64 `json:"cluster-memory"`
	InstanceMemory int64 `json:"instance-memory"`
	ClusterCpu     int64 `json:"cluster-cpu"`
	InstanceCpu    int64 `json:"instance-cpu"`
	ProcessCount   int64 `json:"process-count"`
	ProcessMemory  int64 `json:"process-memory"`
	ProcessCpu     int64 `json:"process-cpu"`
	ProcessWidth   int64 `json:"process-width"`
}
