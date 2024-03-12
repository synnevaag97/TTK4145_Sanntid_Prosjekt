package main

import (
	"Driver-go/elevio"
	"fmt"
)

func cost_assignOrder(btn_floor int, btn_type elevio.ButtonType, requestElevators map[string]Elevator) map[string]Elevator {

	testElevators := elevatorMap_copy(requestElevators)
	fastestTime := -1
	fastestElevator := requestElevators[LocalID]

	if btn_type == elevio.BT_Cab {
		fastestElevator.Orders[btn_floor][btn_type] = true
	} else {
		for _, elevator := range testElevators {
			if !elevator.Error {
				elevator.Orders[btn_floor][btn_type] = true
				time := cost_timeToIdle(elevator)
				if time < fastestTime || fastestTime == -1 {
					fastestElevator = elevator
					fastestTime = time
				}
			}
		}

	}
	requestElevators[fastestElevator.Id] = fastestElevator

	fmt.Println("Order given to: ", fastestElevator.Id)

	return requestElevators
}

func cost_timeToIdle(e Elevator) int {
	duration := 0

	switch e.Behaviour {
	case EB_Idle:
		e.Dirn = localOrders_nextAction(e).Dirn
		if e.Dirn == elevio.MD_Stop {
			return duration
		}
	case EB_Moving:
		duration += int(TRAVEL_TIME) / 2
		e.Floor += int(e.Dirn)
	case EB_DoorOpen:
		duration -= int(DOOR_OPEN_TIME) / 2
	}

	for {
		if localOrders_shouldStop(e) {
			e = localOrders_clearAtCurrentFloor(e)
			duration += int(DOOR_OPEN_TIME)
			e.Dirn = localOrders_nextAction(e).Dirn
			if e.Dirn == elevio.MD_Stop {
				return duration
			}
		}
		e.Floor += int(e.Dirn)
		duration += int(TRAVEL_TIME)
	}
}
