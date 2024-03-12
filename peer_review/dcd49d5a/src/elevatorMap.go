package main

import (
	"Driver-go/elevio"
	"fmt"
	"time"
)

var ActiveElevatorMap = make(map[string]Elevator)
var ConfirmedHallOrders [N_FLOORS][N_BUTTONS]bool

const TRAVEL_TIME time.Duration = 2 * time.Second

/*
func getElevators() map[string]Elevator {
	elevMtx.Lock()
	defer elevMtx.Unlock()
	return Elevators
}

func setElevators(elev map[string]Elevator) {
	elevMtx.Lock()
	defer elevMtx.Unlock()
	E
}
*/

func elevatorMap_init() {
	LocalElevator.Id = LocalID
	LocalElevator.Error = false

	lights_setAllHallLights(LocalElevator.Orders)
	elevio.SetDoorOpenLamp(false)

	fsm_onInitBetweenFloors()

	ActiveElevatorMap[LocalID] = LocalElevator
	network_sendElevatorMapMessage(ActiveElevatorMap, MT_Init)
}

func elevatorMap_allElevatorsExists(newElevators map[string]Elevator, localElevators map[string]Elevator) bool {
	elevatorExists := true

	for id := range newElevators {
		_, exists := localElevators[id]
		if !exists {
			elevatorExists = false
		}
	}

	return elevatorExists
}

func elevatorMap_addOrdersToConfirmedOrders(senderLights [N_FLOORS][N_BUTTONS]bool,
	confirmedOrders [N_FLOORS][N_BUTTONS]bool) [N_FLOORS][N_BUTTONS]bool {

	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS-1; btn++ {
			confirmedOrders[floor][btn] = confirmedOrders[floor][btn] || senderLights[floor][btn]
		}
	}
	return confirmedOrders
}

func elevatorMap_deleteConfirmedHallOrders(elevatorMap map[string]Elevator) [N_FLOORS][N_BUTTONS]bool {
	var hallOrders [N_FLOORS][N_BUTTONS]bool

	for _, elevator := range elevatorMap {
		for floor := 0; floor < N_FLOORS; floor++ {
			for btn := 0; btn < N_BUTTONS-1; btn++ {
				hallOrders[floor][btn] = hallOrders[floor][btn] || elevator.Orders[floor][btn]
			}
		}
	}

	return hallOrders

}

func elevatorMap_update(newElevatorMap map[string]Elevator, localElevatorMap map[string]Elevator) map[string]Elevator {

	updatedElevatorMap := newElevatorMap

	for id, elevator := range updatedElevatorMap {
		if id == LocalID {
			elevator.Behaviour = localElevatorMap[id].Behaviour
			elevator.Dirn = localElevatorMap[id].Dirn
			//elevator.Error = localElevatorMap[id].Error
			elevator.Floor = localElevatorMap[id].Floor
		}
		updatedElevatorMap[id] = elevator
	}
	return updatedElevatorMap
}

func elevatorMap_handleIncomingMessage(receivedElevatorMapMessage elevatorMapMessage) {
	receivedElevatorMap := receivedElevatorMapMessage.ElevatorMap
	senderID := receivedElevatorMapMessage.Id
	message := receivedElevatorMapMessage.Message

	fmt.Println("receive ID ", senderID)

	switch message {
	case MT_Init:
		fmt.Println("INIT")
		if senderID != LocalID {

			if elevatorMap_allElevatorsExists(receivedElevatorMap, ActiveElevatorMap) {
				elevator := receivedElevatorMap[senderID]
				elevator.Orders = ActiveElevatorMap[senderID].Orders

				ActiveElevatorMap[senderID] = elevator
			} else {
				ActiveElevatorMap[senderID] = receivedElevatorMap[senderID]
			}
			network_sendElevatorMapMessage(ActiveElevatorMap, MT_Normal)
		}
	case MT_Delete:
		ConfirmedHallOrders = elevatorMap_deleteConfirmedHallOrders(receivedElevatorMap)
		fmt.Println("DELETE: ", ConfirmedHallOrders)
		lights_setAllHallLights(ConfirmedHallOrders)

		ActiveElevatorMap = elevatorMap_update(receivedElevatorMap, ActiveElevatorMap)

		network_sendElevatorMapMessage(ActiveElevatorMap, MT_Ack)
	case MT_Normal:
		fmt.Println("Normal")
		ConfirmedHallOrders = elevatorMap_addOrdersToConfirmedOrders(receivedElevatorMap[senderID].Orders, ConfirmedHallOrders)
		ConfirmedHallOrders = elevatorMap_addOrdersToConfirmedOrders(receivedElevatorMap[LocalID].Orders, ConfirmedHallOrders)
		lights_setAllHallLights(ConfirmedHallOrders)

		ActiveElevatorMap = elevatorMap_update(receivedElevatorMap, ActiveElevatorMap)
		network_sendElevatorMapMessage(ActiveElevatorMap, MT_Ack)

	case MT_Ack:
		fmt.Println("Ack")
		ConfirmedHallOrders = elevatorMap_addOrdersToConfirmedOrders(receivedElevatorMap[senderID].Orders, ConfirmedHallOrders)
		ConfirmedHallOrders = elevatorMap_addOrdersToConfirmedOrders(receivedElevatorMap[LocalID].Orders, ConfirmedHallOrders)

		lights_setAllHallLights(ConfirmedHallOrders)
	case MT_Error:
		fmt.Println("Redistributing orders")
		ActiveElevatorMap = elevatorMap_update(receivedElevatorMap, ActiveElevatorMap)
		ActiveElevatorMap = elevatorMap_redistributeOrders(ActiveElevatorMap)
		network_sendElevatorMapMessage(ActiveElevatorMap, MT_Normal)
	}

	elevators_print(ActiveElevatorMap)

	LocalElevator.Orders = ActiveElevatorMap[LocalID].Orders
	LocalElevator.Error = ActiveElevatorMap[LocalID].Error

	fsm_startIdleElevator()
}

func elevatorMap_copy(elevatorMap map[string]Elevator) map[string]Elevator {
	copiedElevatorMap := make(map[string]Elevator)

	for id, elevator := range elevatorMap {
		copiedElevatorMap[id] = elevator
	}
	return copiedElevatorMap

}

func elevatorMap_addNewOrder(btn_floor int, btn_type elevio.ButtonType, elevatorMap map[string]Elevator) map[string]Elevator {

	updatedElevatorMap := elevatorMap_copy(elevatorMap)

	updatedElevatorMap = cost_assignOrder(btn_floor, btn_type, updatedElevatorMap)

	if btn_type == elevio.BT_Cab {
		ActiveElevatorMap = updatedElevatorMap
		lights_setAllCabLights(updatedElevatorMap[LocalID])
	}

	return updatedElevatorMap
}

func elevatorMap_redistributeOrders(elevatorMap map[string]Elevator) map[string]Elevator {
	for id, elevator := range elevatorMap {
		if elevator.Error {
			for floor := 0; floor < N_FLOORS; floor++ {
				for button := 0; button < N_BUTTONS-1; button++ {
					if elevator.Orders[floor][button] {
						fmt.Println("Redistributing ", floor, button, "from ", elevator.Id)

						elevator.Orders[floor][button] = false
						elevatorMap[id] = elevator
						elevatorMap = elevatorMap_addNewOrder(floor, elevio.ButtonType(button), elevatorMap)
					}
				}
			}
		}
	}
	return elevatorMap
}
