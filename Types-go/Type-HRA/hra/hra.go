package hra

import (
	"Driver-go/elevio"
	"Types-go/Type-Node/node"
)

const NumButtons int = 3

//Hall requests assigner structure of the elevator state for the cost function
type HRAElevState struct {
	Behavior    string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

//Hall requests assigner structure of the input requests for the cost function
type HRAInput struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

func DatabasetoHRA(database map[string]node.NetworkNode, numFloors int) map[string]HRAElevState {
	database_json := make(map[string]HRAElevState)
	var dir string
	CabRequests := make([]bool, numFloors)

	for k := range database {
		if database[k].Initiated {
			if database[k].Elevator.Direction == elevio.MD_Up {
				dir = "up"
			} else if database[k].Elevator.Direction == elevio.MD_Down {
				dir = "down"
			} else {
				dir = "stop"
			}

			for f := 0; f < numFloors; f++ {
				CabRequests[f] = database[k].Elevator.Requests[f][2]
			}
			elev_k := HRAElevState{
				Behavior:    string(database[k].Elevator.State),
				Floor:       database[k].Elevator.Floor,
				Direction:   dir,
				CabRequests: CabRequests[:],
			}

			database_json[k] = elev_k
		}
	}
	return database_json
}
