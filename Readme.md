## Hub built in GoLang for BINE

### Quick links
* [Dev environment setup](#setting-up-development-environment)
* [Run the Bine HUB setup](#running-the-bine-hub)
* [Code organization](#code-organization)
* [Testing the hub with java client](#testing-the-file-server)

### Setting up Development environment

#### Requirements on Ubuntu or Windows Subsystem for Linux
 *  Go language - https://golang.org/dl/
 *  Python 3 - https://www.python.org/downloads/

#### Fetching application dependencies after installing Go and Python
```
./setup_box.sh
```

### Running the BINE-HUB
```
./start_hub.sh
```

### Code organization
There are two main components to the BINE HUB - IoT Edge device written in Python and the file server written in Golang. These components communicate with each other using [ZeroMQ](https://zeromq.org/) sockets.

1. The IoT Edge device is responsible for speaking with azure iot hub to receive commands and report hub level telemetry to the cloud. Please refer to ```device_sdk``` for the code.

2. The File server has the following components 
    * ```hub.go``` sets up all the go routines and starts a http server.
    * ```database.go``` maintains a database to support the file server.
    * ```repository.go``` wrapper around the database to provide useful utility functions.
    * ```folder_checker.go``` Checks if there is any tampering with the cached files and purges them if necessary.
    * ```cloud_sync.go``` offers methods to download folders and delete folders and update the database accordingly.
    * ```route_handler.go``` creates and manages http routes to offer APIs to access the file server

### Testing the File server

Please clone the following repository and follow the instructions there to test the file server, once it is up and running - [TestClient](https://dev.azure.com/binemsr/Hub/_git/TestClient)