package main

import (
	"time"

	elevio "./elev_driver"
	network "./network_module"
	orderhandler "./order_handler"
	slog "./sessionlog"
	statemachine "./stateMachine"
	turnofflight "./turn_off_lights"
)

func main() {
	//InitElevator
	numFloors := 4
	elevio.Init("localhost:15657", numFloors)

	go statemachine.CheckFloor()

	//Get to a floor if booted between floors.
	for elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Up)
	}
	if statemachine.IsIdle() {
		elevio.SetMotorDirection(elevio.MD_Stop)
	}
	time.Sleep(100 * time.Millisecond)

	//Init complete, start Go routines to enter normal operation
	go orderhandler.CheckButtons()
	go slog.CheckDisconnect()
	go slog.LogExecuter()
	network.InitNetwork()
	turnofflight.InitTurnOffLights()
	go network.NetworkReceive()
	go turnofflight.TurnOffLightReceive()

	//Run forever
	for {
		select {}
	}
}
