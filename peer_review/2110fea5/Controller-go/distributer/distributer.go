package distributer

import (
	"Cost-go/cost_fns"
	"Driver-go/elevio"
	"Elevator-go/singleElevator"
	"Network-go/Bcast/bcast"
	"Network-go/Peers/peers"
	"Types-go/Type-Elevator/elevator"
	"Types-go/Type-Msg/messages"
	"Types-go/Type-Node/node"
	"Watchdog-go/watchdog"
	"encoding/json"
	"time"
)

func RunDistributer(Id string, port string, numFloors int, reboot_chan chan<- bool) int {
	time.Sleep(1 * time.Second)

	// Define network channels
	LightMsgTx := make(chan messages.LightMsg, 30)
	LightMsgRx := make(chan messages.LightMsg, 30)

	ReqCompleteTx := make(chan messages.ReqCompleteMsg, 30)
	ReqCompleteRx := make(chan messages.ReqCompleteMsg, 30)

	InitiationMsgTx := make(chan messages.InitiationMsg, 30)
	InitiationMsgRx := make(chan messages.InitiationMsg, 30)

	UnInitiationTx := make(chan messages.UnInitiatedMsg, 30)
	UnInitiationRx := make(chan messages.UnInitiatedMsg, 30)

	DatabaseTx := make(chan []byte, 10)
	DatabaseRx := make(chan []byte, 10)

	ElevatorChangesTx := make(chan messages.UpdateElevMsg, 30)
	ElevatorChangesRx := make(chan messages.UpdateElevMsg, 30)

	NewHallRequestTx := make(chan messages.UpdateHallRequestMsg, 30)
	NewHallRequestRx := make(chan messages.UpdateHallRequestMsg, 30)

	NewPeerOnNetwork := make(chan peers.PeerUpdate, 30)
	LostPeersOnNetwork := make(chan []string, 30)

	port_bcast := 16555
	go bcast.Transmitter(
		port_bcast,
		InitiationMsgTx,
		DatabaseTx,
		ElevatorChangesTx,
		NewHallRequestTx,
		LightMsgTx,
		ReqCompleteTx,
		UnInitiationTx)

	go bcast.Receiver(
		port_bcast,
		InitiationMsgRx,
		DatabaseRx,
		ElevatorChangesRx,
		NewHallRequestRx,
		LightMsgRx,
		ReqCompleteRx,
		UnInitiationRx)

	go peers.PeerUpdates(
		Id,
		NewPeerOnNetwork,
		LostPeersOnNetwork)

	// Define elevator and watchdog channels
	new_hall_request_from_elevator := make(chan elevio.ButtonEvent, 30)
	completed_hall_request_from_elevator := make(chan elevio.ButtonEvent, 30)
	elev_changes_from_polling := make(chan messages.UpdateElevMsg, 100)

	order_expired_from_watchdog := make(chan messages.ReqCompleteMsg, 30)
	order_completed_to_watchdog := make(chan messages.ReqCompleteMsg, 30)
	assigned_hall_requests_to_watchdog := make(chan map[string][][2]bool, 30)

	cab_completed_to_watchdog := make(chan messages.ReqCompleteMsg, 30)
	cab_request_to_watchdog := make(chan elevio.ButtonEvent, 30)

	initiation_expired := make(chan bool, 30)
	initiation_database_arrived := make(chan bool, 30)
	start_initiation_timer := make(chan bool, 30)

	elev := elevator.ElevatorState{}
	elev.State = node.UNDEFINED

	// Run elevator and watchdog modules.
	go singleElevator.RunElevator(
		numFloors,
		&elev,
		port,
		new_hall_request_from_elevator,
		completed_hall_request_from_elevator,
		cab_request_to_watchdog,
		cab_completed_to_watchdog)

	go watchdog.RunWatchdog(
		numFloors,
		Id,
		assigned_hall_requests_to_watchdog,
		order_completed_to_watchdog,
		order_expired_from_watchdog,
		cab_request_to_watchdog,
		cab_completed_to_watchdog,
		initiation_expired,
		initiation_database_arrived,
		start_initiation_timer,
		reboot_chan)

	// Initialization of dataif thisE, ok base
	database := node.InitiateLocalDatabase(numFloors, Id)

	// Wait till elevator is initiated.
	for {
		if elev.State == node.IDLE {
			break
		}
	}

	go elev.PollElevatorChanges(numFloors, Id, ElevatorChangesTx, elev_changes_from_polling, &database)

	// Run distributer
	for {
		select {

		case new_peer := <-NewPeerOnNetwork:

			if (len(new_peer.Peers) == 1) && (new_peer.New) == Id {
				if thisDatabase, ok := database[Id]; ok {
					thisDatabase.Initiated = true
					database[Id] = thisDatabase
				}

			} else if (len(new_peer.Peers) > 1) && (database[Id].Initiated) {
				database_string, _ := json.Marshal(database)
				DatabaseTx <- database_string

			} else {
				start_initiation_timer <- true
			}

		case <-initiation_expired:
			if thisDatabase, ok := database[Id]; ok {
				thisDatabase.Initiated = true
				database[Id] = thisDatabase
			}

		case database_initiation_string := <-DatabaseRx:
			database_from_initiated_nodes := make(map[string]node.NetworkNode)
			json.Unmarshal(database_initiation_string, &database_from_initiated_nodes)
			if database[Id].Initiated == false {

				initiation_database_arrived <- true
				node.InitiateGlobalDatabase(Id, numFloors, &database, database_from_initiated_nodes)

				InitiatedMsg := messages.Create_InitialisationMsg(Id, database)
				InitiationMsgTx <- InitiatedMsg

				elev.FetchCabChanges(numFloors, Id, &database)
				elev.FetchLights(numFloors, Id, &database)
			}

		case new_node_initiated := <-InitiationMsgRx:
			if new_node_initiated.Id != Id {
				database[new_node_initiated.Id] = new_node_initiated.Database[new_node_initiated.Id]
			}

		case lost_peers := <-LostPeersOnNetwork:
			for k := range lost_peers {
				if thisDatabase, ok := database[lost_peers[k]]; ok {
					thisDatabase.Initiated = false
					database[lost_peers[k]] = thisDatabase
				}
			}

			hall_requests := node.GetActiveHallRequestsInNodes(lost_peers, &database)

			assigned_hall_requests := cost_fns.Request_assigner(numFloors, database, hall_requests)
			assigned_hall_requests_to_watchdog <- assigned_hall_requests

			node.UpdateDatabase_AddHallRequests(numFloors, &database, assigned_hall_requests)

			elev.FetchHallChanges(numFloors, Id, &database)

		case uninitiated_node := <-UnInitiationRx:

			if (uninitiated_node.Sending_Id != Id) && (uninitiated_node.Unitiated_Id == Id) && (database[Id].Initiated == true) {
				UnInitMsg := messages.Create_UnInitialisationMsg(Id, Id, true)
				UnInitiationTx <- UnInitMsg
			} else if uninitiated_node.Sending_Id == uninitiated_node.Unitiated_Id {
				if Database, ok := database[uninitiated_node.Sending_Id]; ok {
					Database.Initiated = uninitiated_node.Initiated
					database[uninitiated_node.Sending_Id] = Database
				}
			}

		case hall_request := <-new_hall_request_from_elevator:
			HallRequestMsg := messages.Create_UpdateHallRequestMsg(Id, hall_request)
			NewHallRequestTx <- HallRequestMsg

			light_msg := messages.Create_LightMsg(hall_request, true)
			LightMsgTx <- light_msg

			hall_requests := []elevio.ButtonEvent{hall_request}
			assigned_hall_requests := cost_fns.Request_assigner(numFloors, database, hall_requests)

			assigned_hall_requests_to_watchdog <- assigned_hall_requests

			node.UpdateDatabase_AddHallRequests(numFloors, &database, assigned_hall_requests)

			elev.FetchHallChanges(numFloors, Id, &database)

		case updated_hall_req_from_network := <-NewHallRequestRx:

			hall_requests := []elevio.ButtonEvent{updated_hall_req_from_network.Button_Event}
			assigned_hall_requests := cost_fns.Request_assigner(numFloors, database, hall_requests)
			assigned_hall_requests_to_watchdog <- assigned_hall_requests

			node.UpdateDatabase_AddHallRequests(numFloors, &database, assigned_hall_requests)

			elev.FetchHallChanges(numFloors, Id, &database)

		case updated_elevator := <-ElevatorChangesRx:
			if (updated_elevator.Id != Id) && (database[updated_elevator.Id].Initiated) {
				node.UpdateDatabase_AddElevatorChange(numFloors, &database, updated_elevator.Id, updated_elevator.Elevator)
			}

		case completedHallReq := <-completed_hall_request_from_elevator:
			light_msg := messages.Create_LightMsg(completedHallReq, false)
			LightMsgTx <- light_msg

			CompletedRequest := messages.Create_ReqCompleteMsg(Id, completedHallReq)
			order_completed_to_watchdog <- CompletedRequest
			ReqCompleteTx <- CompletedRequest

			node.UpdateDatabase_RemoveRequest(&database, Id, completedHallReq)

		case reqcomplete := <-ReqCompleteRx:
			if reqcomplete.Id != Id {
				order_completed_to_watchdog <- reqcomplete
				node.UpdateDatabase_RemoveRequest(&database, reqcomplete.Id, reqcomplete.Request)
			}

		case light_msg := <-LightMsgRx:
			elevio.SetButtonLamp(light_msg.ButtonEvent.Button, light_msg.ButtonEvent.Floor, light_msg.Value)

		case order_expired_watchdog := <-order_expired_from_watchdog:

			if thisDatabase, ok := database[order_expired_watchdog.Id]; ok {
				thisDatabase.Initiated = false
				database[order_expired_watchdog.Id] = thisDatabase
			}

			uninitMsg := messages.Create_UnInitialisationMsg(order_expired_watchdog.Id, Id, false)
			UnInitiationTx <- uninitMsg

			nodes := []string{order_expired_watchdog.Id}
			hall_requests := node.GetActiveHallRequestsInNodes(nodes, &database)

			assigned_hall_requests := cost_fns.Request_assigner(numFloors, database, hall_requests)
			assigned_hall_requests_to_watchdog <- assigned_hall_requests

			node.UpdateDatabase_AddHallRequests(numFloors, &database, assigned_hall_requests)

			elev.FetchHallChanges(numFloors, Id, &database)
		}
	}
}
