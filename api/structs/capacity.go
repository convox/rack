package structs

type Capacity struct {
	ClusterMemory  int64 `json:"cluster-memory"`
	InstanceMemory int64 `json:"instance-memory"`
	ClusterCPU     int64 `json:"cluster-cpu"`
	InstanceCPU    int64 `json:"instance-cpu"`
	ProcessCount   int64 `json:"process-count"`
	ProcessMemory  int64 `json:"process-memory"`
	ProcessCPU     int64 `json:"process-cpu"`
	ProcessWidth   int64 `json:"process-width"`
}
