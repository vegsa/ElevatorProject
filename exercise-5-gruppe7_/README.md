Set up of the elevators:

Make a copy of the program such that we have two identical programs. Then we need to change the port of the simulator and program such that the programs corresponds to a simulator of its own. Then we need to change the port-numbers that the channels listens to in one of the two programs. This has to be done in InitNetwork and InitTurnOffLights such that the port the transmitter of orderTx in the first program corresponds to the port the reciever of orderRx is listening to in the second program. Each of these pairs have their own port. 


Packet Loss: 

The communication is buildt on that the two elevators sends each message between them 5-10 times (has to be decieded). Each message has a corresponding ID such that we can filter out the correct messages we need. The same messages has the same ID, where the IDs are ints which is incremented for each new message. Then it gets compared to a lastId etc. that keeps track of what the last ID at the recieving elevator was. Thus we can filter out the correct message and make sure that we handle the same message multiple times if multiple messages gets to the destination.


Disconnect of "network":

In NetworkTransmit we have a counter. If it does not get a response from the other elevator in 0.2 seconds the elevator will take the order itself by running NetworkDisconnected.


Disconnect of power to elevator:

We always check if the elevator is disconnected with the function CheckDisconnect() in sessionlog.go. If the elevator does not reach its destination within 10 seconds we set the variable disconnect = true. As long as disconnect is true, the orders handeled in orderhandler.go will be sent to the function ElevDisconnect() in network.go and not NetworkTransmit(). ElevDisconnect() will send all of the orders it gets to the other elevator since we know that the other elevator is working fine as we assume that at least one elevator works normally. We can reconnect the power to the elevator by pressing the obstruction button (-) twice, and then it should works as it did before it disconnected.

What to do:

- Check that everything works. There might be some errors and some cases that is not covered yet.
- Fix the code quality. Check what is good code quality and try to make our code as good as possible.

