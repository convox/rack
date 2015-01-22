package models

import "time"

type App struct {
	Name       string
	Status     string
	Repository string
	Release    string

	CpuUsed     int
	CpuTotal    int
	MemoryUsed  int
	MemoryTotal int
	DiskUsed    int
	DiskTotal   int

	Cluster   *Cluster
	Builds    Builds
	Processes Processes
	Releases  Releases
}

type Apps []App

type Build struct {
	Id        string
	Status    string
	Release   string
	CreatedAt time.Time
	EndedAt   time.Time
	Logs      string
}

type Builds []Build

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

type Release struct {
	Id        string
	Ami       string
	CreatedAt time.Time
}

type Releases []Release
