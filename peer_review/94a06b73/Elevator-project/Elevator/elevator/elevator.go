package elevator

import (
	"Elevator-go/Elevator/elevio"
	"Elevator-go/Elevator/fsm"
	cf "Elevator-go/Elevator/type_"
	"time"
)

var LocalElevState *cf.Elevator

func Elevator(onRequestButtonPress chan elevio.ButtonEvent, ch_onFloorArrival chan int,
	ch_onStopButtonPress chan bool, ch_obstruction chan bool,
	ch_localElevatorStateToNtk chan cf.LocalElevatorState) {

	LocalElevState = fsm.Fsm_onInitElevator()

	doorTimer := time.NewTimer(time.Duration(cf.DoorOpenDuration) * time.Second)

	for {
		select {
		case buttonEvent := <-onRequestButtonPress:
			fsm.Fsm_onRequestButtonPress(LocalElevState, buttonEvent.Floor, buttonEvent.Button, doorTimer)
			/* fmt.Printf("Online  Elevators state:      %v\n", cf.OnlineElevatorsState) */

		case floor := <-ch_onFloorArrival:

			fsm.Fsm_onFloorArrival(LocalElevState, floor, doorTimer)

			/* the elevator should send its state at every floor */
			ch_localElevatorStateToNtk <- cf.LocalElevatorState{ /* does it cause deadlock when there is no network? */
				ElevatorState: cf.ElevBehavToTx{
					ElevFloor:    LocalElevState.Floor,
					Direcn:       LocalElevState.Dir,
					ElevRequests: LocalElevState.Requests,
					ElevBehav:    LocalElevState.Behave},
				ElevatorId: cf.LocalElevId}

			/* fmt.Printf("Online  Elevators state:      %v\n", cf.OnlineElevatorsState) */

		case <-doorTimer.C:
			fsm.Fsm_onDoorTimeout(LocalElevState, doorTimer)

			/* fmt.Printf("Online  Elevators state:      %v\n", cf.OnlineElevatorsState) */

		case stop := <-ch_onStopButtonPress: /* not functioning properly */
			if stop {
				doorTimer.Reset(time.Duration(cf.DoorOpenDuration) * time.Second)
				elevio.SetStopLamp(true)
			}
		case obstruction := <-ch_obstruction:
			if LocalElevState.Behave == cf.DoorOpen && obstruction {
				doorTimer.Reset(time.Duration(cf.DoorOpenDuration) * time.Second)
			}
		}
	}
}
