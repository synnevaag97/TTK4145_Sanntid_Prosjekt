package elevator

import (
	"Driver-go/elevio"
	"Types-go/Type-Msg/messages"
	"Types-go/Type-Node/node"
	"fmt"
	"time"
)

const NumButtons int = 3

type ElevatorState struct {
	State            node.States
	Floor            int
	Direction        elevio.MotorDirection
	Requests         [][NumButtons]bool
	CurrentRequest   elevio.ButtonEvent
	FLAG_obstruction bool
	FLAG_doooOpen    bool
}

//Initialization of the elevator defining all the default states
func (elev *ElevatorState) InitELevatorState(floors int) {
	elev.FLAG_doooOpen = false
	elev.FLAG_obstruction = false
	elev.CurrentRequest = elevio.ButtonEvent{}
	elev.CurrentRequest.Floor = -1
	elev.CurrentRequest.Button = elevio.BT_HallUp
	elev.Direction = elevio.MD_Stop
	elev.Requests = make([][NumButtons]bool, floors)
	for f := 0; f < floors; f++ {
		for b := 0; b < NumButtons; b++ {
			elev.Requests[f][b] = false
		}
	}

	floor := elevio.GetFloor()
	if floor == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		for {
			floor = elevio.GetFloor()
			if floor != -1 {
				elevio.SetMotorDirection(elevio.MD_Stop)
				break
			}
		}
	}
	elev.State = "idle"
	elev.Floor = floor
	elevio.SetFloorIndicator(floor)
}

//Initialization of all the status light of the elevator to false
func InitLights(numFloors int) {
	for f := 0; f < numFloors; f++ {
		for b := 0; b < NumButtons; b++ {
			elevio.SetButtonLamp(elevio.ButtonType(b), f, false)
		}
	}
}

//Print the elevator state
func (elevator *ElevatorState) PrintELevatorState() {
	fmt.Printf("Elevator State\n")
	fmt.Printf("|------------------------|\n")
	fmt.Printf("|  State = %s  			 \n", elevator.State)
	fmt.Printf("|  Floor = %d  			 \n", elevator.Floor)
	fmt.Printf("|  Direction = %d  		 \n", elevator.Direction)
	fmt.Printf("|  Current request = %d  		 \n", elevator.CurrentRequest)
	fmt.Printf("|------------------------|\n")
}

//print the request database
func (elev *ElevatorState) PrintRequestDatabase() {
	fmt.Printf("\n")
	fmt.Printf("Request Database\n")
	fmt.Printf("|-------------------------------------------|\n")
	fmt.Printf("|Buttons\\Floor	|  1        2      3      4 |\n")
	fmt.Printf("|-------------------------------------------|\n")
	fmt.Printf("|Hall up 	| %t  %t  %t  %t|\n", elev.Requests[0][0], elev.Requests[1][0], elev.Requests[2][0], elev.Requests[3][0])
	fmt.Printf("|Hall down	| %t  %t  %t  %t|\n", elev.Requests[0][1], elev.Requests[1][1], elev.Requests[2][1], elev.Requests[3][1])
	fmt.Printf("|Cab 		| %t  %t  %t  %t|\n", elev.Requests[0][2], elev.Requests[1][2], elev.Requests[2][2], elev.Requests[3][2])
	fmt.Printf("|-------------------------------------------|\n")
}

//Remove a request from the elevator
func (elev *ElevatorState) RemoveRequest(request elevio.ButtonEvent) {
	elev.Requests[request.Floor][int(request.Button)] = false
	elevio.SetButtonLamp(request.Button, request.Floor, false)
	elev.Requests[request.Floor][2] = false
	elevio.SetButtonLamp(elevio.BT_Cab, request.Floor, false)

}

//Add a request to the elevator
func (elev *ElevatorState) AddRequest(button int, floor int) {
	elevio.SetButtonLamp(elevio.ButtonType(button), floor, true)
	elev.Requests[floor][button] = true
}

//Poll the new current request
func (elev *ElevatorState) PollNewCurrentRequest(request chan<- elevio.ButtonEvent, numFloors int) {
	btypes := elevio.ButtonEvent{}
	for {
		if elev.CurrentRequest.Floor == -1 {
			for b := 0; b < NumButtons; b++ {
				if elev.Requests[elev.Floor][b] && elev.CurrentRequest.Floor == -1 {
					elev.CurrentRequest.Floor = elev.Floor
					elev.CurrentRequest.Button = elevio.ButtonType(b)
					btypes.Floor = elev.Floor
					btypes.Button = elevio.ButtonType(b)
					request <- btypes
				}
			}
		}
		if elev.CurrentRequest.Floor == -1 {
			if elev.Floor < 2 {
				for f := 3; f >= 0; f-- {
					for b := 0; b < NumButtons; b++ {
						if elev.Requests[f][b] && elev.CurrentRequest.Floor == -1 {
							elev.CurrentRequest.Floor = f
							elev.CurrentRequest.Button = elevio.ButtonType(b)
							btypes.Floor = f
							btypes.Button = elevio.ButtonType(b)
							request <- btypes
						}
					}
				}

			} else if elev.Floor >= 2 {
				for f := 0; f < numFloors; f++ {
					for b := 0; b < NumButtons; b++ {
						if elev.Requests[f][b] && elev.CurrentRequest.Floor == -1 {
							elev.CurrentRequest.Floor = f
							elev.CurrentRequest.Button = elevio.ButtonType(b)
							btypes.Floor = f
							btypes.Button = elevio.ButtonType(b)
							request <- btypes
						}
					}
				}
			}
		}
	}
}

