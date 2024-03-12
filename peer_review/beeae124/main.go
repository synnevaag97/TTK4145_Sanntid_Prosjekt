package main

import (
	"Driver-go/elevio"
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"fmt"
	"os"
)

func main() {

	//Default value of elevatorPort, can be modified with command-line passing
	elevatorPort := "15657"
	elevatorID := "1"
	fmt.Println("my pid: ", os.Getpid())
	//Checking for command-line passed arguments to be assigned to elevatorPort
	cmd_line_args := os.Args
	fmt.Println("Length cmd_arg:", len(cmd_line_args))
	if len(cmd_line_args) > 1 {
		fmt.Println(cmd_line_args[1])
		elevatorPort = cmd_line_args[1]
		fmt.Println("Id from os.Arg: ", cmd_line_args[2])
		elevatorID = cmd_line_args[2]
	}
	elevio.Init("localhost:"+elevatorPort, elevio.N_FLOORS)


	//Network UDP channels
	channel_peerUpdate 					:= make(chan peers.PeerUpdate)
	channel_peerTxEnable 				:= make(chan bool)
	channel_Tx_slaveState 				:= make(chan slaveStateMsg)
	channel_Rx_slaveState 				:= make(chan slaveStateMsg)
	channel_Tx_AllHallRequests 			:= make(chan [elevio.N_FLOORS][2]bool, 10)
	channel_Rx_AllHallRequests 			:= make(chan [elevio.N_FLOORS][2]bool, 10)
	channel_Tx_distribution 			:= make(chan SharedNodeInformation)
	channel_Rx_distribution 			:= make(chan SharedNodeInformation)
	channel_Tx_clear_request 			:= make(chan []Request)
	channel_Rx_clear_request 			:= make(chan []Request)
	channel_Tx_slave_hall_request 		:= make(chan Request)
	channel_Rx_slave_hall_request 		:= make(chan Request)
	channel_Tx_operational 				:= make(chan operationalStatusMsg)
	channel_Rx_operational 				:= make(chan operationalStatusMsg)

	//FSM channels
	channel_buttons_pushed 				:= make(chan elevio.ButtonEvent)
	channel_at_floor 					:= make(chan int)
	channel_obstruction_active 			:= make(chan bool)
	channel_node_connection_status 		:= make(chan map[string]bool, 1) 
	channel_update_shared_data 			:= make(chan SharedNodeInformation, 1)      
	channel_update_distributionMsg 		:= make(chan SharedNodeInformation, 1)
	channel_send_updated_master 		:= make(chan string, 1)
	channel_assign_hall_requests 		:= make(chan [elevio.N_FLOORS][2]bool, 1) 

	//Process pairs channels
	channel_processPairs_batonPass_1_2 	:= make(chan string, 1)
	channel_processPairs_batonPass_2_3 	:= make(chan string, 1)
	channel_processPairs_batonPass_3_4 	:= make(chan string, 1)
	channel_processPairs_batonPass_4_1	:= make(chan string, 1)

	//FSM go routines
	go elevio.PollButtons(channel_buttons_pushed)
	go elevio.PollFloorSensor(channel_at_floor)
	go elevio.PollObstructionSwitch(channel_obstruction_active)

	//Network go routines
	go peers.Transmitter(15764, elevatorID, channel_peerTxEnable)
	go peers.Receiver(15764, channel_peerUpdate)
	go bcast.Transmitter(16863,
		channel_Tx_slaveState,
		channel_Tx_slave_hall_request,
		channel_Tx_distribution,
		channel_Tx_clear_request,
		channel_Tx_operational,
		channel_Tx_AllHallRequests)

	go bcast.Receiver(16863,
		channel_Rx_slaveState,
		channel_Rx_slave_hall_request,
		channel_Rx_distribution,
		channel_Rx_clear_request,
		channel_Rx_operational,
		channel_Rx_AllHallRequests)

	//Process pairs
	go processPairs(
		elevatorPort,
		elevatorID,
		channel_processPairs_batonPass_1_2,
		channel_processPairs_batonPass_4_1)

	//Starting FSM
	go finiteStateMachine(
		elevatorID,
		channel_buttons_pushed,
		channel_at_floor,
		channel_obstruction_active,
		channel_assign_hall_requests,
		channel_node_connection_status,
		channel_update_shared_data,
		channel_processPairs_batonPass_1_2,
		channel_Rx_slaveState,
		channel_Rx_slave_hall_request,
		channel_Rx_clear_request,
		channel_Rx_operational,
		channel_Rx_AllHallRequests,
		channel_Tx_slaveState,
		channel_Tx_slave_hall_request,
		channel_Tx_clear_request,
		channel_Tx_operational,
		channel_Tx_AllHallRequests,
		channel_send_updated_master,
		channel_update_distributionMsg,
		channel_processPairs_batonPass_2_3)

	network_startupRoutine(
		channel_peerUpdate,
		channel_node_connection_status)

	go network_peerUpdate(
		elevatorID,
		channel_peerUpdate,
		channel_processPairs_batonPass_2_3,
		channel_processPairs_batonPass_3_4,
		channel_node_connection_status)

	go updateSharedDataAndAssignHallRequests(
		elevatorID,
		channel_send_updated_master,
		channel_processPairs_batonPass_3_4,
		channel_update_distributionMsg,
		channel_Rx_distribution,
		channel_Tx_distribution,
		channel_update_shared_data,
		channel_processPairs_batonPass_4_1,
		channel_assign_hall_requests)


	select {}
}