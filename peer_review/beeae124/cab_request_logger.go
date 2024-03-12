package main

import (
	"Driver-go/elevio"
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
)

func writeCabRequestLog(cabRequests [elevio.N_FLOORS]bool) {
	//Write to file NB: Overwrites file with same name in folder
	fileWrite, err := os.Create("./elevator_cab_request_log.txt") 
	if err != nil {
		log.Fatalf("failed to open")

	}

	writer := bufio.NewWriter(fileWrite)

	//Convert cab requests to a slice of strings.
	var cabRequestsString [elevio.N_FLOORS]string
	for i := 0; i < elevio.N_FLOORS; i++ {
		stringFromBool := fmt.Sprint(cabRequests[i])
		cabRequestsString[i] = stringFromBool

	}
	for _, line := range cabRequestsString {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			log.Fatalf("Error writing to file. Err: %s", err.Error())
		}
	}
	writer.Flush()
}

func readCabRequestLog() [elevio.N_FLOORS]bool {
	//Read from file
	file, err := os.Open("elevator_cab_request_log.txt")

	if err != nil {
		log.Fatalf("failed to open")

	}

	//Read/Scan the file
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var cabRequestsFromLog []string

	for scanner.Scan() {
		cabRequestsFromLog = append(cabRequestsFromLog, scanner.Text())
	}

	//Close the file
	file.Close()

	//Convert the string values to a slice of bools
	var cabRequestsBool [elevio.N_FLOORS]bool
	for floor_number, value := range cabRequestsFromLog {
		fmt.Println(value)
		boolFromString, err := strconv.ParseBool(value)
		if err != nil {
			log.Fatal(err)
		}

		cabRequestsBool[floor_number] = boolFromString
	}
	return cabRequestsBool
}