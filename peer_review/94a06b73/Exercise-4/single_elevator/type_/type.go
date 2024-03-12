package config

import "Elevator-go/elevio"

const NumFloors = 4
const NumButtons = 3
const DoorOpenDuration = 3

type Behaviour int

const (
	Idle     Behaviour = 0
	DoorOpen Behaviour = 1
	Moving   Behaviour = 2
)

type ClearRequestVariant int

const (
	// Assume everyone waiting for the elevator gets on the elevator, even if
	// they will be traveling in the "wrong" direction for a while
	CV_All ClearRequestVariant = 0

	// Assume that only those that want to travel in the current direction
	// enter the elevator, and keep waiting outside otherwise
	CV_InDirn ClearRequestVariant = 1
)

type Config struct {
	ClearRequestVariant ClearRequestVariant
	TimerCount          int
}

type Direction int

type Action struct {
	Dirin Direction
	Behav Behaviour
}

type Elevator struct {
	Floor    int
	Dir      elevio.MotorDirection
	Requests [][]bool
	Behave   Behaviour
	Econfig  Config
}
