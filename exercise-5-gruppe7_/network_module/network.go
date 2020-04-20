package network

import (
	"flag"
	"fmt"
	"os"
	"time"

	elevio "../elev_driver"
	hallcall "../hall_call_handler"
	statemachine "../stateMachine"
	"./drivers/bcast"
	"./drivers/localip"
)

// ID to the elevator.
var id string

//Used to handle the package loss. IDs on the different messages.
var orderId int

var lastOrderId int

var stateId int

var takeOrderMessageId int

var sendOrderMessageId int

var lastMessageId int

var stateIdCheck int

type Score struct {
	Id    string
	Score int
}

type Neworder struct {
	Order   elevio.ButtonEvent
	Floor   int
	Dir     int
	Id      string
	Idle    bool
	OrderId int
}

type StateMsg struct {
	Dir     int
	Id      string
	Idle    bool
	AtFloor int
	StateId int
}

type takeOrder struct {
	ElevId    string
	MessageId int
}

var orderTx = make(chan Neworder)
var orderRx = make(chan Neworder)
var stateTX = make(chan StateMsg)
var stateRX = make(chan StateMsg)
var takeOrderTX = make(chan takeOrder)
var takeOrderRx = make(chan takeOrder)

func InitNetwork() {
	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	// ... or alternatively, we can use the local IP address.
	// (But since we can run multiple programs on the same PC, we also append the
	//  process ID)
	if id == "" {
		localIP, err := localip.LocalIP()
		fmt.Println(localIP)
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}
	fmt.Println("ID 1:", id)

	// We make channels for sending and receiving our custom data types
	go bcast.Transmitter(20007, orderTx)
	go bcast.Receiver(20008, orderRx)
	go bcast.Transmitter(20009, stateTX)
	go bcast.Receiver(20010, stateRX)
	go bcast.Transmitter(20011, takeOrderTX)
	go bcast.Receiver(20012, takeOrderRx)
}

func transmitStates(stateId int) {
	var sm StateMsg
	sm.Id = id
	sm.AtFloor = statemachine.GetFloor()
	sm.Dir = statemachine.GetDirection()
	sm.Idle = statemachine.IsIdle()
	sm.StateId = stateId
	stateTX <- sm
}

// Receives the orders from the other elevator and communicates with it.
func NetworkReceive() {
	ElevScore1 := Score{
		Id:    id,
		Score: -100,
	}
	ElevScore2 := Score{
		Id:    id,
		Score: -100,
	}

	var takeO takeOrder

	//Check for new messages
	for {
		select {
		case order := <-orderRx:
			if order.OrderId == lastOrderId {
				break
			}
			lastOrderId = order.OrderId
			stateId = stateId + 1
			for i := 0; i < 5; i++ {
				transmitStates(stateId)
			}
			ElevScore1.Score = hallcall.ComputeScore(statemachine.GetDirection(), order.Order, statemachine.GetFloor(), statemachine.IsIdle())
			ElevScore1.Id = id
			ElevScore2.Score = hallcall.ComputeScore(order.Dir, order.Order, order.Floor, order.Idle)
			ElevScore2.Id = order.Id
			takeId := ""
			takeOrderMessageId = takeOrderMessageId + 1
			takeO.MessageId = takeOrderMessageId
			if ElevScore2.Score >= ElevScore1.Score {
				takeO.ElevId = ElevScore2.Id
				for i := 0; i < 5; i++ {
					takeOrderTX <- takeO
				}
				takeId = ElevScore2.Id
			} else {
				takeO.ElevId = ElevScore1.Id
				for i := 0; i < 5; i++ {
					takeOrderTX <- takeO
				}
				takeId = ElevScore1.Id
			}
			if takeId == id {
				to := 1
				for to == 1 {
					select {
					case takeOrder := <-takeOrderRx:
						if takeOrder.MessageId != lastMessageId {
							lastMessageId = takeOrder.MessageId
							to = 2
							if takeOrder.ElevId == id {
								fmt.Println("Take order2 receive")
								hallcall.HandleHallCall(order.Order, ElevScore1.Score)
							}
						}
					}
				}
			} else {
				elevio.SetButtonLamp(order.Order.Button, order.Order.Floor, true)
			}
		}
	}
}

