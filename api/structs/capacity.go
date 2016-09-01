package structs

type Capacity struct {
	ClusterCPU     int64 `json:"cluster-cpu"`
	ClusterMemory  int64 `json:"cluster-memory"`
	InstanceCPU    int64 `json:"instance-cpu"`
	InstanceMemory int64 `json:"instance-memory"`
	ProcessCount   int64 `json:"process-count"`
	ProcessCPU     int64 `json:"process-cpu"`
	ProcessMemory  int64 `json:"process-memory"`
	ProcessWidth   int64 `json:"process-width"`
}
