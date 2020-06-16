package main

//SatdataSwitch ... to switch between noovo and mstore API
func SatdataSwitch(cmd string) {
	switch cmd {
	case "noovo":
		go checkForVOD()
	case "mstore":
		go checkForVODViaMstore()
	}

}
