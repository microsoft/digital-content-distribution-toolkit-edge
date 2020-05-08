import zmq
import random
import sys
import time
import threading
from azure.iot.device import IoTHubDeviceClient, Message

CONNECTION_STRING = "HostName=gohub.azure-devices.net;DeviceId=MyPythonDevice;SharedAccessKey=zA2DAirXTqJ0TGkpf+8fTLYVCxC3YlPJsO+UUO2QS98="

def iothub_client_init():
    # Create an IoT Hub client
    client = IoTHubDeviceClient.create_from_connection_string(CONNECTION_STRING)
    return client

def iothub_client_telemetry_sample_run(client, zmq_port="5555"):
    try:
        print ( "IoT Hub device sending periodic messages, press Ctrl-C to exit" )

        while True:
            # Build the message with simulated telemetry values.
            message = Message(msg_txt_formatted)

            # Add a custom application property to the message.
            # An IoT hub can filter on these properties without access to the message body.
            # if temperature > 30:
            #   message.custom_properties["temperatureAlert"] = "true"
            # else:
            #   message.custom_properties["temperatureAlert"] = "false"

            # Send the message.
            print( "Sending message: {}".format(message) )
            client.send_message(message)
            print ( "Message successfully sent" )
            time.sleep(10)

    except KeyboardInterrupt:
        print ( "IoTHubClient sample stopped" )
    
def get_message(file_path):
    with open(file_path, 'r') as file_handle:
        return file_handle.read()

def get_download_command(contents):
    return "DOWNLOAD_CMD" + contents

def get_delete_command(contents):
    return "DELETE_CMD" + contents

class Log(threading.Thread):
    def __init__(self, socket, client):
        threading.Thread.__init__(self)
        self.socket = socket
        self.client = client

    def run(self):
        logs = [] # batch send logs, every 100 maybe
        print("Waiting to receive messsage from GO\n")
        while True:
            log_from_go = self.socket.recv()
            logs.append(log_from_go)
            if len(logs) > 100:
                for log in logs:
                    message = Message(log)
                    self.client.send_message(message)
                logs = []
                print("FLUSHED 100 messages")

class IOTHub(threading.Thread):
    def __init__(self):
        threading.Thread.__init__(self)
    
    def run(self):
        port = "5555"
        context = zmq.Context()
        socket = context.socket(zmq.PAIR)
        socket.bind("tcp://*:%s" % port)
        # Send download command
        # download_message = get_download_command(get_message('./test/download-ars-season.json'))
        # socket.send_string(download_message)
        
        client = iothub_client_init()
        logging_thread = Log(socket, client)
        logging_thread.daemon = True
        logging_thread.start()
        print("Waiting to receive message from Azure IOT HUB\n")
        while True:
            message = client.receive_message()
            for property in vars(message).items():
                if property[0] == 'data':
                    socket.send_string(str(property[1]))


iotHub_thread = IOTHub()
iotHub_thread.daemon = True
iotHub_thread.start()

# Start this to listen to interrupts
while True:
    time.sleep(10)