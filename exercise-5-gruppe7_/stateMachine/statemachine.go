package statemachine

import (
	elevio "../elev_driver"
)

var toFloor = -1
var floor int
var idle bool
var doorOpen bool
var newFloor bool

func SetDoorOpen(x bool) {
	doorOpen = x
}

func GetDoorOpen() bool {
	return doorOpen
}

func GetFloor() int {
	return floor
}

func SettoFloor(x int) {
	toFloor = x
}

func GetDirection() int {
	if !IsIdle() {
		if floor < toFloor {
			return 1
		} else {
			return -1
		}
	} else {
		return 0
	}
}

func SetIdle() {
	idle = true
}
func IsIdle() bool {
	if (toFloor == elevio.GetFloor()) || (toFloor == -1) {
		idle = true
	} else {
		idle = false
	}
	return idle
}

func GettoFloor() int {
	return toFloor
}

func GetNewFloor() bool {
	return newFloor
}

func SetNewFloor(x bool) {
	newFloor = x
}

func CheckFloor() {
	drv_floors := make(chan int)
	go elevio.PollFloorSensor(drv_floors)
	for {
		select {
		case floorSensor := <-drv_floors:
			SetNewFloor(true)
			floor = floorSensor
			elevio.SetFloorIndicator(floor)
		}

	}
}
