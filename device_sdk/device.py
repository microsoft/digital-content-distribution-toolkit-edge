from __future__ import print_function

import configparser
from concurrent import futures
# from multiprocessing import Process
import threading
import time
import sys
import fcntl
import os
import grpc
import datetime
import json

import logger_pb2
import logger_pb2_grpc
import commands_pb2
import commands_pb2_grpc

from azure.iot.device import IoTHubDeviceClient, Message, ProvisioningDeviceClient
from azure.iot.device import IoTHubDeviceClient, Message, MethodResponse

config = configparser.ConfigParser()
symmetric_key = None
registration_id = None
id_scope = None
capabilityModelId = None
capabilityModel = None 
provisioning_host = None

def lock_file(f):
    fcntl.lockf(f, fcntl.LOCK_EX)

def unlock_file(f):
    fcntl.lockf(f, fcntl.LOCK_UN)

class AtomicOpen:
    """
    Generic class to implement context to open file with lock and flush, release lock before closing
    """
    def __init__(self, path, *args, **kwargs):
        self.file = open(path,*args, **kwargs)
        lock_file(self.file)

    def __enter__(self, *args, **kwargs):
        return self.file

    def __exit__(self, exc_type=None, exc_value=None, traceback=None):        
        self.file.flush()  # Flush to make sure all buffered contents are written to file
        os.fsync(self.file.fileno())  # Release the lock on the file
        unlock_file(self.file)
        self.file.close()
        # Handle exceptions that may have come up during execution, by default any exceptions are raised to the user.
        if(exc_type != None):
            return False
        else:
            return True

def register_device():
    provisioning_host = config.get('host', 'provisioningHost')
    symmetric_key = config.get('section_device', 'sas_key')
    registration_id = config.get('section_device','deviceId')
    id_scope = config.get('section_device','scope')
    capabilityModelId=config.get('section_device','capabilityModelId')
    capabilityModel = "{iotcModelId : '" + capabilityModelId + "'}" 
    
    provisioning_device_client = ProvisioningDeviceClient.create_from_symmetric_key(
        provisioning_host=provisioning_host,
        registration_id=registration_id,
        id_scope=id_scope,
        symmetric_key=symmetric_key,
    )
    provisioning_device_client.provisioning_payload = capabilityModel
    registration_result = provisioning_device_client.register()
    print('Registration result: {}'.format(registration_result.status))
    return registration_result

def connect_device():
    device_client = None
    symmetric_key = config.get('section_device', 'sas_key')
    try:
      registration_result = register_device()
      if registration_result.status == 'assigned':
        device_client = IoTHubDeviceClient.create_from_symmetric_key(
          symmetric_key=symmetric_key,
          hostname=registration_result.registration_state.assigned_hub,
          device_id=registration_result.registration_state.device_id,
        )
        # Connect the client.
        device_client.connect()
        print('Device connected successfully')
    finally:
      return device_client

def init_device(device_client):
    # device properties
    processorArchitecture= config.get('section_device','processorArchitecture')
    softwareVersion= config.get('section_device','softwareVersion')
    totalMemory= config.getint('section_device','totalMemory')
    totalStorage= config.getint('section_device','totalStorage')
    processorManufacturer= config.get('section_device','processorManufacturer')
    osName = config.get('section_device','osName')
    manufacturer= config.get('section_device','manufacturer')
    model= config.get('section_device','model')
    config.read('customerdetails.ini')
    customerName=config.get('customer_details', 'customer_name')
    location=config.get('customer_details', 'location')
    
    device_client.patch_twin_reported_properties({'location':location})
    device_client.patch_twin_reported_properties({'name':customerName})
    device_client.patch_twin_reported_properties({'model':model})
    device_client.patch_twin_reported_properties({'swVersion':softwareVersion})
    device_client.patch_twin_reported_properties({'osName':osName})
    device_client.patch_twin_reported_properties({'processorManufacturer':processorManufacturer})
    device_client.patch_twin_reported_properties({'totalMemory':totalMemory})
    device_client.patch_twin_reported_properties({'totalStorage':totalStorage})
    device_client.patch_twin_reported_properties({'manufacturer':manufacturer})
    device_client.patch_twin_reported_properties({'processorArchitecture': processorArchitecture})
    
def iothub_client_init():
    client = connect_device()
    return client
  
def send_upstream_messages(iot_client):
    while True:
        try:  # keep on spinning this even in case of error in production
            with AtomicOpen(config.get("LOGGER", "LOG_FILE_PATH"), "r+") as fout:
                temp = [line.strip() for line in fout.readlines()]
                fout.seek(0)
                fout.write("")
                fout.truncate()
            
            # print(temp)
            for x in temp:
                if(len(x) != 0):
                    message = Message(x)
                    print(message)
                    iot_client.send_message(message)
        except Exception as ex:
            message = Message(str({"DeviceId": config.get("DEVICE_INFO", "DEVICE_NAME"), "MessageType": "Critical", "MessageSubType": "DeviceSDK", "MessageBody": {"Message": "exception in send_upstream_messages in deivce SDK {}".format(ex)}}))
            print(message)
            iot_client.send_message(message)

