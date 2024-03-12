package main

import (
	"Driver-go/elevio"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

type ElevState struct {
	Behaviour   	string               		`json:"behaviour"`
	Floor       	int                  		`json:"floor"`
	Direction   	string                		`json:"direction"`
	CabRequests 	[elevio.N_FLOORS]bool 		`json:"cabRequests"`
}

type HRAInput struct {
	HallRequests 	[elevio.N_FLOORS][2]bool 	`json:"hallRequests"`
	States       	map[string]ElevState  		`json:"states"`
}

func getElevState(elevator Elevator) ElevState {
	Behaviour := ""
	switch elevator.Behaviour {
	case EB_DoorOpen:
		Behaviour = "doorOpen"
	case EB_Moving:
		Behaviour = "moving"
	case EB_Idle:
		Behaviour = "idle"
	}

	Direction := ""
	switch elevator.Dirn {
	case elevio.D_Up:
		Direction = "up"
	case elevio.D_Down:
		Direction = "down"
	case elevio.D_Stop:
		Direction = "stop"
	}

	var CabRequests [elevio.N_FLOORS]bool

	for floor := 0; floor < elevio.N_FLOORS; floor++ {
		CabRequests[floor] = elevator.Requests[floor][2]
	}

	return ElevState{
		Behaviour,
		elevator.Floor,
		Direction,
		CabRequests,
	}
}


func hallRequestAssigner(
	elevator 							    Elevator,
	channel_update_distributionMsg 	chan<- 	SharedNodeInformation) 	Elevator {

	if !elevator.IsMaster {
		return elevator
	}

	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}


	activeNodes := make(map[string]ElevState)

	if !elevator.SharedData.NodeConnectionStatus[elevator.ID] {
		activeNodes[elevator.ID] = elevator.SharedData.States[elevator.ID]

		if !elevator.SharedData.NodeOperationalStatus[elevator.ID] {
			return elevator
		}

	} else{
		for node := range elevator.SharedData.NodeConnectionStatus {
			if elevator.SharedData.NodeConnectionStatus[node] &&
				elevator.SharedData.NodeOperationalStatus[node] {
				activeNodes[node] = elevator.SharedData.States[node]
			}
		}

		if len(activeNodes) == 0{
			//Avoids crashing when none of the elevators are connected or operational.
			//If none are connected, each elevator acts on its own and completes every order.
			//Note that you need to use a specific elevator's panel to send them hall calls,
			//since UDP is used to synchronise the panels.
			//If none of the elevators are operational, nothing will happen, but since the project
			//specifications says to assume that there is always at least one operational
			//elevator this is an acceptable flaw.
			activeNodes[elevator.ID] = elevator.SharedData.States[elevator.ID]

			if !elevator.SharedData.NodeOperationalStatus[elevator.ID] {
				return elevator
			}
		} 
	}

	input := HRAInput{
		HallRequests: elevator.SharedData.AllHallRequests,
		States:       activeNodes,
	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
	}

	cmd := exec.Command("utilities/"+hraExecutable, "-i", string(jsonBytes))
	ret, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + string(ret))
		return elevator
	}

	output := new(map[string][elevio.N_FLOORS][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
	}

	elevator.SharedData.HRAOutput = (*output)
	channel_update_distributionMsg <- elevator.SharedData
	
	return elevator
}