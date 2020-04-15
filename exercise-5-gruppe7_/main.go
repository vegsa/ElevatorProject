package main

import (
	//"fmt"
	"time"
	elevio "./elev_driver"
	slog "./sessionlog"
	statemachine "./stateMachine"
	orderhandler "./order_handler"
	network "./network_module" 
	turnofflight "./turn_off_lights"
)

func main() {
	//InitElevator
	numFloors := 4
	elevio.Init("localhost:15657", numFloors)

	// Have to start this goroutine here, because we need to 
	//know what floor it is when we start the other goroutines. Will not update
	// if we put it after the boot of the elevator.
	go statemachine.CheckFloor()

	//Get to a floor if booted between floors.
	//Moved this up here because of when we start the program with orders in the log.
	//Will get difficulties with that issue if we do this after the goroutines, I think.
	for elevio.GetFloor() == -1{
		elevio.SetMotorDirection(elevio.MD_Up)
	}
	if statemachine.IsIdle(){
		elevio.SetMotorDirection(elevio.MD_Stop)
	}
	time.Sleep(100*time.Millisecond)


	//Start GO routines
	go orderhandler.CheckButtons()
	//go statemachine.CheckFloor()
	go slog.CheckDisconnect()
	go slog.QueueWatchdog()
	network.InitNetwork()
	turnofflight.InitTurnOffLights()
	go network.NetworkReceive()
	go turnofflight.TurnOffLightReceive()

	
	//Run forever
	for {
		select{
		}

	}
}
