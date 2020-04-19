package order_handler

//********
//Have a routine to receive local button pushes. If they are cab calls they are handled in the package. Hall calls are transmitted to all elevators.
//********

import (
	"fmt"
	"time"

	elevio "../elev_driver"
	network "../network_module"
	slog "../sessionlog"
	statemachine "../stateMachine"
)

var numFloors = 4

func handleCabcall(order elevio.ButtonEvent) {
	idle := statemachine.IsIdle()
	motorDir := statemachine.GetDirection()
	atFloor := statemachine.GetFloor()
	if atFloor == order.Floor && elevio.GetFloor() != -1 {
		elevio.SetDoorOpenLamp(true)
		time.Sleep(2 * time.Second)
		elevio.SetDoorOpenLamp(false)
	} else if idle == true || (((motorDir == 1) == (order.Floor > atFloor)) && ((motorDir == 1) == (order.Floor >= atFloor))) {
		slog.StoreInSessionLog(order.Floor, true)
		elevio.SetButtonLamp(order.Button, order.Floor, true)
	} else {
		slog.StoreInSessionLog(order.Floor, false)
		elevio.SetButtonLamp(order.Button, order.Floor, true)
	}
}

func CheckButtons() {
	drv_buttons := make(chan elevio.ButtonEvent)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	for {
		select {
		case a := <-drv_buttons:
			fmt.Println("Cab call")
			if a.Button == 2 {
				handleCabcall(a)
			} else {
				fmt.Println("Hall call")
				if slog.GetDisconnect() == true {
					network.ElevDisconnect(a)
				} else {
					network.NetworkTransmit(a)
				}
			}

		case a := <-drv_obstr:
			fmt.Printf("obstruct %+v\n", a)
			if a == false {
				slog.Reconnect()
			}

		case a := <-drv_stop:
			fmt.Printf("stop %+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					elevio.SetButtonLamp(b, f, false)
				}
			}
		}
	}
}
