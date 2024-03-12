package main

import (
	"Driver-go/elevio"
	"time"
)

type slaveStateMsg struct {
	SlaveID    string
	SlaveState ElevState
}

type operationalStatusMsg struct {
	SlaveID     string
	Operational bool
}


type OperationalWDTStatus int

const (
	OWDT_DEACTIVATED = iota
	OWDT_ACTIVATED
	OWDT_TIMEOUT
)




func finiteStateMachine(
	elevatorID 									string,
	channel_buttons_pushed 				<-chan 	elevio.ButtonEvent,
	channel_at_floor 					<-chan 	int,
	channel_obstruction_active 			<-chan 	bool,
	channel_assign_hall_requests 		<-chan 	[elevio.N_FLOORS][2]bool,
	channel_node_connection_status 		<-chan 	map[string]bool,
	channel_update_shared_data 			<-chan 	SharedNodeInformation,
	channel_processPairs_batonPass_1_2  <-chan 	string,
	channel_Rx_slaveState 				<-chan 	slaveStateMsg,
	channel_Rx_slave_hall_request 		<-chan 	Request,
	channel_Rx_clear_requests 			<-chan 	[]Request,
	channel_Rx_operational 				<-chan 	operationalStatusMsg,
	channel_Rx_AllHallRequests 			<-chan 	[elevio.N_FLOORS][2]bool,
	channel_Tx_slaveState 				chan<- 	slaveStateMsg,
	channel_Tx_slave_hall_request 		chan<- 	Request,
	channel_Tx_clear_requests 			chan<- 	[]Request,
	channel_Tx_operational 				chan<- 	operationalStatusMsg,
	channel_Tx_AllHallRequests 			chan<- 	[elevio.N_FLOORS][2]bool,
	channel_send_updated_master 		chan<- 	string,
	channel_update_distributionMsg 		chan<- 	SharedNodeInformation,
	channel_processPairs_batonPass_2_3 	chan<- 	string) {

	//timers
	doorTimedOut := time.NewTimer(3 * time.Second)
	doorTimedOut.Stop()

	operationalWDT := time.NewTimer(8 * time.Second)
	operationalWDT.Stop()
	var operationalWDTStatus OperationalWDTStatus
	operationalWDTStatus = OWDT_DEACTIVATED

	//Start-up of elevator
	elevator := fsm_elevatorStartup(
		elevatorID,
		channel_at_floor,
		channel_node_connection_status,
		channel_send_updated_master,
		channel_Tx_slaveState)

	for {
		select {

		case buttonPushed := <-channel_buttons_pushed:

			btnFloor := buttonPushed.Floor
			btnType := elevio.Button(buttonPushed.Button)

			switch elevator.Behaviour {

			case EB_Idle:
				
				if requests_shouldClearImmediately(elevator, btnFloor, btnType) {
					elevio.SetDoorOpenLamp(true)
					doorTimedOut.Reset(3 * time.Second)
					elevator.Behaviour = EB_DoorOpen

				} else if btnType == elevio.B_Cab {
					
					elevator.Requests[btnFloor][btnType] = true
					fsm_updateCabRequestLog(elevator.Requests)

					nextAction := requests_nextAction(elevator)

					elevator.Dirn = nextAction.Dirn
					elevator.Behaviour = nextAction.Behaviour

					if nextAction.Behaviour == EB_Moving {
						elevio.SetMotorDirection(nextAction.Dirn)
					}

					if elevator.IsMaster {break}

					//Slave should send new state to master
					slaveState := getElevState(elevator)
					channel_Tx_slaveState <- slaveStateMsg{elevator.ID, slaveState}
					

				} else if elevator.IsMaster {
					elevator.SharedData.AllHallRequests[btnFloor][btnType] = true
					elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)

				} else {
					//Send hall request to Master so they can distribute it to the right elevator
					channel_Tx_slave_hall_request <- Request{btnFloor, int(btnType)}
				}
				
			default:

				if btnType == elevio.B_Cab {

					elevator.Requests[btnFloor][btnType] = true
					fsm_updateCabRequestLog(elevator.Requests)

					if elevator.IsMaster {break}

					//Slave should send new state to master
					slaveState := getElevState(elevator)
					channel_Tx_slaveState <- slaveStateMsg{elevator.ID, slaveState}
				
				} else if elevator.IsMaster {
					
					elevator.SharedData.AllHallRequests[btnFloor][btnType] = true
					elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)

				} else {
					//Send hall request to Master so they can distribute it to the right elevator
					channel_Tx_slave_hall_request <- Request{btnFloor, int(btnType)}
				}
			}


		case atFloor := <-channel_at_floor:
			
			elevio.SetFloorIndicator(atFloor)
			elevator.Floor = atFloor

			switch elevator.Behaviour {
			case EB_Moving:
				if requests_shouldStop(elevator) {
					elevio.SetMotorDirection(elevio.D_Stop)
					elevator = fsm_clearRequests(elevator, channel_Tx_clear_requests)

					elevio.SetDoorOpenLamp(true)
					doorTimedOut.Reset(3 * time.Second)
					elevator.Behaviour = EB_DoorOpen
					setAllLights(elevator)
				}

			default:
				break
			}

			if elevator.IsMaster {
				elevator.SharedData.States[elevator.ID] = getElevState(elevator)
				elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)

			} else {
				slaveState := getElevState(elevator)
				channel_Tx_slaveState <- slaveStateMsg{elevator.ID, slaveState}
			}

			//Reset operational wd-timer when arriving at a new floor, if still orders at another floor
			if requests_areAbove(elevator) || requests_areBelow(elevator) {
				operationalWDT.Reset(8 * time.Second)
				operationalWDTStatus = OWDT_ACTIVATED
			} else {
				operationalWDT.Stop()
				operationalWDTStatus = OWDT_DEACTIVATED
			}
			if elevator.IsMaster {
				elevator.SharedData.NodeOperationalStatus[elevator.ID] = true
				elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)

			} else {
				channel_Tx_operational <- operationalStatusMsg{SlaveID: elevator.ID, Operational: true}
			}


		case newSlaveHallRequest := <-channel_Rx_slave_hall_request:
			//Master receives hall request from slave
		
			if !elevator.IsMaster {break}

			elevator.SharedData.AllHallRequests[newSlaveHallRequest.Floor][newSlaveHallRequest.BtnType] = true
			setAllLights(elevator)
			elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)


		case clearedHallRequests := <-channel_Rx_clear_requests:

			if !elevator.IsMaster {break}
			
			requestsHaveBeenCleared := false //Flag to make sure we don't run HRA unnecessarily if the requests have already been cleared
			for _, request := range clearedHallRequests {
				if elevator.SharedData.AllHallRequests[request.Floor][request.BtnType] {
					requestsHaveBeenCleared = true
				}
				elevator.SharedData.AllHallRequests[request.Floor][request.BtnType] = false
			}
			if requestsHaveBeenCleared {
				setAllLights(elevator)
				elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)
			}


		case updatedSharedData := <-channel_update_shared_data:
			//Slave receives updated system information from Master

			if elevator.IsMaster {break}
			
			elevator.SharedData = updatedSharedData


		case updatedSlaveState := <-channel_Rx_slaveState:
			//Master receives updated state information from slave

			if !elevator.IsMaster {break}
			
			elevator.SharedData.States[updatedSlaveState.SlaveID] = updatedSlaveState.SlaveState
			elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)


		case operationalStatus := <-channel_Rx_operational:
			//A slave has either become operational or unoperational, and lets the Master know

			if !elevator.IsMaster {break}

			elevator.SharedData.NodeOperationalStatus[operationalStatus.SlaveID] = operationalStatus.Operational
			elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)


		case nodeConnectionStatus := <-channel_node_connection_status:
			elevator.SharedData.NodeConnectionStatus = make(map[string]bool, N_TOTAL_ELEVATORS)
			for key, element := range nodeConnectionStatus {
				elevator.SharedData.NodeConnectionStatus[key] = element
			}

			if elevator.IsMaster {
				//Share all hall requests to a potentially new master
				channel_Tx_AllHallRequests <- elevator.SharedData.AllHallRequests
			}

			elevator.IsMaster, elevator.SharedData.MasterID = network_chooseMaster(elevator)
			channel_send_updated_master <- elevator.SharedData.MasterID

			//Merge all hall requests from other previous masters before running HRA
			merged_AllHallRequests := elevator.SharedData.AllHallRequests 

			for i := 0; i < len(channel_Rx_AllHallRequests); i++ {
				msg := <-channel_Rx_AllHallRequests
				for floor := 0; floor < elevio.N_FLOORS; floor++ {
					for button := 0; button < 2; button++ {
						if msg[floor][button] {
							merged_AllHallRequests[floor][button] = true
						}
					}
				}
			}

			if elevator.IsMaster {
				elevator.SharedData.AllHallRequests = merged_AllHallRequests
				elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)
				
			} else {
				//Send the Slave's state to Master over UDP
				slaveState := getElevState(elevator)
				channel_Tx_slaveState <- slaveStateMsg{elevator.ID, slaveState}
			}


		case assignedHallRequests := <-channel_assign_hall_requests:

			for floor := 0; floor < elevio.N_FLOORS; floor++ {
				elevator.Requests[floor][0] = assignedHallRequests[floor][0]
				elevator.Requests[floor][1] = assignedHallRequests[floor][1]
			}
			setAllLights(elevator)
			if elevator.Behaviour == EB_Idle{

				//The elevator isn't doing anything, so it should immediately start its next action.
				nextAction := requests_nextAction(elevator)
				
				elevator.Dirn = nextAction.Dirn
				elevator.Behaviour = nextAction.Behaviour

				switch nextAction.Behaviour {
				case EB_DoorOpen:
					elevio.SetDoorOpenLamp(true)
					doorTimedOut.Reset(3 * time.Second)
					elevator = fsm_clearRequests(elevator, channel_Tx_clear_requests)

					if !elevator.IsMaster {break}
					elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)

				case EB_Moving:
					elevio.SetMotorDirection(nextAction.Dirn)

				case EB_Idle:
					break
				}
			}


		case obstructionActive := <-channel_obstruction_active:
			if obstructionActive {
				operationalWDT.Stop()
				elevio.SetMotorDirection(elevio.D_Stop)
				doorTimedOut.Stop()
				elevio.SetDoorOpenLamp(true)

				if elevator.IsMaster {
					elevator.SharedData.NodeOperationalStatus[elevator.ID] = false
					elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)

				} else {
					channel_Tx_operational <- operationalStatusMsg{elevator.ID, false}
				}
				
			} else {
				operationalWDT.Reset(8 * time.Second)
				elevator.Behaviour = EB_DoorOpen
				doorTimedOut.Reset(3 * time.Second)
				
				if elevator.IsMaster {
					elevator.SharedData.NodeOperationalStatus[elevator.ID] = true
					elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)

				} else {
					channel_Tx_operational <- operationalStatusMsg{elevator.ID, true}
				}
			}


		case <-doorTimedOut.C:
			if elevator.Behaviour == EB_DoorOpen {

				nextAction := requests_nextAction(elevator)
				elevator.Dirn = nextAction.Dirn
				elevator.Behaviour = nextAction.Behaviour
				
				switch nextAction.Behaviour {

				case EB_DoorOpen:
					doorTimedOut.Reset(3 * time.Second)
					elevator = fsm_clearRequests(elevator, channel_Tx_clear_requests)

					if !elevator.IsMaster {break}
					elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)

				default:
					elevio.SetDoorOpenLamp(false)
					elevio.SetMotorDirection(elevator.Dirn)
				}
				
				fsm_updateCabRequestLog(elevator.Requests)
			}


		case <-operationalWDT.C:
		//The watchdog-timer that monitors if the elevator is operational has run out
			if elevator.IsMaster {
				elevator.SharedData.NodeOperationalStatus[elevatorID] = false
				elevator = hallRequestAssigner(elevator, channel_update_distributionMsg)
			} else {
				channel_Tx_operational <- operationalStatusMsg{SlaveID: elevatorID, Operational: false}
			}
			operationalWDTStatus = OWDT_TIMEOUT
			
		case processPairsBaton := <-channel_processPairs_batonPass_1_2:
			channel_processPairs_batonPass_2_3 <- processPairsBaton	
		}

		//Operational node wd-timer
		//Start a wd-timer if the node has orders above or below the current floor
		if (requests_areAbove(elevator) || requests_areBelow(elevator)) && (operationalWDTStatus == OWDT_DEACTIVATED) {
			operationalWDT.Reset(8 * time.Second)
			operationalWDTStatus = OWDT_ACTIVATED
		}

		//Self-diagnostics, try to move to a different floor to check motor operationality
		if operationalWDTStatus == OWDT_TIMEOUT {
			if elevator.Floor < 2 {
				//drive up
				elevator.Dirn = elevio.D_Up
				elevator.Behaviour = EB_Moving
				elevio.SetMotorDirection(elevio.D_Up)
			} else {
				//drive down
				elevator.Dirn = elevio.D_Down
				elevator.Behaviour = EB_Moving
				elevio.SetMotorDirection(elevio.D_Down)
			}
		}
	}
}

