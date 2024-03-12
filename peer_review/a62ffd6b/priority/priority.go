package priority

import (
	"Project/Driver-go/elevio"
	"Project/events"
	"Project/order"
	"fmt"
	"time"
)

const (
	TRAVEL_TIME    = 2500 // 2.5s
	DOOR_OPEN_TIME = 3000
)

const (
	IDLE      = 0
	MOVING    = 1
	DOOR_OPEN = 2
)

type Elevators struct {
	Id                int
	ElevatorBehaviour int
	Floor             int
	ElevatorDirection elevio.MotorDirection
	OrderRequest      [order.NumFloors][order.NumButtonTypes]int
	Orders            [order.NumFloors][order.NumButtonTypes]int
	CompletedOrders   [order.NumFloors][order.NumButtonTypes]int
}

var Elevator Elevators
var ElevatorList []Elevators

var lastOrder [order.NumFloors][order.NumButtonTypes]int

func InitLastOrder() {
	for i := 0; i < order.NumFloors; i++ {
		for j := 0; j < order.NumButtonTypes; j++ {
			lastOrder[i][j] = 0
		}
	}
}

func UpdateElevator(id int, floor *int) Elevators {
	Elevator.Id = id
	Elevator.ElevatorBehaviour = events.ElevatorState
	Elevator.Floor = *floor
	Elevator.ElevatorDirection = events.ElevatorDir
	Elevator.OrderRequest = order.ElevatorRequest
	Elevator.Orders = order.ElevatorOrders
	Elevator.CompletedOrders = order.ElevatorCompleted
	return Elevator
}

func TimeToIdle(elevator Elevators) int {
	duration := 0
	duration = SumOrders(elevator, 0, order.NumFloors) * (DOOR_OPEN_TIME + TRAVEL_TIME)
	return duration
}

func SumOrders(elevator Elevators, min, max int) int {
	sum := 0
	for i := min; i < max; i++ {
		for j := 0; j < order.NumButtonTypes; j++ {
			if elevator.Orders[i][j] == 1 {
				sum += 1
			}
		}
	}
	return sum
}

func FastestElevatorToIdle(elevatorList []Elevators, connectedPeers []int) int {
	var t []int
	var index = 0
	for _, v := range elevatorList {
		for _, u := range connectedPeers {
			if u == v.Id {
				t = append(t, TimeToIdle(v))
				fmt.Printf("TIMETOIDLE: %v\n", t)
			}
		}

	}
	a := 99999999999999
	for i, e := range t {
		if e < a {
			{
				a = e
				index = i
			}
		}
	}
	return connectedPeers[index]
}

func CompareOrderRequests(elevatorList []Elevators) ([]Elevators, [order.NumFloors][order.NumButtonTypes]int) {
	var masterOrderRequest [order.NumFloors][order.NumButtonTypes]int
	for k := range elevatorList {
		for i := 0; i < order.NumFloors; i++ {
			for j := 0; j < order.NumButtonTypes-1; j++ {
				if elevatorList[k].OrderRequest[i][j] > 0 {
					masterOrderRequest[i][j] = 1
					elevatorList[k].OrderRequest[i][j] = 0
				}
			}
		}
	}
	return elevatorList, masterOrderRequest
}

func AssignOrders(id int, elevatorList []Elevators, masterOrderRequest [order.NumFloors][order.NumButtonTypes]int) ([]Elevators, [4][3]int) {
	var index int
	for i, v := range elevatorList {
		if v.Id == id {
			index = i
		}
	}
	for i := 0; i < order.NumFloors; i++ {
		for j := 0; j < order.NumButtonTypes-1; j++ {
			if masterOrderRequest[i][j] > 0 {
				if !OrderAlreadyTaken(elevatorList, i, j) {
					elevatorList[index].Orders[i][j] = 1
					masterOrderRequest[i][j] = 0
					return elevatorList, masterOrderRequest
				}
				masterOrderRequest[i][j] = 0
			}
		}
	}
	return elevatorList, masterOrderRequest
}

func PendingMasterRequest(masterOrderRequest [order.NumFloors][order.NumButtonTypes]int) bool {
	sum := 0
	for i := 0; i < order.NumFloors; i++ {
		for j := 0; j < order.NumButtonTypes; j++ {
			if masterOrderRequest[i][j] > 0 {
				sum += 1
			}
		}
	}
	if sum > 0 {
		return true
	} else {
		return false
	}
}

