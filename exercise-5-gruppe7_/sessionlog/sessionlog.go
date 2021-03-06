package session_log

//********
// Handles the local file, "log", containing orders that are being executed now and in the future.
// Also contains a log executer that executes the physical actions on the elevator.
//********

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"time"

	elevio "../elev_driver"
	statemachine "../stateMachine"
	turnoff "../turn_off_lights"
)

var disconnect bool
var thisSession int
var nextSessions int
var path = "./log.txt"
var log []byte
var initFlag = true
var logMutex = &sync.Mutex{}

//Simple bubble sort to sort orders in log
func sortOrders(numbers []byte, direction int) []byte {
	for i := len(numbers); i > 0; i-- {
		for j := 1; j < i; j++ {
			if direction == 1 {
				if numbers[j-1] > numbers[j] {
					intermediate := numbers[j]
					numbers[j] = numbers[j-1]
					numbers[j-1] = intermediate
				}
			} else if direction == -1 {
				if numbers[j-1] < numbers[j] {
					intermediate := numbers[j]
					numbers[j] = numbers[j-1]
					numbers[j-1] = intermediate
				}
			}

		}
	}
	return numbers
}

//Make sure that the new order is not redundant with something already in the log
func newOrder(slice []byte, val byte) bool {
	for _, item := range slice {
		if item == val {
			return false
		}
	}
	return true
}

//Returns the current session log read from disk
func GetSessionLog() []byte {
	logMutex.Lock()
	data, _ := ioutil.ReadFile("log.txt")
	logMutex.Unlock()
	return data
}

//Take a new order and store it to disk if it is not already there.
//place == 1 for orders that will be executed in this session (e.g a new order for floor 3 if elev is moving from 1 -> 4)
//place == 0 for orders that is assigned to the elevator will not be executed in this session (e.g. new order floor 1 while elev moving from 2 -> 4)
func StoreInSessionLog(order int, placeInSession bool) {
	if createFile() == true {
		log = GetSessionLog()
		thisSession = int(log[0])
		if thisSession >= len(log) {
			thisSession = len(log) - nextSessions - 1
		}
		//put order in right place
		if newOrder(log[1:], byte(order)) {
			if placeInSession == true {
				thisSession = thisSession + 1
				println(thisSession)
				log = append([]byte{byte(thisSession)}, log...)
				log[1] = byte(order)
				sortOrders(log[1:thisSession+1], statemachine.GetDirection())
			} else if placeInSession == false {
				nextSessions = nextSessions + 1
				log = append(log, byte(order))
				if statemachine.GetDirection() == 1 {
					sort.Slice(log[thisSession+1:1+thisSession+nextSessions], func(i, j int) bool { return log[i] < log[j] })
				} else if statemachine.GetDirection() == -1 {
					sort.Slice(log[thisSession+1:1+thisSession+nextSessions], func(i, j int) bool { return log[i] < log[j] })
				}
			}
			fmt.Println("Queue:", log)
			logMutex.Lock()
			_ = ioutil.WriteFile(path, log, 0644)
			logMutex.Unlock()
		}

	}
}

// create "log.txt" file if it does not exists
func createFile() bool {
	var _, err = os.Stat(path)
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		ioutil.WriteFile(path, []byte{0}, 0644)
		if isError(err) {
			return false
		}
		defer file.Close()
		return true
	}
	return true
}

//Delete the first order in log and save to disk
func DeleteOrder() {
	sLog := GetSessionLog()
	copy(sLog[1:], sLog[2:])
	sLog = sLog[:len(sLog)-1]
	thisSession -= 1
	sLog[0] = byte(thisSession)
	_ = ioutil.WriteFile(path, sLog, 0644)
	if len(sLog) < 1 {
		statemachine.SetIdle()
	}
}

//Continously checking if any new orders have been stored in SessionLog and make the elev move to desired floor
func LogExecuter() {
	for {
		if statemachine.GetNewFloor() {
			sLog := GetSessionLog()
			if len(sLog) > 1 {
				if statemachine.GetFloor() == int(sLog[1]) {
					DeleteOrder()
					elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), statemachine.GetFloor(), false)
					elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_HallUp), statemachine.GetFloor(), false)
					elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_HallDown), statemachine.GetFloor(), false)
					turnoff.TurnOffLightTransmit(statemachine.GetFloor())

					elevio.SetMotorDirection(elevio.MD_Stop)
					elevio.SetDoorOpenLamp(true)
					time.Sleep(2 * time.Second)
					elevio.SetDoorOpenLamp(false)
					sLog = GetSessionLog()
					if thisSession >= 1 && len(sLog) > 1 {
						statemachine.SettoFloor(int(sLog[1]))
						elevio.SetMotorDirection(elevio.MotorDirection(statemachine.GetDirection()))
					} else if nextSessions >= 1 {
						thisSession = nextSessions
						nextSessions = 0
						sLog[0] = byte(thisSession)
						statemachine.SettoFloor(int(sLog[1]))
						_ = ioutil.WriteFile(path, sLog, 0644)
						elevio.SetMotorDirection(elevio.MotorDirection(statemachine.GetDirection()))
					}
				}
			}
			statemachine.SetNewFloor(false)
		} else if statemachine.IsIdle() {
			sLog := GetSessionLog()
			if len(sLog) > 1 {
				statemachine.SettoFloor(int(sLog[1]))
				elevio.SetMotorDirection(elevio.MotorDirection(statemachine.GetDirection()))
			}

		}
		time.Sleep(20 * time.Millisecond)
	}
}

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}

	return (err != nil)
}

// Reconnects the elevator "power"
func Reconnect() {
	disconnect = false
	sLog := GetSessionLog()
	if len(sLog) > 1 {
		statemachine.SettoFloor(int(sLog[1]))
		elevio.SetMotorDirection(elevio.MotorDirection(statemachine.GetDirection()))
	}
}

func GetDisconnect() bool {
	return disconnect
}

// Checks if elevator is disconnected from "power"
func CheckDisconnect() bool {
	for {
		counter := 0
		//for (elevio.GetFloor() == -1 ) {
		sLog := GetSessionLog()
		for len(sLog) > 1 && statemachine.GetDirection() != 0 {
			counter = counter + 1
			if counter > 10 {
				disconnect = true
				return true
			}
			time.Sleep(1 * time.Second)
		}
	}
}
