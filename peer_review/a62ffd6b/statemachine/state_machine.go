package statemachine

import (
	"Project/Driver-go/elevio"
	"Project/events"
	"Project/order"
)

func InitState(toIdleState chan bool) {
	toIdleState <- true
}

func StateHandler(toMoveState, toDoorState, toIdleState chan bool, floor *int) {
	for {
		select {
		case <-toMoveState:
			MoveState(toDoorState, toMoveState, floor)
		case <-toDoorState:
			OpenDoorState(toIdleState, floor)
		case <-toIdleState:
			IdleState(toMoveState, toDoorState, toIdleState, floor)
		}
	}

}

func MoveState(toDoorState, toMoveState chan bool, floor *int) {
	events.ElevatorState = events.Moving
	for {
		select {
		case <-events.AtFloor:
			if order.ElevatorOrders[*floor][elevio.BT_Cab] == 1 {
				toDoorState <- true
			} else if events.ElevatorDir == elevio.MD_Up && order.ElevatorOrders[*floor][elevio.BT_HallUp] == 1 {
				toDoorState <- true
			} else if events.ElevatorDir == elevio.MD_Down && order.ElevatorOrders[*floor][elevio.BT_HallDown] == 1 {
				toDoorState <- true
			} else {
				edgeCases(floor, toDoorState, toMoveState)
			}
			return
		}
	}

}

func edgeCases(floor *int, toDoorState, toMoveState chan bool) {
	if *floor == order.NumFloors-1 && order.PendingOrders(order.NumFloors, order.NumFloors) {
		events.ElevatorDir = elevio.MD_Down
		toDoorState <- true
	} else if *floor == 0 && order.PendingOrders(0, 1) {
		events.ElevatorDir = elevio.MD_Up
		toDoorState <- true
	} else if events.ElevatorDir == elevio.MD_Up && !order.PendingOrders(*floor+1, order.NumFloors) {
		events.ElevatorDir = elevio.MD_Down
		toDoorState <- true
	} else if events.ElevatorDir == elevio.MD_Down && !order.PendingOrders(0, *floor) {
		events.ElevatorDir = elevio.MD_Up
		toDoorState <- true
	} else {
		toMoveState <- true
	}
}

func OpenDoorState(toIdleState chan bool, floor *int) {
	events.ElevatorState = events.DoorOpen
	elevio.SetMotorDirection(elevio.MD_Stop)
	events.OpenDoor(floor)
	events.CloseDoor(toIdleState) // Wait 3 Seconds for door to close
}

func IdleState(toMoveState, toDoorState, toIdleState chan bool, floor *int) {
	events.ElevatorState = events.Idle
	switch events.ElevatorDir {
	case elevio.MD_Up:
		if order.PendingOrders(*floor+1, order.NumFloors) {
			elevio.SetMotorDirection(elevio.MD_Up)
			toMoveState <- true
		} else if order.PendingOrders(*floor, *floor+1) {
			events.ElevatorDir = elevio.MD_Down
			toDoorState <- true
		} else {
			events.ElevatorDir = elevio.MD_Down
			toIdleState <- true
		}
	case elevio.MD_Down:
		if order.PendingOrders(0, *floor) {
			elevio.SetMotorDirection(elevio.MD_Down)
			toMoveState <- true
		} else if order.PendingOrders(*floor, *floor+1) {
			events.ElevatorDir = elevio.MD_Up
			toDoorState <- true
		} else {
			events.ElevatorDir = elevio.MD_Up
			toIdleState <- true
		}
	case elevio.MD_Stop:
		events.ElevatorDir = events.ElevatorLastDir
		toDoorState <- true
	}

}
