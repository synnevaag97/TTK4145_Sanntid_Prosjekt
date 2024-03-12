package singleElevator

import (
	"Driver-go/elevio"
	"Types-go/Type-Elevator/elevator"
	"Types-go/Type-Msg/messages"
	"fmt"
)

const NumButtons int = 3

func RunElevator(
	numFloors int,
	elev *elevator.ElevatorState,
	port string,
	hall_request_to_distributer chan<- elevio.ButtonEvent,
	hall_request_completed_to_distributer chan<- elevio.ButtonEvent,
	cab_request_to_watchdog chan<- elevio.ButtonEvent,
	cab_request_completed chan<- messages.ReqCompleteMsg,
) {

	elevio.Init("localhost:"+port, numFloors)
	elevator.InitLights(numFloors)
	elev.InitELevatorState(numFloors)

	fmt.Println("Elevator state defined: ")
	elev.PrintELevatorState()

	//Define channels for driver
	button_pressed := make(chan elevio.ButtonEvent)
	new_floor := make(chan int)
	obstr := make(chan bool)
	new_request_to_elevator := make(chan elevio.ButtonEvent)

	go elevio.PollButtons(button_pressed)
	go elevio.PollFloorSensor(new_floor)
	go elevio.PollObstructionSwitch(obstr)
	go elev.PollNewCurrentRequest(new_request_to_elevator, numFloors)

	for {
		select {
		case button_event := <-button_pressed:
			if button_event.Button == elevio.BT_Cab {
				elev.AddRequest(int(button_event.Button), button_event.Floor)
				cab_request_to_watchdog <- button_event
			} else {
				hall_request_to_distributer <- button_event
				elevio.SetButtonLamp(button_event.Button, button_event.Floor, true)
			}

		case newFloor := <-new_floor:
			elevio.SetFloorIndicator(newFloor)
			elev.Floor = newFloor
			if elev.CheckRequestAtFloor() {
				if elev.Floor == elev.CurrentRequest.Floor {
					elev.RemoveRequest(elev.CurrentRequest)
					if elev.CurrentRequest.Button != elevio.BT_Cab {
						hall_request_completed_to_distributer <- elev.CurrentRequest
					} else {
						request_completed := messages.Create_ReqCompleteMsg("", elev.CurrentRequest)
						cab_request_completed <- request_completed
					}
				} else {
					req := elevio.ButtonEvent{}
					req.Floor = elev.Floor
					if elev.Requests[elev.Floor][0] && elev.Direction == elevio.MD_Up {
						req.Button = elevio.ButtonType(0)
						hall_request_completed_to_distributer <- req
						elev.RemoveRequest(req)
					} else if elev.Requests[elev.Floor][1] && elev.Direction == elevio.MD_Down {
						req.Button = elevio.ButtonType(1)
						hall_request_completed_to_distributer <- req
						elev.RemoveRequest(req)
					} else if elev.Requests[elev.Floor][2] {
						req.Button = elevio.ButtonType(2)
						elev.RemoveRequest(req)
						request_completed := messages.Create_ReqCompleteMsg("", req)
						cab_request_completed <- request_completed
					}
				}
				go elev.LoadingCab()

			}

		case obstr_changed := <-obstr:
			if elev.FLAG_doooOpen {
				if obstr_changed {
					fmt.Printf("Obstruction in door \n")
					elev.FLAG_obstruction = true
				} else {
					fmt.Printf("Obstruction removed \n")
					elev.FLAG_obstruction = false
					go elev.LoadingCab()
				}
			}

		case newReq := <-new_request_to_elevator:
			if elev.Floor == newReq.Floor {
				if newReq.Button == elevio.BT_Cab {
					elev.RemoveRequest(elev.CurrentRequest)
					request_completed := messages.Create_ReqCompleteMsg("", elev.CurrentRequest)
					cab_request_completed <- request_completed
				} else {
					elev.RemoveRequest(elev.CurrentRequest)
					hall_request_completed_to_distributer <- elev.CurrentRequest
				}
				go elev.LoadingCab()

			} else if elev.Floor < newReq.Floor {
				elev.Direction = elevio.MD_Up
				elev.State = "moving"
				elevio.SetMotorDirection(elev.Direction)

			} else if elev.Floor > newReq.Floor {
				elev.Direction = elevio.MD_Down
				elev.State = "moving"
				elevio.SetMotorDirection(elev.Direction)
			}

		}
	}
}
