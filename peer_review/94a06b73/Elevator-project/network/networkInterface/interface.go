package intf

import (
	cf "Elevator-go/Elevator/type_"
	"Elevator-go/network/bcast"
	"Elevator-go/network/peers"
	"fmt"
)

var MasterElevatorId string
var OnlineElevsId []string

var flag bool = true

func Network_interface(
	ch_orderToExternalElevator chan cf.OrderToExternalElev,
	ch_orderFromExternalElevator chan cf.OrderToExternalElev,
	ch_localElevatorStateToNtk chan cf.LocalElevatorState,
	ch_ackToMaster chan string, ch_ackFromElevs chan string) {

	ch_orderFromexternal := make(chan cf.OrderToExternalElev)
	ch_elevStatesFromExternal := make(chan cf.LocalElevatorState)
	ch_OnlineElevsId := make(chan peers.PeerUpdate)
	ch_elevStatesToExternal := make(chan cf.LocalElevatorState)
	ch_peerTxEnable := make(chan bool)

	ch_ackFromController := make(chan string)
	ch_ackTocontroller := make(chan string)

	go peers.Transmitter(15647, cf.LocalElevId, ch_peerTxEnable)
	go peers.Receiver(15647, ch_OnlineElevsId)

	go bcast.Transmitter(16569, ch_orderToExternalElevator, ch_elevStatesToExternal, ch_ackFromController)
	go bcast.Receiver(16569, ch_elevStatesFromExternal, ch_orderFromexternal, ch_ackTocontroller)

	for {
		select {
		case state := <-ch_localElevatorStateToNtk: /* broadcast local elevator state */

			ch_elevStatesToExternal <- state

		case p := <-ch_OnlineElevsId: /* if new elevator joined or lost(peer change) */
			fmt.Printf("Online Elevators:    %q\n", p.Peers)
			fmt.Printf("Lost Elevator:    %q\n", p.Lost)

			/* update master elevator */
			for i, elev := range p.Peers {
				if i == 0 || elev > MasterElevatorId {
					MasterElevatorId = elev
				}
			}
			/* update online elevators id */
			OnlineElevsId = p.Peers
			fmt.Printf("Master elevator:    %v\n", MasterElevatorId)

			/* if an elevator is offline remove it from online elevators state list(OnlineElevatorsState) */
			if len(p.Lost) != 0 {
			Outer:
				for i := 0; i < len(p.Lost); i++ {
					for j := 0; j < len(cf.OnlineElevatorsState); j++ {
						if p.Lost[i] == cf.OnlineElevatorsState[j].ElevatorId {
							cf.OnlineElevatorsState = append(cf.OnlineElevatorsState[:j], cf.OnlineElevatorsState[j+1:]...)
							break Outer
						}
					}
				}
			}

		case order := <-ch_orderFromexternal: /* receive order from network and send it to local elevator */
			ch_orderFromExternalElevator <- order

		case orderToExternal := <-ch_orderToExternalElevator: /* broadcast local order through network */
			ch_orderToExternalElevator <- orderToExternal

		case state := <-ch_elevStatesFromExternal: /* receive other elevators state from network */

			/* update online elevators state (OnlineElevatorsState) */
			for i := 0; i < len(cf.OnlineElevatorsState); i++ {
				if state.ElevatorId == cf.OnlineElevatorsState[i].ElevatorId {
					cf.OnlineElevatorsState[i] = state
					flag = false
					break
				}
			}
			if flag {
				cf.OnlineElevatorsState = append(cf.OnlineElevatorsState, state)
			} else {
				flag = true
			}
		case ack := <-ch_ackTocontroller:
			ch_ackToMaster <- ack
		case ack := <-ch_ackFromElevs:
			ch_ackFromController <- ack
		}
	}
}
