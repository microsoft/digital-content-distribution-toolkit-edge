import random
import sys
from azure.iot.hub import IoTHubRegistryManager

MESSAGE_COUNT = 2
AVG_WIND_SPEED = 10.0
MSG_TXT = "{\"service client sent a message\": %.2f}"

CONNECTION_STRING = "HostName=gohub.azure-devices.net;SharedAccessKeyName=service;SharedAccessKey=/asbil6M5YnN72WCQOH6qGex/GmhEG1gkNLM5Vt9ArA="
DEVICE_ID = "MyPythonDevice"

def get_download_command(contents):
    return "DOWNLOAD_CMD" + contents

def iothub_messaging_sample_run():
    try:
        # Create IoTHubRegistryManager
        registry_manager = IoTHubRegistryManager(CONNECTION_STRING)
    
        print('Sending message')
        data = "unread"
        with open('../test/download-ars-season.json', 'r') as command_file:
            data = get_download_command(command_file.read())
        props={}
        # optional: assign system properties
        props.update(messageId = "message")
        props.update(correlationId = "correlation")
        props.update(contentType = "application/json")

        # optional: assign application properties
        prop_text = "PropMsg"
        props.update(testProperty = prop_text)

        registry_manager.send_c2d_message(DEVICE_ID, data, properties=props)

        try:
            # Try Python 2.xx first
            raw_input("Press Enter to continue...\n")
        except:
            pass
            # Use Python 3.xx in the case of exception
            input("Press Enter to continue...\n")

    except Exception as ex:
        print ( "Unexpected error {0}" % ex )
        return
    except KeyboardInterrupt:
        print ( "IoT Hub C2D Messaging service sample stopped" )

if __name__ == '__main__':
    print ( "Starting the Python IoT Hub C2D Messaging service sample..." )

    iothub_messaging_sample_run()