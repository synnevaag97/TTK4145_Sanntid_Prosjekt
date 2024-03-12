package messages

import (
	"Driver-go/elevio"
	"Types-go/Type-Node/node"
	"time"
)

const NumButtons int = 3

// Light message data structure
type LightMsg struct {
	ButtonEvent elevio.ButtonEvent
	Value       bool
}

//Creating a light message function
func Create_LightMsg(Button_Event elevio.ButtonEvent, Value bool) LightMsg {
	light_msg := LightMsg{}
	light_msg.ButtonEvent = Button_Event
	light_msg.Value = Value
	return light_msg
}

// Initialisation message data structure
type InitiationMsg struct {
	Id       string
	Database map[string]node.NetworkNode
}

func Create_InitialisationMsg(Id string, database map[string]node.NetworkNode) InitiationMsg {
	Init_msg := InitiationMsg{}
	Init_msg.Id = Id
	Init_msg.Database = database
	return Init_msg
}

// Uninintiated message data structure to inform another node that it appears at uninitiated, confirm initiated
type UnInitiatedMsg struct {
	Sending_Id   string
	Unitiated_Id string
	Initiated    bool
}

//Creating a uninitialization message function
func Create_UnInitialisationMsg(Unitiated_Id string, sending_id string, initiated bool) UnInitiatedMsg {
	Init_msg := UnInitiatedMsg{}
	Init_msg.Unitiated_Id = Unitiated_Id
	Init_msg.Sending_Id = sending_id
	Init_msg.Initiated = initiated
	return Init_msg
}

//Request complete message data structure
type ReqCompleteMsg struct {
	Id        string
	Request   elevio.ButtonEvent
	Timestamp int64
}

//Creating a request complete message function
func Create_ReqCompleteMsg(Id string, Request elevio.ButtonEvent) ReqCompleteMsg {
	ReqComplete_Msg := ReqCompleteMsg{}
	ReqComplete_Msg.Id = Id
	ReqComplete_Msg.Request = Request
	ReqComplete_Msg.Timestamp = time.Now().Unix()
	return ReqComplete_Msg
}

//Update elevator message data structure
type UpdateElevMsg struct {
	Id       string
	Elevator node.NetworkElevState
}

func Create_UpdateElevMsg(Id string, elev node.NetworkElevState) UpdateElevMsg {
	UpdateElev_Msg := UpdateElevMsg{}
	UpdateElev_Msg.Elevator = elev
	UpdateElev_Msg.Id = Id
	return UpdateElev_Msg
}

//Update hall request message datastruture
type UpdateHallRequestMsg struct {
	Id           string
	Button_Event elevio.ButtonEvent
}

//Creating an update hall request message function
func Create_UpdateHallRequestMsg(Id string, Hall_Requests elevio.ButtonEvent) UpdateHallRequestMsg {
	UpdateHallRequest_Msg := UpdateHallRequestMsg{}
	UpdateHallRequest_Msg.Id = Id
	UpdateHallRequest_Msg.Button_Event = Hall_Requests
	return UpdateHallRequest_Msg
}