// If we get a new order this sends the order and communicates
// with the other elevator.
func NetworkTransmit(order elevio.ButtonEvent) {
	var newOrder Neworder
	var takeO takeOrder

	newOrder.Order = order
	newOrder.Floor = statemachine.GetFloor()
	newOrder.Dir = statemachine.GetDirection()
	newOrder.Id = id
	newOrder.Idle = statemachine.IsIdle()
	orderId = orderId + 1
	newOrder.OrderId = orderId

	ElevScore1 := Score{
		Id:    id,
		Score: -100,
	}
	ElevScore2 := Score{
		Id:    id,
		Score: -100,
	}

	//Send new order
	for i := 0; i < 5; i++ {
		orderTx <- newOrder
	}
	runLoop := 1

	disconn := time.Tick(200 * time.Millisecond)
	for runLoop == 1 {
		select {
		// Communication with the other elevator.
		case states := <-stateRX:
			if states.StateId == stateIdCheck {
				break
			}
			stateIdCheck = states.StateId
			runLoop = 2
			ElevScore1.Score = hallcall.ComputeScore(newOrder.Dir, order, newOrder.Floor, newOrder.Idle)
			ElevScore1.Id = id
			ElevScore2.Score = hallcall.ComputeScore(states.Dir, order, states.AtFloor, states.Idle)
			ElevScore2.Id = states.Id

			loop := 1
			for loop == 1 {
				select {
				case takeOrder := <-takeOrderRx:
					if takeOrder.MessageId != lastMessageId {
						lastMessageId = takeOrder.MessageId
						loop = 2
						if takeOrder.ElevId == id {
							hallcall.HandleHallCall(order, ElevScore1.Score)
						} else {
							sendOrderMessageId = sendOrderMessageId + 1
							takeO.MessageId = sendOrderMessageId
							takeO.ElevId = ElevScore2.Id
							for i := 0; i < 5; i++ {
								takeOrderTX <- takeO
							}
							elevio.SetButtonLamp(order.Button, order.Floor, true)
						}
					}
				}
			}
		// If the other elevator does not answer.
		case <-disconn:
			NetworkDisconnect(order)
			runLoop = 2
		}
	}
}

// If the other elevator does not answer this handles the order.
func NetworkDisconnect(order elevio.ButtonEvent) {
	atFloor := statemachine.GetFloor()
	motorDir := statemachine.GetDirection()
	idle := statemachine.IsIdle()
	score := hallcall.ComputeScore(motorDir, order, atFloor, idle)
	hallcall.HandleHallCall(order, score)
}

// If elevator "power" is disconnected this handles and sends the
// orders over to the other elevator. Works almost as NetworkTransmit
// but this does not have the ability to take the order.
func ElevDisconnect(order elevio.ButtonEvent) {
	var newOrder Neworder
	var takeO takeOrder

	newOrder.Order = order
	newOrder.Floor = 100 						// Sets this to a high value to make sure that the score is very low when the order is handled.
	newOrder.Dir = statemachine.GetDirection()
	newOrder.Id = id
	newOrder.Idle = statemachine.IsIdle()
	orderId = orderId + 1
	newOrder.OrderId = orderId

	for i := 0; i < 5; i++ {
		orderTx <- newOrder
	}

	runLoop := 1
	for runLoop == 1 {
		select {

		case states := <-stateRX:
			if states.StateId == stateIdCheck {
				break
			}
			runLoop = 2
			loop := 1
			for loop == 1 {
				select {
				case takeOrder := <-takeOrderRx:
					if takeOrder.MessageId != lastMessageId {
						loop = 2
						lastMessageId = takeOrder.MessageId
						sendOrderMessageId = sendOrderMessageId + 1
						takeO.MessageId = sendOrderMessageId
						takeO.ElevId = states.Id
						for i := 0; i < 5; i++ {
							takeOrderTX <- takeO
						}
					}
				}
			}
		}
	}
}
