package ctrl

import (
	"Elevator-go/Elevator/elevio"
	cf "Elevator-go/Elevator/type_"
	"Elevator-go/network/bcast"
	intf "Elevator-go/network/networkInterface"
	"Elevator-go/network/peers"
	"strings"
	"time"
)

var No_Id string = "no-id"
var All_requests []elevio.ButtonEvent
var Order elevio.ButtonEvent
var Order_toMaster cf.OrderToExternalElev
var elev_floor, tr_num int

type elev_motor int

const (
	initial_value     elev_motor = 0
	motor_functioning elev_motor = 1
	motor_fail        elev_motor = 2
)

var elev_motor_status elev_motor = initial_value

var ackn_fromMaster bool

type ackn_fromElevs int

const (
	Start            ackn_fromElevs = 1
	not_acknowledged ackn_fromElevs = 2
	acknowledged     ackn_fromElevs = 3
)

var cknowledgment ackn_fromElevs

func Elevator_controller(
	ch_localOrderFromButton chan elevio.ButtonEvent,
	ch_orderFromExternalElevator chan cf.OrderToExternalElev,
	ch_orderToLocalElevator chan elevio.ButtonEvent,
	ch_orderToExternalElevator chan cf.OrderToExternalElev,
	ch_ackToMaster chan string, ch_ackFromElevs chan string) {

	ch_rx_allRequests := make(chan cf.Elevator)
	ch_peerTxEnable := make(chan bool)
	ch_peerDiscovery := make(chan peers.PeerUpdate)

	go peers.Transmitter(15648, cf.LocalElevId, ch_peerTxEnable)
	go peers.Receiver(15648, ch_peerDiscovery)

	go bcast.Receiver(16570, ch_rx_allRequests)

	sendNewOreder := time.NewTicker(100 * time.Millisecond)
	prevOreder := time.NewTicker(10 * time.Millisecond)

	prevOreder.Stop()
	cknowledgment = Start
	ch_peerTxEnable <- false
	tr_num = 1

	for {
		select {
		case localOrder := <-ch_localOrderFromButton: /* Order is from local elevator */

			switch {
			case localOrder.Button == elevio.BT_Cab: /* if the local order is cab call, send it to local elevator */

				ch_orderToLocalElevator <- localOrder /* send order to local elevator */

			case localOrder.Button == elevio.BT_HallUp || localOrder.Button == elevio.BT_HallDown:

				if intf.MasterElevatorId == cf.LocalElevId { /* if the elavetor is a master, ... */

					All_requests = append(All_requests, localOrder)

				} else { /* if the elevator is not a master, it is either ... */

					if len(intf.OnlineElevsId) == 0 { /* offline or ... */

						ch_orderToLocalElevator <- localOrder /* send order to local elevator */

					} else { /* a slave. */

						/* Forward order to the master with specific Id */
						ch_orderToExternalElevator <- cf.OrderToExternalElev{Order: localOrder, Elev_Id: strings.Join([]string{No_Id, cf.LocalElevId}, "")}

						Order_toMaster = cf.OrderToExternalElev{Order: localOrder, Elev_Id: strings.Join([]string{No_Id, cf.LocalElevId}, "")}
						ackn_fromMaster = false
					}
				}
			}

		case externalOrder := <-ch_orderFromExternalElevator: /* order from network */

			if intf.MasterElevatorId == cf.LocalElevId && externalOrder.Elev_Id[0:5] == No_Id { /* if the local elevator is master and the order is forwarded from a slave */
				ch_ackFromElevs <- externalOrder.Elev_Id

				All_requests = append(All_requests, externalOrder.Order)

			} else if externalOrder.Elev_Id == cf.LocalElevId { /* if the local elevator is a slave and the order belongs to it */

				ch_orderToLocalElevator <- externalOrder.Order /* send order to local elevator */
				ch_ackFromElevs <- cf.LocalElevId              /* send acknowledgment to master elevator */

			}

		case ack := <-ch_ackToMaster:
			if intf.MasterElevatorId == cf.LocalElevId && ack[0:5] != No_Id { /* from other elevators */
				/* fmt.Printf("Ack from %v (not master)\n", ack) */
				cknowledgment = acknowledged
			} else if intf.MasterElevatorId != cf.LocalElevId && ack[5:] == cf.LocalElevId { /* from master elevator */
				/* fmt.Printf("Ack from %v (master) to %v  \n", intf.MasterElevatorId, ack[5:]) */
				Order_toMaster = cf.OrderToExternalElev{}
				ackn_fromMaster = true
			}
		case <-sendNewOreder.C:
			if intf.MasterElevatorId == cf.LocalElevId {
				if len(All_requests) > 0 {
					/* if previous order is not acknowledged or if there is motor failer store is back to All_Requests */
					if cknowledgment == not_acknowledged || elev_motor_status == motor_fail {
						All_requests = append(All_requests, Order)
					}
					/* select elevator */
					SelectElevator()
					/* take the next order from All_Requests */
					Order = All_requests[0]
					All_requests = All_requests[1:]

					cknowledgment = not_acknowledged
					/* start order timer */
					prevOreder = time.NewTicker(10 * time.Millisecond)
				} else {
					/* if there is no request stop order timer */
					prevOreder.Stop()
				}
			} else if !ackn_fromMaster && Order_toMaster.Elev_Id == strings.Join([]string{No_Id, cf.LocalElevId}, "") {
				/* Forward order to the master with specific Id */
				ch_orderToExternalElevator <- Order_toMaster
			}

		case <-prevOreder.C:
			if intf.MasterElevatorId == cf.LocalElevId && cknowledgment == not_acknowledged {
				if intf.OnlineElevsId[0] == cf.LocalElevId { /* it either excute the order locally if it is selected by 'SelectElevator()' or ... */

					ch_orderToLocalElevator <- Order /* send order to local elevator */
					cknowledgment = acknowledged

				} else { /* it will send it to selected elevator if the order is not for the master. */

					/* send the local hall call order to the elevator selected by 'SelectElevator()' */
					ch_orderToExternalElevator <- cf.OrderToExternalElev{Order: Order, Elev_Id: intf.OnlineElevsId[0]}

				}
			}

		case requests := <-ch_rx_allRequests:
			/* Hall light handler */
			for floor := 0; floor < cf.NumFloors; floor++ {
				for btn := 0; btn < cf.NumButtons-1; btn++ {
					elevio.SetButtonLamp(elevio.ButtonType(btn), floor, requests.Requests[floor][btn])
				}
			}
			if intf.MasterElevatorId == cf.LocalElevId {
				/* fmt.Println("Elevator status: ", requests.Behave) */
			}
			/* if intf.MasterElevatorId == cf.LocalElevId {
				if tr_num == 1 {
					elev_floor = requests.Floor
					tr_num = 2
					fmt.Println("Initial floor: ", elev_floor)
				} else if tr_num == 2 {
					tr_num = 1
					if elev_floor != requests.Floor {
						elev_motor_status = motor_functioning
						fmt.Println("Final floor(working): ", requests.Floor)
					} else {
						elev_motor_status = motor_fail
						fmt.Println("Final floor(failed): ", requests.Floor)
					}
				}
			} */
		}
	}
}

func SelectElevator() {
	a := intf.OnlineElevsId[0]
	intf.OnlineElevsId = intf.OnlineElevsId[1:]
	intf.OnlineElevsId = append(intf.OnlineElevsId, a)
}
