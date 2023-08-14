package uidgenerator

import "time"

const (
	container = iota + 1
	actual
)

/*
workerNode for M_WORKER_NODE
*/
type workerNode struct {
	// unique id (table unique)
	Id uint64
	// Type of CONTAINER: HostName, ACTUAL : IP.
	HostName string
	// Type of CONTAINER: Port, ACTUAL : Timestamp + Random(0-10000)
	Port string
	// Type of CONTAINER or Actual
	Type int
	// Worker launch date, default now
	launchDate time.Time
	// Created time
	created time.Time
	// Last modified
	modified time.Time
}