func setAllLights(elevator Elevator) {
	for floor := 0; floor < elevio.N_FLOORS; floor++ {
		for btn := elevio.ButtonType(0); btn < 3; btn++ {
			if btn == elevio.BT_Cab {
				elevio.SetButtonLamp(btn, floor, elevator.Requests[floor][btn])
			} else {
				elevio.SetButtonLamp(btn, floor, elevator.SharedData.AllHallRequests[floor][btn])
			}
		}
	}
}

func fsm_onInitBetweenFloors(elevator Elevator) Elevator {
	elevio.SetMotorDirection(elevio.D_Down)
	elevator.Dirn = elevio.D_Down
	elevator.Behaviour = EB_Moving
	return elevator
}

func fsm_elevatorStartup(
	elevatorID 								string,
	channel_at_floor				<-chan 	int,
	channel_node_connection_status 	<-chan 	map[string]bool,
	channel_send_updated_master 	chan<- 	string,
	channel_Tx_slaveState 			chan<- 	slaveStateMsg) Elevator {

	elevator := elevator_unitialized(elevatorID)
	
	elevator = fsm_onInitBetweenFloors(elevator)
	floor := <-channel_at_floor
	elevio.SetFloorIndicator(floor)
	elevator.Floor = floor

	elevio.SetMotorDirection(elevio.D_Stop)
	elevator.Behaviour = EB_Idle
	elevator.Dirn = elevio.D_Stop

	elevator.SharedData.NodeConnectionStatus = <-channel_node_connection_status
	for node := range elevator.SharedData.NodeConnectionStatus {
		elevator.SharedData.NodeOperationalStatus[node] = true
	}

	elevator.IsMaster, elevator.SharedData.MasterID = network_chooseMaster(elevator)
	channel_send_updated_master <- elevator.SharedData.MasterID

	if elevator.IsMaster {
		//Initialize the elevator states in SharedData with dummy-values
		for key := range elevator.SharedData.NodeConnectionStatus {
			elevator.SharedData.States[key] = ElevState{
				Behaviour:   "idle",
				Floor:       0,
				Direction:   "stop",
				CabRequests: [elevio.N_FLOORS]bool{false, false, false, false}}
		}
		//Give the state belonging to the master the correct value
		elevator.SharedData.States[elevator.ID] = getElevState(elevator)

	} else {
		//Broadcast slave's state to master
		slaveState := getElevState(elevator)
		channel_Tx_slaveState <- slaveStateMsg{elevator.ID, slaveState}
	}

	elevator.Requests = fsm_recoverCabRequestsLog(elevator.Requests)
	setAllLights(elevator)
	elevio.SetDoorOpenLamp(false)
	return elevator
}

