package hall_call_handler

//********
// Routines for computing a suitability score for each elevator and handling of designated hall call orders.
// Moved here because of loops when it was in order_handler.
//********

import (
	"math"
	"time"

	elevio "../elev_driver"
	slog "../sessionlog"
	statemachine "../stateMachine"
	turnofflights "../turn_off_lights"
)

var numFloors = 4

//every elevator run compute score to see who is most suitable to take new order
func ComputeScore(dir int, order elevio.ButtonEvent, atFloor int, idle bool) int {
	N := numFloors - 1
	d := int(math.Abs(float64(atFloor - order.Floor)))
	if idle {
		return (N + 3) - d
	}
	//towards call
	if ((dir == 1) == (order.Floor > atFloor)) && ((dir == 1) == (order.Floor >= atFloor)) {
		if order.Button == 0 { //same dir button
			return (N + 2) - d
		} else if order.Button == 1 { //opposite dir button
			return (N + 1) - d
		}
	}
	//away from call
	return 1
}

func HandleHallCall(order elevio.ButtonEvent, score int) {
	atFloor := statemachine.GetFloor()
	if atFloor == order.Floor && statemachine.GetDirection() == 0 {
		elevio.SetDoorOpenLamp(true)
		time.Sleep(2 * time.Second)
		elevio.SetDoorOpenLamp(false)
		turnofflights.TurnOffLightTransmit(statemachine.GetFloor())
	} else if score > 1 {
		slog.StoreInSessionLog(order.Floor, 1)
		SetLights(order)
	} else {
		slog.StoreInSessionLog(order.Floor, 0)
		SetLights(order)
	}
}

func SetLights(order elevio.ButtonEvent) {
	elevio.SetButtonLamp(order.Button, order.Floor, true)
}
