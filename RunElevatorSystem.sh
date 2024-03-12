Help()
{
   # Display Help
   echo "=========================================================="
   echo "Run the Elevator System program"
   echo "=========================================================="
   echo "Syntax: ./RunElevatorSystem nodeID port"
   echo
   echo "parameters:"
   echo "nodeID - The node name of the elevator system"
   echo "port - The port number to with the elevator system is connecting to"
   echo "=========================================================="
}
ready=false

if [[ $1 == "" ]]
    then
        Help
    else
        ready=true
fi

while $ready
do
    if [[ $2 == "" ]]
    then
        echo " --- Starting node $1 on default port ---"
        ./elevatorSystem -id=$1
    else
        echo " --- Starting node $1 on port $2 ---"
        ./elevatorSystem -id=$1 -port=$2
    
    fi
done