def getFileParams(param):
    fileparams = []
    if param is not None:
        result = param.split(";")
        for x in result:
            y = x.split(",")
            fileparams.append(commands_pb2.File(name=y[0], cdn=y[1], hashsum=y[2]))
    return fileparams

def command_listener(iot_client):
    print("Listening for device commands...")
    with grpc.insecure_channel('localhost:{}'.format(config.getint("GRPC", "DOWNSTREAM_PORT"))) as channel:
        stub = commands_pb2_grpc.RelayCommandStub(channel)
        while True:
            method_request = iot_client.receive_method_request()
            print (
                "\nMethod callback called with:\nmethodName = {method_name}\npayload = {payload}".format(
                    method_name=method_request.name,
                    payload=method_request.payload
                )
            )
            if method_request.name == "SetTelemetryInterval":
                try:
                    INTERVAL = int(method_request.payload)
                    print("\nSet Telemetry Interval : {}".format(INTERVAL))
                except ValueError:
                    response_payload = {"Response": "Invalid parameter"}
                    response_status = 400
                else:
                    response_payload = {"Response": "Executed direct method {}".format(method_request.name)}
                    response_status = 200
            elif(method_request.name == "Download"):
                print("Sending request to downlaoad")
                payload = eval(method_request.payload)
                try:
                    _folder_path = payload['folder_path']
                    _metadata_files = getFileParams(payload['metadata_files'])
                    _bulk_files = getFileParams(payload['bulk_files'])
                    _channels = [commands_pb2.Channel(channelname=x) for x in payload['channels'].split(";")]
                    _deadline = int(payload['deadline'])
                    print(dict(folderpath=_folder_path, metadatafiles=_metadata_files,
                    bulkfiles=_bulk_files, channels=_channels, deadline=_deadline
                    ))
                    
                    download_params = commands_pb2.DownloadParams(folderpath=_folder_path, metadatafiles=_metadata_files,
                    bulkfiles=_bulk_files, channels=_channels, deadline=_deadline)
                    response = stub.Download(download_params)
                    print(response)
                except Exception as ex:
                    print("Exception {}".format(ex))
                    response_payload = {"Response": "Error in sending from device SDK: {}".format(ex)}
                    response_status = 400
                else:
                    response_payload = {"Response": "Executed method  call {}".format(method_request.name)}
                    response_status = 200
            
                print('Turning off the LED')
                method_response = MethodResponse.create_from_method_request(
                method_request, status = 200
                )
            elif(method_request.name == "Delete"):
                print("Sending request to delete", method_request.payload)
                payload = eval(method_request.payload)
                try:
                    _folder_path = payload['folder_path']
                    _recursive = bool(payload['recursive'])
                    _delete_after = int(payload['delete_after'])
                    delete_params = commands_pb2.DeleteParams(folderpath=_folder_path, recursive=_recursive,
                    delteafter=_delete_after)
                    response = stub.Delete(delete_params)
                    print(response)
                except Exception as ex:
                    print("Exception {}".format(ex))
                    response_payload = {"Response": "Error in sending from device SDK: {}".format(ex)}
                    response_status = 400
                else:
                    response_payload = {"Response": "Executed method  call {}".format(method_request.name)}
                    response_status = 200            
            else:
                response_payload = {"Response": "Method call {} not defined".format(method_request.name)}
                response_status = 404
            method_response = MethodResponse(method_request.request_id, response_status, payload=response_payload)
            iot_client.send_method_response(method_response)

if __name__ == '__main__':
    config.read('hub_config.ini')
    print(config.sections())
    # device and host configuration are stored in the config file
    config.read('device.ini')
    provisioning_host = config.get('host', 'provisioningHost')
    symmetric_key = config.get('section_device', 'sas_key')
    registration_id = config.get('section_device','deviceId')
    id_scope = config.get('section_device','scope')
    
    # capability Model
    capabilityModelId=config.get('section_device','capabilityModelId')
    capabilityModel = "{iotcModelId : '" + capabilityModelId + "'}" 
    iot_client = iothub_client_init()
    init_device(iot_client)
    
    if iot_client is not None and iot_client.connected:
      print('Send reported properties on startup')
      iot_client.patch_twin_reported_properties({'state': 'true'})
      print("Listen for method calls...")
      t = threading.Thread(target=command_listener, args=[iot_client])
      t.daemon = True
      t.start()
      
      print("Send Upstream messages...")
      send_upstream_messages(iot_client)
    else:
      print('Device could not connect')