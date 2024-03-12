package arbitration

import (
	"Elevdriver/config"
	"Elevdriver/data_structure"
	"Elevdriver/network/peers"
	"fmt"
	"strconv"
)

// ______UTILITY FUNCTION___________

func findElementInArray(i int, list []string) (bool, int, int) {

	/* Iterates through array
	and returns whether the element is found,
	the integer value of the element,
	and the index of the element */

	index := 0
	str_i := strconv.Itoa(i)

	for _, b := range list {
		if b == str_i {
			element, _ := strconv.Atoi("b")
			return true, element, index
		}
		index++
	}

	return false, -1, -1
}

//_______MAIN ARBITRATION FUNCTION_______________

func Arbitration(systemInfo data_structure.System_info_t,
	arbitration_status_to_network_chan chan data_structure.Arbitration_t,
	arbitration_status_to_distrbutor_chan chan data_structure.Arbitration_t,
	stuck_send_chan chan bool) {

	/* Decides whether the elevator is the master,
	sends information about the master status to distributor and elevator_network,
	and updates the elevator's list of which elevators are still connected. */

	// Invalid startup values.
	validMaster := false
	currentMaster := 1000

	var arb data_structure.Arbitration_t

	// String value of own ID to be compared to Peers list of IDs.
	idString := strconv.Itoa(systemInfo.Id)

	networkUpdateCh := make(chan peers.PeerUpdate)

	// Send own ID to network, and disable elevator if it is stuck.
	go peers.Transmitter(systemInfo.PeerPort, idString, stuck_send_chan)

	// Receive network updates.
	go peers.Receiver(systemInfo.PeerPort, networkUpdateCh)

	for {
		select {
		case p := <-networkUpdateCh:

			fmt.Printf("Network:\n")
			fmt.Printf("  Elevators: %q\n", p.Peers)
			fmt.Printf("  New:       %q\n", p.New)
			fmt.Printf("  Lost:      %q\n", p.Lost)

			// Determine whether the elevator is connected to a network.
			if len(p.Peers) > 1 {
				arb.Connected = true
			} else {
				arb.Connected = false
				for i := 0; i < config.NUM_ELEVATORS; i++ {
					arb.Alive_list[i] = i == systemInfo.Id
				}

			}

			// Checks whether the p.Peers list is empty to avoid indexing errors,
			// no arbitration should happen if an empty network update is
			// received
			if len(p.Peers) > 0 {

				/* Peers is sorted in ascending order,
				so the lowest ID on the network
				will always be the first element.*/

				minId, _ := strconv.Atoi(p.Peers[0])

				// If master's ID is not in the p.Peers list, it is lost.
				masterLost, _, _ := findElementInArray(currentMaster, p.Lost)

				if !validMaster {

					// The startup case.

					currentMaster = minId
					validMaster = true

					if systemInfo.Id == minId {
						arb.Is_master = true
					} else {
						arb.Is_master = false
					}

				} else if masterLost {

					// In case a master is removed from the network.

					currentMaster = minId

					if systemInfo.Id == minId {
						arb.Is_master = true
					} else {
						arb.Is_master = false
					}

				}

				// Update the list of connected elevators to set dead connections to false.
				for i, _ := range arb.Alive_list {
					peerAlive, _, _ := findElementInArray(i, p.Peers)
					if !peerAlive {
						arb.Alive_list[i] = false
					} else {
						arb.Alive_list[i] = true
					}
				}
			}

			// Send the result of the arbitration to elevator_network
			// and distributor.
			arbitration_status_to_network_chan <- arb
			arbitration_status_to_distrbutor_chan <- arb

		}

	}
}
