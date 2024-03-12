# Project in TTK4145: Controlling multiple elevators over multiple floors.

## To start up a node:
Open two terminals(1 & 2) and navigate to folder ./project-group-[REDACTED]-1
Terminal 1 will run the elevatorserver, to communicate with the physical elevator.
Terminal 2 will run the project executable to controll the elevator.

**In Terminal 1:**
From ./project-group-[REDACTED]-1$ 
Type the line:
./elevatorserver

**In Terminal 2:**
From ./project-group-[REDACTED]-1$ 
Type the line:
./Elevator 'port' 'id'
To be compatible with elevatorserver, choose port: 15657
Options for id(*): -10, -9, -8

(*)In principle any id, but the system has a filter configured to only allow these id's due to the intense circumstances at the real-time-lab.

<br/>

## To control multiple elevators set up a node on each elevator.

Alternative setup - One physical elevator and two simulators:
Open one terminal and navigate to folder ./project-group-[REDACTED]-1/utilities
Type the line:
./elev_start

This runs a bash script utilizing tmux and the SimElevatorServer, which simulates an elevatorserver and provides a graphical interface in the terminal.

<br/>

## Project structure

Some of the content in this project was handed out as supporting utilities. 
A brief overview of this content follows:
        Communication with hardware in ./Driver-go 
        Network framework for UDP broadcast in ./Network-go
        Cost-algorithm to calculte order distribution 'hall_request_assigner' in ./utilities
        Simulator for testing purposes 'Simulator-v2-1.5' in ./utilities
   

<br/>

## Master-slave configuration
The project is built around always having one master, while the rest of the nodes are slaves. If the master dies, a new master  will be chosen from the slaves based on lowest ID. If a node reconnects, a new selection of master is also done. 

The master is responsible for distributing hall calls as well as information about the other nodes. This results in all nodes having the necessary information for becoming master. 

<br/>

## Process Pairs
-"Design for crashability"

To combat go-routines potentially going into a lock, the program contains process pairs as a counter-measure.
Each node starts it's own backup.
Then the program sends a "baton" from one go-routine to the next one like a "daisy chain" (until it has visited all go routines). If the baton "stops" in a go-routine (e.g. fsm has a lock), the program will terminate itself and the already started backup routine will become the main routine and spawn a new backup.