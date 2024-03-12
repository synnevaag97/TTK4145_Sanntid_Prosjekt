package main

import (
	"Driver-go/elevio"
	"time"
)

const N_TOTAL_ELEVATORS int = 3

type ElevatorBehaviour int

const (
	EB_Idle = iota
	EB_DoorOpen
	EB_Moving
)

//Default is CV_InDir
type ClearRequestVariant int

const (
	// Assume everyone waiting for the elevator gets on the elevator, even if
	// they will be traveling in the "wrong" direction for a while
	CV_All = iota

	// Assume that only those that want to travel in the current direction
	// enter the elevator, and keep waiting outside otherwise
	CV_InDirn
)

type Config struct {
	clearRequestVariant 	ClearRequestVariant
	doorOpenDuration_s  	time.Duration
}

type Elevator struct {
	ID       				string
	IsMaster   				bool
	Floor    				int
	Dirn     				elevio.Dirn
	Requests 				[elevio.N_FLOORS][elevio.N_BUTTONS]bool
	Behaviour  				ElevatorBehaviour
	Config     				Config
	SharedData 				SharedNodeInformation
}

type SharedNodeInformation struct { //Shared information between different nodes
	MasterID              	string                              
	NodeConnectionStatus  	map[string]bool                     
	NodeOperationalStatus 	map[string]bool                     
	States                	map[string]ElevState             	`json:"states"`          //Input to HRA
	AllHallRequests       	[elevio.N_FLOORS][2]bool            `json:"allHallRequests"` //Input to HRA
	HRAOutput             	map[string][elevio.N_FLOORS][2]bool `json:"hraOutput"`       //Output from HRA
}


func elevator_unitialized(ID string) Elevator {
	configuration := Config{
		clearRequestVariant: CV_InDirn,
		doorOpenDuration_s:  3.0,
	}

	var requests_matrix 	[elevio.N_FLOORS][elevio.N_BUTTONS]bool
	var ElevStateunint 	ElevState

	elevator_uninit := Elevator{
		ID: 			ID,
		Floor:     		-1,
		IsMaster:    		false,
		Dirn:      		elevio.D_Stop,
		Requests:  		requests_matrix,
		Behaviour: 		EB_Idle,
		Config:    		configuration,
		SharedData: 	SharedNodeInformation{
			NodeConnectionStatus:  		map[string]bool{ID: true},
			NodeOperationalStatus: 		map[string]bool{ID: true},
			States:                		map[string]ElevState{ID: ElevStateunint},
			AllHallRequests:       		[elevio.N_FLOORS][2]bool{{false, false}},
		},
	}

	return elevator_uninit
}