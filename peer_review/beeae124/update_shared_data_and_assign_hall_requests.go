package main

import (
	"Driver-go/elevio"
	"time"
)

func updateSharedDataAndAssignHallRequests(
	elevatorID 									string,
	channel_send_updated_master 		<-chan 	string,
	channel_processPairs_batonPass_3_4 	<-chan 	string,
	channel_update_distributionMsg		<-chan 	SharedNodeInformation,
	channel_Rx_distribution 			<-chan 	SharedNodeInformation,
	channel_Tx_distribution 			chan<- 	SharedNodeInformation,
	channel_update_shared_data 			chan<- 	SharedNodeInformation,
	channel_processPairs_batonPass_4_1 	chan<- 	string,
	channel_assign_hall_requests        chan<-  [elevio.N_FLOORS][2]bool) {

	//Initialization of distributionMsg
	var distributionMsg 			SharedNodeInformation
	var distributionMsg_received 	SharedNodeInformation

	//Create tickers
	distributionMsgTicker := time.NewTicker(120 * time.Millisecond)
	distributionMsgTicker.Stop()
	hallRequestAssignmentTicker := time.NewTicker(250 * time.Millisecond)

	//Choosing initial master
	MasterID := <-channel_send_updated_master

	for {
		if len(channel_send_updated_master) > 0 {   //If len(channel) is in theory a race condition, but the channel is not accessed any other place
			MasterID = <-channel_send_updated_master
		}
		
		if elevatorID != MasterID {
			distributionMsgTicker.Stop()
		}

		select {
		case distributionMsg = <-channel_update_distributionMsg:
			if elevatorID != MasterID {
				break
			}

			//Start ticker
			distributionMsgTicker.Reset(120 * time.Millisecond)

		case <-distributionMsgTicker.C:
			if elevatorID != MasterID {
				break
			}

			channel_Tx_distribution <- distributionMsg

		case distributionMsg_received = <-channel_Rx_distribution:
			if elevatorID == MasterID {
				break
			}

			//Create a copy of the updated SharedNodeInformation (distributionMsg_received), for sending on channel
			updatedSNI_copy := copy_SharedNodeInformation(distributionMsg_received)

			channel_update_shared_data <- updatedSNI_copy

		case <-hallRequestAssignmentTicker.C:
			if elevatorID == MasterID {
				channel_assign_hall_requests <- distributionMsg.HRAOutput[elevatorID]
			} else {
				channel_assign_hall_requests <- distributionMsg_received.HRAOutput[elevatorID]
			}

		case processPairsBaton := <-channel_processPairs_batonPass_3_4:
			channel_processPairs_batonPass_4_1 <- processPairsBaton
		}		
	}
}


func copy_SharedNodeInformation(nodeInfo	SharedNodeInformation) SharedNodeInformation {
	var nodeInfo_copy SharedNodeInformation

	nodeInfo_copy.MasterID = nodeInfo.MasterID

	nodeInfo_copy.NodeConnectionStatus = make(map[string]bool)
	for key, element := range nodeInfo.NodeConnectionStatus {
		nodeInfo_copy.NodeConnectionStatus[key] = element
	}

	nodeInfo_copy.NodeOperationalStatus = make(map[string]bool)
	for key, element := range nodeInfo.NodeOperationalStatus {
		nodeInfo_copy.NodeOperationalStatus[key] = element
	}

	nodeInfo_copy.States = make(map[string]ElevState)
	for key, element := range nodeInfo.States {
		nodeInfo_copy.States[key] = element
	}
	for floor := 0; floor < elevio.N_FLOORS; floor++ {
		for button := 0; button < 2; button++ {
			nodeInfo_copy.AllHallRequests[floor][button] =
				nodeInfo.AllHallRequests[floor][button]
		}
	}

	nodeInfo_copy.HRAOutput = make(map[string][elevio.N_FLOORS][2]bool)
	for key, element := range nodeInfo.HRAOutput {
		nodeInfo_copy.HRAOutput[key] = element
	}
	return nodeInfo_copy
}