func fsm_clearRequests(elevator Elevator, channel_Tx_clear_request chan<- []Request) Elevator {
	var clearedHallRequests []Request
	elevator, clearedHallRequests = requests_clearAtCurrentFloor(elevator)
	
	if elevator.IsMaster {
		for _, request := range clearedHallRequests {
			elevator.SharedData.AllHallRequests[elevator.Floor][request.BtnType] = false
			setAllLights(elevator)
		}
	} else if len(clearedHallRequests) > 0 {
		//Send to Master
		channel_Tx_clear_request <- clearedHallRequests
		//Multiple sendings as a packet loss counter-measure
		channel_Tx_clear_request <- clearedHallRequests
		channel_Tx_clear_request <- clearedHallRequests
		channel_Tx_clear_request <- clearedHallRequests
	}
	return elevator
}

func fsm_updateCabRequestLog(requests [elevio.N_FLOORS][elevio.N_BUTTONS]bool) {
	var cabRequests[elevio.N_FLOORS]bool
	for floor := 0; floor < elevio.N_FLOORS; floor++ {
		cabRequests[floor] = requests[floor][elevio.B_Cab]
	}
	writeCabRequestLog(cabRequests)
}

func fsm_recoverCabRequestsLog(requests [elevio.N_FLOORS][elevio.N_BUTTONS]bool) [elevio.N_FLOORS][elevio.N_BUTTONS]bool{
	cabRequests:= readCabRequestLog()
	for floor := 0; floor < elevio.N_FLOORS; floor++ {
		requests[floor][elevio.B_Cab] = cabRequests[floor] 
	}	
	return requests	
}