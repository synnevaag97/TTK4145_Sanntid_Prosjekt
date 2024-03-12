
#To set up: 

Go get 	github.com/google/uuid

Go get	github.com/xtaci/kcp-go/v5

---

#Parameters: 

id: Elevator id. Standard: 0

elevport: The port the elevator is accessible from. Standard: 15657

superport: The port the supervisor is listening on, should not be edited. Standard: 80001

peerport: The port that the elevators are broadcasting their alivesignal on. Standard: 38257

init: If the elevator has been run before, should be set to true. Standard: False