func UpdateOrders(elevatorList []Elevators, id int) {
	ElevatorList = elevatorList
	for _, v := range elevatorList {
		for i := 0; i < order.NumFloors; i++ {
			for j := 0; j < order.NumButtonTypes-1; j++ {
				if v.Id == id {
					order.ElevatorOrders[i][j] = v.Orders[i][j]
				}
				if v.Orders[i][j] > 0 {
					order.ElevatorRequest[i][j] = 0
				}
			}
		}
	}
}

func ClearSlaveCompleted(elevatorList []Elevators, id int) {

	for _, v := range elevatorList {
		if v.Id == id {

			for i := 0; i < order.NumFloors; i++ {
				for j := 0; j < order.NumButtonTypes; j++ {
					if v.Orders[i][j] == 0 && lastOrder[i][j] == 1 {
						order.ElevatorCompleted[i][j] = 0
					}
				}
			}
			lastOrder = v.Orders
		}
	}
}

func OrderAlreadyTaken(elevatorList []Elevators, floor, buttontype int) bool {
	taken := false
	for _, v := range elevatorList {
		if v.Orders[floor][buttontype] > 0 {
			taken = true
		}
	}
	return taken
}

func UpdateOrderLights() {
	var j elevio.ButtonType
	var sum = 0
	for {
		time.Sleep(100 * time.Millisecond)
		order.UpdateCabOrderLights()
		for i := 0; i < order.NumFloors; i++ {
			for j = 0; j < order.NumButtonTypes-1; j++ {
				for k := range ElevatorList {
					if ElevatorList[k].Orders[i][j] > 0 {
						sum++
					}
				}
				if sum > 0 {
					elevio.SetButtonLamp(j, i, true)
				} else {
					elevio.SetButtonLamp(j, i, false)
				}
				sum = 0
			}
		}
	}
}

func ClearCompletedOrders(elevatorList []Elevators) []Elevators {
	var j elevio.ButtonType
	for i := 0; i < order.NumFloors; i++ {
		for j = 0; j < order.NumButtonTypes-1; j++ {
			for k := range ElevatorList {
				if elevatorList[k].CompletedOrders[i][j] > 0 {
					elevatorList[k].Orders[i][j] = 0
					elevatorList[k].CompletedOrders[i][j] = 0
				}
			}
		}
	}
	return elevatorList
}

func DistributeOrders(elevatorList []Elevators, connectedPeers []int) []Elevators {
	time.Sleep(100 * time.Millisecond)
	elevatorList = ClearCompletedOrders(elevatorList)
	elevatorList, masterOrderRequest := CompareOrderRequests(elevatorList)
	for PendingMasterRequest(masterOrderRequest) {
		fastestId := FastestElevatorToIdle(elevatorList, connectedPeers)
		time.Sleep(50 * time.Millisecond)
		fmt.Printf("Fastest id: %v\n", fastestId)
		elevatorList, masterOrderRequest = AssignOrders(fastestId, elevatorList, masterOrderRequest)
	}
	return elevatorList
}

func RedistributeDisconnectedOrders(lostPeers []int, elevatorList []Elevators) []Elevators {
	zero := [4][3]int{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}, {0, 0, 0}}
	var cabOrders [4][1]int
	for _, v := range lostPeers {
		for i, u := range elevatorList {
			if v == u.Id {
				for j := range elevatorList[i].Orders {
					cabOrders[j][0] = elevatorList[i].Orders[j][2]
				}
				for k := 0; k < order.NumFloors; k++ {
					for j := 0; j < order.NumButtonTypes-1; j++ {
						elevatorList[i].OrderRequest[k][j] = elevatorList[i].Orders[k][j]
					}
				}
				elevatorList[i].Orders = zero
				for j := range elevatorList[i].Orders {
					elevatorList[i].Orders[j][2] = cabOrders[j][0]

				}
			}
		}
	}
	return elevatorList
}

func UpdateCabOrders(elevatorList []Elevators, id int) {
	for _, v := range elevatorList {
		if v.Id == id {
			for i := 0; i < order.NumFloors; i++ {
				for j := order.NumButtonTypes - 1; j < order.NumButtonTypes; j++ {
					order.ElevatorOrders[i][j] = v.Orders[i][j]
				}
			}
		}
	}

}
