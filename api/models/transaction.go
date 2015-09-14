package models

import "time"

// A Transaction groups more than one event on a Name/Type resource
type Transaction struct {
	Name   string
	Type   string
	Status string
	Start  time.Time
	End    time.Time
}

type Transactions []Transaction

func (slice Transactions) Len() int {
	return len(slice)
}

func (slice Transactions) Less(i, j int) bool {
	return slice[i].Start.Before(slice[j].Start)
}

func (slice Transactions) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
