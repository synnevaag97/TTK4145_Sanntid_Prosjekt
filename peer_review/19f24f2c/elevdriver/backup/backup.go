package backup

import (
	"Elevdriver/data_structure"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func Backup(systemInfo data_structure.System_info_t,
	cab_request_to_backup_chan chan data_structure.Elevator_data_t) {

	//Save local_orders to backup
	for {
		select {
		case cab_requests := <-cab_request_to_backup_chan:
			backup, err := json.Marshal(cab_requests)
			if err != nil {
				fmt.Println("Backup: Failed to encode JSON file")
				break
			}

			err = ioutil.WriteFile("backup.json", backup, 0644)
			if err != nil {
				fmt.Print("Backup: Failed to open backup file")
			}
			break
		}
	}
}

func GetBackup(systemInfo data_structure.System_info_t) (data_structure.Elevator_data_t, error) {
	var cab_requests data_structure.Elevator_data_t
	if !systemInfo.Init {
		//Read backup if init is false
		read_data, err := ioutil.ReadFile("backup.json")
		if err != nil {
			fmt.Print("Backup: Failed to read backupfile")
			return cab_requests, err
		}

		err = json.Unmarshal(read_data, &cab_requests)
		if err != nil {
			fmt.Println("Backup: Failed to decode JSON file")
			return cab_requests, err
		}
	}
	return cab_requests, nil
}
