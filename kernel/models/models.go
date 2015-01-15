package models

type App struct {
	Name   string
	Status string

	CpuUsed     int
	CpuTotal    int
	MemoryUsed  int
	MemoryTotal int
	DiskUsed    int
	DiskTotal   int

	Cluster   *Cluster
	Processes Processes
}

type Apps []App

type Cluster struct {
	Name   string
	Status string

	CpuUsed     int
	CpuTotal    int
	MemoryUsed  int
	MemoryTotal int
	DiskUsed    int
	DiskTotal   int

	Apps Apps
}

type Clusters []Cluster

type Container struct {
	Name string

	CpuUsed     int
	CpuTotal    int
	MemoryUsed  int
	MemoryTotal int
	DiskUsed    int
	DiskTotal   int
}

type Containers []Container

type Process struct {
	Name    string
	Command string
	Count   int

	CpuUsed     int
	CpuTotal    int
	MemoryUsed  int
	MemoryTotal int
	DiskUsed    int
	DiskTotal   int

	App        *App
	Containers Containers
}

type Processes []Process
