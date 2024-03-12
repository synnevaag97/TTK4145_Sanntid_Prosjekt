package data_structure

import (
	"Elevdriver/config"
	"Elevdriver/elevio"
)

type Elevator_data_t struct {
	Behaviour   string                  `json:"behaviour"`   //idle, moving, doorOpen
	Floor       int                     `json:"floor"`       //uint
	Direction   string                  `json:"direction"`   //up, down, stop
	CabRequests [config.NUM_FLOORS]bool `json:"cabRequests"` //array of booleans identifying if a cabrequest is present for floor.
}

type Cost_data_t struct {
	HallRequests [config.NUM_FLOORS][2]bool `json:"hallRequests"`
	States       map[int]*Elevator_data_t   `json:"states"`
}

type Order_t struct {
	Floor     int
	Direction elevio.ButtonType //0 = Up, 1 = down, 2 = cabReq.
	Finished  bool
}

type Received_elevator_data_t struct {
	Elevator_id   int
	Elevator_data Elevator_data_t
}

type Order_queue_t struct {
	Floor     int
	Direction elevio.ButtonType
}

type Order_list_t struct {
	Floor     int
	Direction elevio.ButtonType
}

type System_info_t struct {
	Id        int
	ElevPort  int
	SuperPort int
	PeerPort  int
	Init      bool
}

type Arbitration_t struct {
	Is_master  bool
	Alive_list [config.NUM_ELEVATORS]bool // List of elevators we still have a connection to
	Connected  bool
}
