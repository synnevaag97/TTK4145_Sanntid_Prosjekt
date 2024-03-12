package main

import (
	"Network-go/network/peers"
	"fmt"
	"sort"
	"strconv"
	"time"
)

const ELEVATOR_ID_1 string = "-10"
const ELEVATOR_ID_2 string = "-9"
const ELEVATOR_ID_3 string = "-8"

var connectedNodes map[string]bool

func network_peerUpdate(
	elevatorID 									string,
	channel_peerUpdate 					<-chan 	peers.PeerUpdate,
	channel_processPairs_batonPass_2_3 	<-chan 	string,
	channel_processPairs_batonPass_3_4 	chan<- 	string,
	channel_node_connection_status 		chan<- 	map[string]bool,) {

	for {
		select{	
		case peerUpdate := <-channel_peerUpdate:
			if peerUpdate.New != "" && (peerUpdate.New == ELEVATOR_ID_1 || peerUpdate.New == ELEVATOR_ID_2 || peerUpdate.New == ELEVATOR_ID_3) {
				connectedNodes[peerUpdate.New] = true
			}
			for lost := range peerUpdate.Lost {
				if peerUpdate.Lost[lost] == ELEVATOR_ID_1 || peerUpdate.Lost[lost] == ELEVATOR_ID_2 || peerUpdate.Lost[lost] == ELEVATOR_ID_3 {
					connectedNodes[peerUpdate.Lost[lost]] = false
				}
			}
			channel_node_connection_status <- connectedNodes
			
		case processPairsBaton := <-channel_processPairs_batonPass_2_3:
			channel_processPairs_batonPass_3_4 <- processPairsBaton
		}	
	}	
}

func network_startupRoutine(
	channel_peerUpdate 				<-chan 	peers.PeerUpdate,
	channel_node_connection_status 	chan<- 	map[string]bool) {

	StartupTimer := time.NewTimer(5 * time.Second)
	StartupTimedOut := false
	connectedNodes = make(map[string]bool, N_TOTAL_ELEVATORS)
	var receivedIDs []string

	for !StartupTimedOut {
		select {
		case peerUpdate := <-channel_peerUpdate:
			receivedIDs = peerUpdate.Peers

		case <-StartupTimer.C:
			StartupTimedOut = true
		}
	}

	for k := range receivedIDs {
		if receivedIDs[k] == ELEVATOR_ID_1 || receivedIDs[k] == ELEVATOR_ID_2 || receivedIDs[k] == ELEVATOR_ID_3 {
			connectedNodes[receivedIDs[k]] = true
			fmt.Printf("CONNECTED NODES: ################# : ID [%v] status [%v]\n", receivedIDs[k], connectedNodes[receivedIDs[k]])
		}
	}

	time.Sleep(1 * time.Second)
	channel_node_connection_status <- connectedNodes
}

func network_chooseMaster(own_elevator Elevator) (bool, string) {
	node_conn_status := own_elevator.SharedData.NodeConnectionStatus
	var online_IDs []int

	for ID_string := range node_conn_status {
		if node_conn_status[ID_string] {
			ID_int, _ := strconv.Atoi(ID_string)
			online_IDs = append(online_IDs, ID_int)
		}
	}

	if len(online_IDs) <= 1 {
		own_elevator.IsMaster = true
		return own_elevator.IsMaster, own_elevator.ID
	}

	sort.Ints(online_IDs)
	ID_min_online := strconv.Itoa(online_IDs[0])

	if own_elevator.ID == ID_min_online {
		own_elevator.IsMaster = true
		fmt.Println("Turned into master! ID: ", own_elevator.ID)
	} else {
		own_elevator.IsMaster = false
		fmt.Println("NOT turned into master! ID: ", own_elevator.ID, own_elevator.IsMaster)
	}
	own_elevator.SharedData.MasterID = ID_min_online
	return own_elevator.IsMaster, ID_min_online
}