//Check if there is a request at the current floor
func (elev *ElevatorState) CheckRequestAtFloor() bool {
	if elev.Floor == elev.CurrentRequest.Floor {
		return true
	} else if elev.Requests[elev.Floor][2] {
		return true
	} else if elev.Direction == 1 {
		if elev.Requests[elev.Floor][0] {
			return true
		}
	} else if elev.Direction == -1 {
		if elev.Requests[elev.Floor][1] {
			return true
		}
	}
	return false
}

//Loading cab function
func (elev *ElevatorState) LoadingCab() {
	elevio.SetMotorDirection(elevio.MD_Stop)
	elev.Direction = elevio.MD_Stop

	elevio.SetDoorOpenLamp(true)
	elev.State = "doorOpen"
	elev.FLAG_doooOpen = true

	time.Sleep(3 * time.Second)

	if !elev.FLAG_obstruction {
		elevio.SetDoorOpenLamp(false)
		elev.FLAG_doooOpen = false

		if elev.Floor == elev.CurrentRequest.Floor {
			elev.CurrentRequest.Floor = -1
			elev.State = "idle"
		} else {
			if elev.CurrentRequest.Floor > elev.Floor {
				elevio.SetMotorDirection(elevio.MD_Up)
			} else {
				elevio.SetMotorDirection(elevio.MD_Down)
			}
			elev.State = "moving"
		}
	}
}

func (elev *ElevatorState) FetchHallChanges(numFloors int, Id string, database *map[string]node.NetworkNode) {
	// Update elevator with hall changes in database.
	for f := 0; f < numFloors; f++ {
		elev.Requests[f][0] = (*database)[Id].Elevator.Requests[f][0]
		elev.Requests[f][1] = (*database)[Id].Elevator.Requests[f][1]
	}
}

func (elev *ElevatorState) FetchCabChanges(numFloors int, Id string, database *map[string]node.NetworkNode) {
	// Update elevator with cab data
	for f := 0; f < numFloors; f++ {
		elev.Requests[f][2] = (*database)[Id].Elevator.Requests[f][2]
	}
}

func (elev *ElevatorState) FetchLights(numFloors int, Id string, database *map[string]node.NetworkNode) {
	// Update hall lights
	for k := range *database {
		if k != Id {
			for f := 0; f < numFloors; f++ {
				for b := 0; b < 2; b++ {
					if (*database)[k].Elevator.Requests[f][b] {
						elevio.SetButtonLamp(elevio.ButtonType(b), f, true)
					}
				}
			}
		}
	}

	// Update cab lights
	for f := 0; f < numFloors; f++ {
		if elev.Requests[f][2] {
			elevio.SetButtonLamp(elevio.ButtonType(2), f, true)
		}
	}
}

func (elev *ElevatorState) PollElevatorChanges(numFloors int, Id string, elev_changes_channel chan<- messages.UpdateElevMsg, elev_changes_from_polling chan<- messages.UpdateElevMsg, database *map[string]node.NetworkNode) {
	// Poll changes of this elevator and add to database.
	for {
		const pollRate = 300 * time.Millisecond

		update := false

		if thisElevator, ok := (*database)[Id]; ok {

			if thisElevator.Elevator.Floor != elev.Floor {
				thisElevator.Elevator.Floor = elev.Floor
				update = true
			}

			if thisElevator.Elevator.Direction != elev.Direction {
				thisElevator.Elevator.Direction = elev.Direction
				update = true
			}

			if thisElevator.Elevator.State != elev.State {
				thisElevator.Elevator.State = elev.State
				update = true
			}

			for f := 0; f < numFloors; f++ {
				if thisElevator.Elevator.Requests[f][2] != elev.Requests[f][2] {
					thisElevator.Elevator.Requests[f][2] = elev.Requests[f][2]
					update = true
				}
			}
			(*database)[Id] = thisElevator
		}
		if update {
			elevMsg := messages.Create_UpdateElevMsg(Id, (*database)[Id].Elevator)
			elev_changes_channel <- elevMsg
			update = false
		}

		time.Sleep(pollRate)
	}
}
