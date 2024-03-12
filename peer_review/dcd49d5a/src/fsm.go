package main

import (
	"Driver-go/elevio"
	"fmt"
)

var LocalElevator Elevator




func fsm_onInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.MD_Down)
	LocalElevator.Dirn = elevio.MD_Down
	LocalElevator.Behaviour = EB_Moving
	ActiveElevatorMap[LocalID] = LocalElevator
}

func fsm_onRequestButtonPress(btn_floor int, btn_type elevio.ButtonType) {
	fmt.Println("BUTTON PRESS", btn_floor, btn_type)
	switch LocalElevator.Behaviour {
	case EB_DoorOpen:
		if localOrders_shouldClearImmediately(LocalElevator, btn_floor, btn_type) {
			fmt.Println("should clear")
			DoorOpenTimer.Reset(DOOR_OPEN_TIME)
			DoorErrorTimer.Reset(DOOR_ERROR_TIME)
		} else {
			newElevators := elevatorMap_addNewOrder(btn_floor, btn_type, ActiveElevatorMap)

			fmt.Println("RequestElevators: ")
			elevators_print(newElevators)
			network_sendElevatorMapMessage(newElevators, MT_Normal)
		}

	case EB_Moving:
		newElevators := elevatorMap_addNewOrder(btn_floor, btn_type, ActiveElevatorMap)
		fmt.Println("RequestElevators: ")
		elevators_print(newElevators)
		network_sendElevatorMapMessage(newElevators, MT_Normal)

	case EB_Idle:
		newElevators := elevatorMap_addNewOrder(btn_floor, btn_type, ActiveElevatorMap)
		fmt.Println("RequestElevators: ")
		elevators_print(newElevators)
		network_sendElevatorMapMessage(newElevators, MT_Normal)

	}

}

func fsm_startIdleElevator() {

	switch LocalElevator.Behaviour {
	case EB_Idle:
		var a Action = localOrders_nextAction(LocalElevator)
		LocalElevator.Dirn = a.Dirn
		LocalElevator.Behaviour = a.Behaviour
		ActiveElevatorMap[LocalID] = LocalElevator
		switch a.Behaviour {
		case EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			DoorOpenTimer.Reset(DOOR_OPEN_TIME)
			DoorErrorTimer.Reset(DOOR_ERROR_TIME)

			LocalElevator = localOrders_clearAtCurrentFloor(LocalElevator)

			ActiveElevatorMap[LocalID] = LocalElevator

			ConfirmedHallOrders = elevatorMap_deleteConfirmedHallOrders(ActiveElevatorMap)
			fmt.Println("DELETE: ", ConfirmedHallOrders)
			lights_setAllHallLights(ConfirmedHallOrders)

			network_sendElevatorMapMessage(ActiveElevatorMap, MT_Delete)

		case EB_Moving:
			elevio.SetMotorDirection(LocalElevator.Dirn)
			if LocalElevator.Dirn != elevio.MD_Stop {
				FloorErrorTimer.Reset(FLOOR_ERROR_TIME)
			}

		case EB_Idle:
		}

	}

}

func fsm_onFloorArrival(newFloor int) {

	LocalElevator.Floor = newFloor
	ActiveElevatorMap[LocalID] = LocalElevator

	elevio.SetFloorIndicator(LocalElevator.Floor)

	switch LocalElevator.Behaviour {
	case EB_Moving:
		FloorErrorTimer.Stop()
		LocalElevator.Error = false

		if localOrders_shouldStop(LocalElevator) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			DoorOpenTimer.Reset(DOOR_OPEN_TIME)
			DoorErrorTimer.Reset(DOOR_ERROR_TIME)
			LocalElevator = localOrders_clearAtCurrentFloor(LocalElevator)

			lights_setAllCabLights(LocalElevator)
			LocalElevator.Behaviour = EB_DoorOpen

			ActiveElevatorMap[LocalID] = LocalElevator

			ConfirmedHallOrders = elevatorMap_deleteConfirmedHallOrders(ActiveElevatorMap)
			fmt.Println("DELETE: ", ConfirmedHallOrders)
			lights_setAllHallLights(ConfirmedHallOrders)

			network_sendElevatorMapMessage(ActiveElevatorMap, MT_Delete)

		} else {
			FloorErrorTimer.Reset(FLOOR_ERROR_TIME)
			ActiveElevatorMap[LocalID] = LocalElevator
			network_sendElevatorMapMessage(ActiveElevatorMap, MT_Normal)

		}
	default:
		fmt.Println("Floor arrivel when !EB_MOVING!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		elevator_print(LocalElevator)
		elevio.SetMotorDirection(elevio.MD_Stop)

	}

}

func fsm_onDoorTimeout() {
	switch LocalElevator.Behaviour {
	case EB_DoorOpen:
		var a Action = localOrders_nextAction(LocalElevator)
		LocalElevator.Dirn = a.Dirn
		LocalElevator.Behaviour = a.Behaviour
		deleteOrder := false

		switch LocalElevator.Behaviour {
		case EB_DoorOpen:
			DoorOpenTimer.Reset(DOOR_OPEN_TIME)
			LocalElevator = localOrders_clearAtCurrentFloor(LocalElevator)
			deleteOrder = true

		case EB_Moving:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(LocalElevator.Dirn)
			DoorErrorTimer.Stop()
			FloorErrorTimer.Reset(FLOOR_ERROR_TIME)

		case EB_Idle:
			elevio.SetDoorOpenLamp(false)
			DoorErrorTimer.Stop()
			elevio.SetMotorDirection(LocalElevator.Dirn)
		}

		ActiveElevatorMap[LocalID] = LocalElevator

		if deleteOrder {
			ConfirmedHallOrders = elevatorMap_deleteConfirmedHallOrders(ActiveElevatorMap)
			fmt.Println("DELETE: ", ConfirmedHallOrders)
			lights_setAllHallLights(ConfirmedHallOrders)
			network_sendElevatorMapMessage(ActiveElevatorMap, MT_Delete)
		} else {
			network_sendElevatorMapMessage(ActiveElevatorMap, MT_Normal)
		}

	default:
		break
	}

}
