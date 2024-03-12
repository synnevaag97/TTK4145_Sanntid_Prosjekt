package node

import (
	"Driver-go/elevio"
	"fmt"
)

//const numFloors int = 4
const NumButtons int = 3

// Database
type States string

const (
	UNDEFINED States = "undefined"
	IDLE      States = "idle"
	MOVING    States = "moving"
	DOOROPEN  States = "doorOpen"
)

type NetworkElevState struct {
	State     States
	Floor     int
	Direction elevio.MotorDirection
	Requests  [][NumButtons]bool
}

type NetworkNode struct {
	Initiated bool
	Elevator  NetworkElevState
}

func InitiateLocalDatabase(numFloors int, Id string) map[string]NetworkNode {
	thiselevator := NetworkNode{}
	thiselevator.Initiated = false
	thiselevator.Elevator.Requests = make([][3]bool, numFloors)
	thiselevator.Elevator.Direction = elevio.MD_Stop
	thiselevator.Elevator.Floor = -1
	thiselevator.Elevator.State = UNDEFINED
	database := make(map[string]NetworkNode)
	database[Id] = thiselevator
	return database
}

func InitiateGlobalDatabase(Id string, numFloors int, database *map[string]NetworkNode, initated_database_from_net map[string]NetworkNode) {
	for k := range initated_database_from_net {
		if k != Id {
			(*database)[k] = NetworkNode{}
			if thisE, ok := (*database)[k]; ok {
				thisE.Initiated = initated_database_from_net[k].Initiated
				thisE.Elevator = initated_database_from_net[k].Elevator
				(*database)[k] = thisE
			}
		} else {
			if thisE, ok := (*database)[k]; ok {
				for f := 0; f < numFloors; f++ {
					thisE.Elevator.Requests[f][2] = initated_database_from_net[k].Elevator.Requests[f][2]
				}
				(*database)[k] = thisE
			}
		}
	}
	if thisE, ok := (*database)[Id]; ok {
		thisE.Initiated = true
		(*database)[Id] = thisE
	}
}

func UpdateDatabase_AddHallRequests(numFloors int, database *map[string]NetworkNode, assignedHallRequest map[string][][2]bool) {
	for k := range assignedHallRequest {
		if thisE, ok := (*database)[k]; ok {
			for f := 0; f < numFloors; f++ {
				thisE.Elevator.Requests[f][0] = (assignedHallRequest)[k][f][0]
				thisE.Elevator.Requests[f][1] = (assignedHallRequest)[k][f][1]
			}
			(*database)[k] = thisE
		}
	}
}

func UpdateDatabase_AddElevatorChange(numFloors int, database *map[string]NetworkNode, elev_Id string, updates NetworkElevState) {
	if thisE, ok := (*database)[elev_Id]; ok {
		thisE.Elevator.Direction = updates.Direction
		thisE.Elevator.Floor = updates.Floor
		thisE.Elevator.State = updates.State

		for f := 0; f < 4; f++ {
			thisE.Elevator.Requests[f][2] = updates.Requests[f][2]
		}
		(*database)[elev_Id] = thisE
	}
}

func UpdateDatabase_RemoveRequest(database *map[string]NetworkNode, incoming_node_Id string, completed_hallReq elevio.ButtonEvent) {
	if thisE, ok := (*database)[incoming_node_Id]; ok {
		thisE.Elevator.Requests[completed_hallReq.Floor][completed_hallReq.Button] = false
		(*database)[incoming_node_Id] = thisE
	}
}

func PrintDatabase(database map[string]NetworkNode) {
	for k := range database {
		fmt.Printf("node: %s \n", k)
		fmt.Printf("|----------------------| \n")
		fmt.Printf("| Initiation %t| \n", database[k].Initiated)
		fmt.Printf("|----------------------| \n")
		fmt.Printf("|FLoor: %d| \n", database[k].Elevator.Floor)
		fmt.Printf("|Dir: %d| \n", database[k].Elevator.Direction)
		fmt.Printf("|State: %s| \n", database[k].Elevator.State)
		fmt.Printf("|----------------------| \n")
		fmt.Printf("| Request Database     |\n")
		fmt.Printf("|-------------------------------------------|\n")
		fmt.Printf("|Buttons\\Floor	|  1        2      3      4 |\n")
		fmt.Printf("|-------------------------------------------|\n")
		fmt.Printf("|Hall up 	| %t  %t  %t  %t|\n", database[k].Elevator.Requests[0][0], database[k].Elevator.Requests[1][0], database[k].Elevator.Requests[2][0], database[k].Elevator.Requests[3][0])
		fmt.Printf("|Hall down	| %t  %t  %t  %t|\n", database[k].Elevator.Requests[0][1], database[k].Elevator.Requests[1][1], database[k].Elevator.Requests[2][1], database[k].Elevator.Requests[3][1])
		fmt.Printf("|Cab 		| %t  %t  %t  %t|\n", database[k].Elevator.Requests[0][2], database[k].Elevator.Requests[1][2], database[k].Elevator.Requests[2][2], database[k].Elevator.Requests[3][2])
		fmt.Printf("|-------------------------------------------|\n")
	}
}

// Extract all active request from the lost nodes.and set them to uninitialised.
func GetActiveHallRequestsInNodes(nodes []string, database *map[string]NetworkNode) []elevio.ButtonEvent {
	var button elevio.ButtonEvent
	var hall_requests []elevio.ButtonEvent
	for k := range nodes {
		if thisE, ok := (*database)[nodes[k]]; ok {
			for f := 0; f < 4; f++ {
				if thisE.Elevator.Requests[f][0] {
					button.Floor = f
					button.Button = 0
					hall_requests = append(hall_requests, button)
				}
				if thisE.Elevator.Requests[f][1] {
					button.Floor = f
					button.Button = 1
					hall_requests = append(hall_requests, button)
				}
			}
			(*database)[nodes[k]] = thisE
		}
	}
	return hall_requests
}

func GetAllHallRequests(numFloors int, database map[string]NetworkNode) [][2]bool {
	hallrequest := make([][2]bool, numFloors)
	for k := range database {
		if database[k].Initiated {
			for f := 0; f < numFloors; f++ {
				if database[k].Elevator.Requests[f][0] {
					hallrequest[f][0] = true
				}
				if database[k].Elevator.Requests[f][1] {
					hallrequest[f][1] = true
				}
			}
		}
	}
	return hallrequest
}
