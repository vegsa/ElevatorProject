package sessionlog

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"
	"sync"

	elevio "../elev_driver"
	statemachine "../stateMachine"
	turnoff "../turn_off_lights"
)

var disconnect bool
var thisOrder int
var nextOrders int
var path = "./log.txt"
var log []byte //{[]byte,int}
var initFlag =true
var mutex = &sync.Mutex{}

func sortOrders(numbers []byte,direction int) []byte {
    //Start the loop in reverse order, so the loop will start with length
    //which is equal to the length of input array and then loop untill   //reaches 1
    for i := len(numbers); i > 0; i-- {
       //The inner loop will first iterate through the full length
       //the next iteration will be through n-1
       // the next will be through n-2 and so on
       for j := 1; j < i; j++ {
            if direction== 1{
                if numbers[j-1] > numbers[j] {
                    intermediate := numbers[j]
                    numbers[j] = numbers[j-1]
                    numbers[j-1] = intermediate
                }    
            }else if direction == -1{
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

func newOrder(slice []byte, val byte) (bool) {
    for _, item := range slice {
        if item == val {
            return false
        }
    }
    return true
}

func GetSessionLog() []byte {
	mutex.Lock()
	data, _ := ioutil.ReadFile("log.txt")
	mutex.Unlock()
	return data
}

// When we start the program with orders in the log, thisOrder has the 
// value 254 when we add a order, which will give a fatal error
// when we sort the orders because then we say that we are going to sort
// the 256 elements in log which is not true.

func StoreInSessionLog(order int, place int) {
	if createFile() == true {
		log = GetSessionLog()
		thisOrder = int(log[0])
		if thisOrder >= len(log) {
			thisOrder = len(log) - nextOrders - 1
		}
		//put order in right place
		if newOrder(log[1:],byte(order)){
			if place == 1 {
				thisOrder = thisOrder + 1
				println(thisOrder)
				log = append([]byte{byte(thisOrder)}, log...)
				log[1] = byte(order) // put in 2 position
				sortOrders(log[1:thisOrder+1],statemachine.GetDirection())
				fmt.Println("sorted")
			} else if place == 0 {
				nextOrders = nextOrders + 1
				log = append(log, byte(order))
				if statemachine.GetDirection() == 1 {
					sort.Slice(log[thisOrder+1:1+thisOrder+nextOrders], func(i, j int) bool { return log[i] < log[j] })
				} else if statemachine.GetDirection() == -1 {
					sort.Slice(log[thisOrder+1:1+thisOrder+nextOrders], func(i, j int) bool { return log[i] < log[j] })
				}
			}
			fmt.Println("Queue:",log)
			mutex.Lock()
			_ = ioutil.WriteFile(path, log, 0644)
			mutex.Unlock()
		}
		
	}
}

func DeleteOrder() {
	sLog := GetSessionLog()
	copy(sLog[1:], sLog[2:])  // Shift a[i+1:] left one index.
	sLog = sLog[:len(sLog)-1] // Truncate slice.
	thisOrder -= 1
	fmt.Println("To", thisOrder)
	sLog[0] = byte(thisOrder)
	_ = ioutil.WriteFile(path, sLog, 0644)
	if len(sLog) < 1 {
		statemachine.SetIdle()
	}
}

func QueueWatchdog() {
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
					if thisOrder >= 1 && len(sLog) > 1{
						statemachine.SettoFloor(int(sLog[1]))
						elevio.SetMotorDirection(elevio.MotorDirection(statemachine.GetDirection()))
					} else if nextOrders >= 1 {
						thisOrder = nextOrders
						nextOrders = 0
						sLog[0] = byte(thisOrder)
						statemachine.SettoFloor(int(sLog[1]))
						_ = ioutil.WriteFile(path, sLog, 0644)
						elevio.SetMotorDirection(elevio.MotorDirection(statemachine.GetDirection()))
					}
				}
			}
			statemachine.SetNewFloor(false)
		} else if statemachine.IsIdle(){
			sLog := GetSessionLog()
			if len(sLog) > 1 {
				statemachine.SettoFloor(int(sLog[1]))
				elevio.SetMotorDirection(elevio.MotorDirection(statemachine.GetDirection()))
			}

		}
		time.Sleep(20 * time.Millisecond)
	}
}

func createFile() bool {
	var _, err = os.Stat(path)

	// create file if not exists
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
		for (len(sLog) > 1 && statemachine.GetDirection() != 0) {
			counter = counter + 1
			fmt.Println("Disconnect?")
			if counter > 10{
				disconnect = true
				return true
			} 
			time.Sleep(1*time.Second)
		}	
	}
}	
