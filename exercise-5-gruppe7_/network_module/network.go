package network

import (
	"flag"
	"fmt"
	"os"
	//"encoding/json"
	elevio "../elev_driver"
	statemachine "../stateMachine"
	//orderHandler "../order_handler"
	cs "../compute_score"
	//slog "../sessionlog"
	"./drivers/bcast"
	"./drivers/localip"
	"time"
	//"./network/peers"
)

var id string

//Used to handle the package loss. IDs on the different messages.
var orderId int

var lastOrder int

var stateId int

var toMessageId int

var messageTransmit int

var lastId int

var idState int

type Score struct {
	Id      string
	Score	int
}

type Neworder struct {
	Order	elevio.ButtonEvent
	Floor   int
	Dir 	int
	Id      string	
	Idle	bool
	OrderId int
}

type StateMsg struct {
	Dir		int
	Id 		string
	Idle	bool
	AtFloor	int
	StateId int
}

type takeOrder struct {
	ElevId		string
	MessageId	int
} 

var orderTx = make(chan Neworder)
var orderRx = make(chan Neworder)
var stateTX = make(chan StateMsg)
var stateRX = make(chan StateMsg)
var takeOrderTX = make(chan takeOrder)
var takeOrderRx = make(chan takeOrder)

func InitNetwork(){
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
	
	// ... and start the transmitter/receiver pair on some port
	// These functions can take any number of channels! It is also possible to
	//  start multiple transmitters/receivers on the same port.
	//go bcast.Transmitter(20007, scoreTx)
	go bcast.Transmitter(20007, orderTx)
	//go bcast.Receiver(20007, scoreRx)
	go bcast.Receiver(20008, orderRx)
	go bcast.Transmitter(20009, stateTX)
	go bcast.Receiver(20010, stateRX)
	go bcast.Transmitter(20011, takeOrderTX)
	go bcast.Receiver(20012, takeOrderRx)
}

func transmitStates(stateId int) {
	var	sm StateMsg
	sm.Id = id
	sm.AtFloor= statemachine.GetFloor()
	sm.Dir = statemachine.GetDirection()
	sm.Idle = statemachine.IsIdle()
	sm.StateId = stateId
	stateTX <-sm
}

// Receives the orders from the other elevator and communicates with it.
func NetworkReceive() {
	ElevScore1 := Score{
		Id: id,
		Score: -100, 
	}
	ElevScore2 := Score{
		Id: id,
		Score: -100,
	}

	var takeo takeOrder 

	//Check for new messages
	for {
		select {
		case order := <-orderRx:
			if order.OrderId == lastOrder {
				break
			}
			lastOrder = order.OrderId
			stateId = stateId + 1
			for i := 0; i < 5; i++ {
				transmitStates(stateId)
			}
			ElevScore1.Score = cs.ComputeScore(statemachine.GetDirection(),order.Order,statemachine.GetFloor(),statemachine.IsIdle())
			ElevScore1.Id = id
			ElevScore2.Score = cs.ComputeScore(order.Dir,order.Order,order.Floor,order.Idle)
			ElevScore2.Id = order.Id
			take := ""
			toMessageId = toMessageId + 1
			takeo.MessageId = toMessageId
			if (ElevScore2.Score >= ElevScore1.Score) {
				takeo.ElevId = ElevScore2.Id
				for i := 0; i < 5; i++ {
					takeOrderTX <- takeo
				}
				take = ElevScore2.Id
			} else {
				takeo.ElevId = ElevScore1.Id
				for i := 0; i < 5; i++ {
					takeOrderTX <- takeo
				}
				take = ElevScore1.Id
			}
			if take == id {
				to := 1
				for to == 1 {
					select {
					case takeOrder := <- takeOrderRx:
						if takeOrder.MessageId != lastId{
							lastId = takeOrder.MessageId
							to = 2
							if (takeOrder.ElevId == id) {
								fmt.Println("Take order2 receive")
								cs.HandleHallCall(order.Order, ElevScore1.Score)
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
	var takeo takeOrder

	newOrder.Order = order
	newOrder.Floor = statemachine.GetFloor()
	newOrder.Dir = statemachine.GetDirection()
	newOrder.Id = id
	newOrder.Idle = statemachine.IsIdle()
	orderId = orderId + 1
	newOrder.OrderId = orderId

	ElevScore1 := Score{
		Id: id,
		Score: -100, 
	}
	ElevScore2 := Score{
		Id: id,
		Score: -100,
	}
	
	//Send new order
	for i := 0; i < 5; i++ {
		orderTx <- newOrder
	}
	runloop := 1

	disconn := time.Tick(200*time.Millisecond)
	for runloop == 1 {
		select {
		// Communication with the other elevator.
		case states := <- stateRX: 
			if states.StateId == idState {
				break
			}
			idState = states.StateId
			runloop = 2
			ElevScore1.Score = cs.ComputeScore(newOrder.Dir, order,newOrder.Floor,newOrder.Idle)
			ElevScore1.Id = id
			ElevScore2.Score = cs.ComputeScore(states.Dir,order,states.AtFloor,states.Idle)
			ElevScore2.Id = states.Id
			
			to := 1
			for to == 1 {
				select {
				case takeOrder := <- takeOrderRx: 
					
					if takeOrder.MessageId != lastId{
						lastId = takeOrder.MessageId
						to = 2
						if (takeOrder.ElevId == id) {
							cs.HandleHallCall(order, ElevScore1.Score)
						} else{
							messageTransmit = messageTransmit + 1
							takeo.MessageId = messageTransmit
							takeo.ElevId = ElevScore2.Id
							for i := 0; i < 5; i++ {
								takeOrderTX <- takeo
							}
							elevio.SetButtonLamp(order.Button, order.Floor, true)
						}
					}
				}
			}
		// If the other elevator does not answer.
		case <- disconn:
			NetworkDisconnected(order)
			runloop = 2
		}
	} 
}

// If the other elevator does not answer this handles the order.
func NetworkDisconnected(order elevio.ButtonEvent) {
	atFloor := statemachine.GetFloor()
	motorDir := statemachine.GetDirection()
	idle := statemachine.IsIdle()
	score := cs.ComputeScore(motorDir, order, atFloor, idle)
	cs.HandleHallCall(order, score)
}

// If elevator "power" is disconnected this handles and sends the 
// orders over to the other elevator. Works almost as NetworkTransmit
// but this does not have the ability to take the order.
func ElevDisconnect(order elevio.ButtonEvent) {
	var newOrder Neworder
	var takeo takeOrder

	newOrder.Order = order
	newOrder.Floor = 50
	newOrder.Dir = statemachine.GetDirection()
	newOrder.Id = id
	newOrder.Idle = statemachine.IsIdle()
	orderId = orderId + 1
	newOrder.OrderId = orderId
	
	for i := 0; i < 5; i++ {
		orderTx <- newOrder
	}
	
	runloop := 1
	for runloop == 1 {
		select{

		case states := <- stateRX:
			if states.StateId == idState {
				break
			}
			runloop = 2
			to := 1
			for to == 1 {
				select {
				case takeOrder := <- takeOrderRx:
					if takeOrder.MessageId != lastId {
						to = 2
						lastId  = takeOrder.MessageId
						messageTransmit = messageTransmit + 1
						takeo.MessageId = messageTransmit
						takeo.ElevId = states.Id
						for i := 0; i < 5; i++ {
							takeOrderTX <- takeo
						}
					}
				}
			}
		}
	}
}