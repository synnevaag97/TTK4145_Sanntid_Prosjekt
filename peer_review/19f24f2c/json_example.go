package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

type HRAElevState struct {
	Behavior    string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

func main1() {

	input := HRAInput{
		HallRequests: [][2]bool{{false, false}, {true, false}, {false, false}, {false, true}},
		States: map[string]HRAElevState{
			"one": HRAElevState{
				Behavior:    "moving",
				Floor:       2,
				Direction:   "up",
				CabRequests: []bool{false, false, false, true},
			},
			"two": HRAElevState{
				Behavior:    "idle",
				Floor:       0,
				Direction:   "stop",
				CabRequests: []bool{false, false, false, false},
			},
		},
	}

	jsonBytes, err := json.Marshal(input)
	fmt.Println("json.Marshal error: ", err)

	ret, err := exec.Command("../hall_request_assigner/hall_request_assigner.exe", "-i", string(jsonBytes)).Output()
	fmt.Println("exec.Command error: ", err)

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	fmt.Println("json.Unmarshal error: ", err)

	fmt.Printf("%+v\n", output)

	fmt.Printf("output: \n")
	for k, v := range *output {
		fmt.Printf("%6v :  %+v\n", k, v)
	}

}
