package request

import (
	"Elevator-go/elevio"
	cf "Elevator-go/type_"
)

func Requests_above(e cf.Elevator) bool {
	for f := e.Floor + 1; f < cf.NumFloors; f++ {
		for btn := range e.Requests[f] {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func Requests_below(e cf.Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := range e.Requests[f] {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func Requests_here(e cf.Elevator) bool {
	for btn := 0; btn < cf.NumButtons; btn++ {
		if e.Requests[e.Floor][btn] {
			return true
		}
	}
	return false
}

func Request_nextAction(e *cf.Elevator) cf.Action {
	switch e.Dir {
	case elevio.MD_Up:
		if Requests_above(*e) {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Up),
				Behav: cf.Moving,
			}
		} else if Requests_here(*e) {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Down),
				Behav: cf.DoorOpen,
			}
		} else if Requests_below(*e) {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Down),
				Behav: cf.Moving,
			}
		} else {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Stop),
				Behav: cf.Idle,
			}
		}
	case elevio.MD_Down:
		if Requests_below(*e) {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Down),
				Behav: cf.Moving,
			}
		} else if Requests_here(*e) {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Up),
				Behav: cf.DoorOpen,
			}
		} else if Requests_above(*e) {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Up),
				Behav: cf.Moving,
			}
		} else {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Stop),
				Behav: cf.Idle,
			}
		}
	case elevio.MD_Stop:
		if Requests_here(*e) {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Stop),
				Behav: cf.DoorOpen,
			}
		} else if Requests_above(*e) {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Up),
				Behav: cf.Moving,
			}
		} else if Requests_below(*e) {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Down),
				Behav: cf.Moving,
			}
		} else {
			return cf.Action{
				Dirin: cf.Direction(elevio.MD_Stop),
				Behav: cf.Idle,
			}
		}
	default:
		return cf.Action{
			Dirin: cf.Direction(elevio.MD_Stop),
			Behav: cf.Idle,
		}
	}
}

func Request_shouldStop(e *cf.Elevator) bool {
	switch e.Dir {
	case elevio.MD_Down:
		return e.Requests[e.Floor][int(elevio.BT_HallDown)] ||
			e.Requests[e.Floor][int(elevio.BT_Cab)] ||
			!Requests_below(*e)
	case elevio.MD_Up:
		return e.Requests[e.Floor][int(elevio.BT_HallUp)] ||
			e.Requests[e.Floor][int(elevio.BT_Cab)] ||
			!Requests_above(*e)
	case elevio.MD_Stop:
		fallthrough
	default:
		return true
	}
}

func Request_shouldClearImmediately(e *cf.Elevator, btn_floor int, btn_type elevio.ButtonType) bool {
	switch e.Econfig.ClearRequestVariant {
	case cf.CV_All:
		return e.Floor == btn_floor
	case cf.CV_InDirn:
		return e.Floor == btn_floor &&
			((e.Dir == elevio.MotorDirection(elevio.MD_Up) && btn_type == elevio.BT_HallUp) ||
				(e.Dir == elevio.MotorDirection(elevio.MD_Down) && btn_type == elevio.BT_HallDown) ||
				(e.Dir == elevio.MotorDirection(elevio.MD_Stop) ||
					btn_type == elevio.BT_Cab))

	default:
		return false
	}
}

func Request_clearAtCurrentFloor(e *cf.Elevator) cf.Elevator {
	switch e.Econfig.ClearRequestVariant {
	case cf.CV_All:
		for btn := 0; btn < cf.NumButtons; btn++ {
			e.Requests[e.Floor][btn] = false
		}

	case cf.CV_InDirn:
		e.Requests[e.Floor][elevio.BT_Cab] = false
		switch e.Dir {
		case elevio.MotorDirection(elevio.MD_Up):
			if !Requests_above(*e) && !e.Requests[e.Floor][elevio.BT_HallUp] {
				e.Requests[e.Floor][elevio.BT_HallDown] = false
			}
			e.Requests[e.Floor][elevio.BT_HallUp] = false
		case elevio.MotorDirection(elevio.MD_Down):
			if !Requests_below(*e) && !e.Requests[e.Floor][elevio.BT_HallDown] {
				e.Requests[e.Floor][elevio.BT_HallUp] = false
			}
			e.Requests[e.Floor][elevio.BT_HallDown] = false
		case elevio.MotorDirection(elevio.MD_Stop):
			fallthrough
		default:
			e.Requests[e.Floor][elevio.BT_HallUp] = false
			e.Requests[e.Floor][elevio.BT_HallDown] = false
		}
	default:
	}
	return *e
}
