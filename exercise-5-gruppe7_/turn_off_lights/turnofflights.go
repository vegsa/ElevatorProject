package turn_off_lights

import (
	elevio "../elev_driver"
	"../network_module/drivers/bcast"
)

// Turns off light in the other elevators workspace.

var turnOffId int

var lastMessageId int

type LightOff struct {
	Floor     int
	MessageId int
}

var turnOffLightTX = make(chan LightOff)
var turnOffLightRX = make(chan LightOff)

func InitTurnOffLights() {
	go bcast.Transmitter(20013, turnOffLightTX)
	go bcast.Receiver(20014, turnOffLightRX)
}

func TurnOffLightTransmit(floor int) {
	var lightTransmit LightOff
	lightTransmit.Floor = floor
	turnOffId = turnOffId + 1
	lightTransmit.MessageId = turnOffId
	for i := 0; i < 5; i++ {
		turnOffLightTX <- lightTransmit
	}
}

func TurnOffLightReceive() {
	for {
		select {
		case turnOff := <-turnOffLightRX:
			if turnOff.MessageId == lastMessageId {
				break
			}
			lastMessageId = turnOff.MessageId
			elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_HallUp), turnOff.Floor, false)
			elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_HallDown), turnOff.Floor, false)
		}
	}
}
