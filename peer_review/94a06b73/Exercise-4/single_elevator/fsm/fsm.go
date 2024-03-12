package fsm

import (
	"Elevator-go/elevio"
	"Elevator-go/request"
	cf "Elevator-go/type_"
	"time"
)

func SetAllLights(e *cf.Elevator) {
	for floor := 0; floor < cf.NumFloors; floor++ {
		for btn := 0; btn < cf.NumButtons; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, e.Requests[floor][btn])
		}
	}
}

func Fsm_onInitElevator() *cf.Elevator {
	requests := make([][]bool, 0)
	for floor := 0; floor < cf.NumFloors; floor++ {
		requests = append(requests, make([]bool, cf.NumButtons))
		for button := range requests[floor] {
			requests[floor][button] = false
		}
	}

	elevet := cf.Elevator{
		//Floor:    ,
		Dir:      elevio.MD_Stop,
		Requests: requests,
		Behave:   cf.Idle,
		Econfig:  cf.Config{ClearRequestVariant: cf.CV_All, TimerCount: 0}}
	return &elevet
}

func Fsm_onRequestButtonPress(e *cf.Elevator, btn_floor int, btn_type elevio.ButtonType, doorTimer *time.Timer) {
	switch e.Behave {
	case cf.DoorOpen:
		if request.Request_shouldClearImmediately(e, btn_floor, btn_type) {
			doorTimer.Reset(time.Duration(3) * time.Second)
		} else {
			e.Requests[btn_floor][btn_type] = true
		}
	case cf.Moving:
		e.Requests[btn_floor][btn_type] = true
	case cf.Idle:
		e.Requests[btn_floor][btn_type] = true
		a := request.Request_nextAction(e)
		e.Dir = elevio.MotorDirection(a.Dirin)
		e.Behave = a.Behav
		switch a.Behav {
		case cf.DoorOpen:
			elevio.SetDoorOpenLamp(true)
			doorTimer.Reset(time.Duration(3) * time.Second)
			*e = request.Request_clearAtCurrentFloor(e)
		case cf.Moving:
			elevio.SetMotorDirection(e.Dir)
		case cf.Idle:

		}
	}
	SetAllLights(e)
}

func Fsm_onFloorArrival(e *cf.Elevator, floor int, doorTimer *time.Timer) {
	e.Floor = floor
	elevio.SetFloorIndicator(e.Floor)
	switch e.Behave {
	case cf.Moving:
		if request.Request_shouldStop(e) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			*e = request.Request_clearAtCurrentFloor(e)
			doorTimer.Reset(time.Duration(cf.DoorOpenDuration) * time.Second)
			e.Behave = cf.DoorOpen
		}
	}
}

func Fsm_onDoorTimeout(e *cf.Elevator, doorTimer *time.Timer) {
	switch e.Behave {
	case cf.DoorOpen:
		a := request.Request_nextAction(e)
		e.Dir = elevio.MotorDirection(a.Dirin)
		e.Behave = a.Behav

		switch e.Behave {
		case cf.DoorOpen:
			doorTimer.Reset(time.Duration(cf.DoorOpenDuration) * time.Second)
			*e = request.Request_clearAtCurrentFloor(e)
			SetAllLights(e)
		case cf.Moving:
			fallthrough
		case cf.Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(e.Dir)

		}
	}
}
