package cost_fns

import (
	"Driver-go/elevio"
	"Types-go/Type-HRA/hra"
	"Types-go/Type-Node/node"
	"encoding/json"
	"os/exec"
	"runtime"
)

func Request_assigner(numFloors int, database map[string]node.NetworkNode, newOrders []elevio.ButtonEvent) map[string][][2]bool {

	file := ""
	switch runtime.GOOS {
	case "linux":
		file = "hall_request_assigner"
	case "windows":
		file = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	database_json := hra.DatabasetoHRA(database, numFloors)
	hallrequest := node.GetAllHallRequests(numFloors, database)
	for k := range newOrders {
		hallrequest[newOrders[k].Floor][int(newOrders[k].Button)] = true
	}

	input := hra.HRAInput{
		HallRequests: hallrequest[:],
		States:       database_json,
	}
	jsonBytes, _ := json.Marshal(input)
	ret, _ := exec.Command("./Cost-go/cost_fns/hall_request_assigner/"+file, "-i", string(jsonBytes)).Output()
	output := new(map[string][][2]bool)
	_ = json.Unmarshal(ret, &output)

	return (*output)
}
