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
from collections import defaultdict

import commands_pb2
import commands_pb2_grpc

from azure.iot.device import IoTHubDeviceClient, Message, ProvisioningDeviceClient, MethodResponse


config = configparser.ConfigParser()
symmetric_key = None
registration_id = None
id_scope = None
capabilityModelId = None
capabilityModel = None 
provisioning_host = None

customerName = None
customerContact = None
storeName = None
storeLocation = None
deviceName = None


def lock_file(f):
    fcntl.flock(f.fileno(), fcntl.LOCK_EX)

def unlock_file(f):
    fcntl.flock(f.fileno(), fcntl.LOCK_UN)

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
    provisioning_host = config.get('DEVICE_SDK', 'provisioningHost')
    symmetric_key = config.get('DEVICE_SDK', 'sas_key')
    registration_id = config.get('DEVICE_SDK','deviceId')
    id_scope = config.get('DEVICE_SDK','scope')
    capabilityModelId=config.get('DEVICE_SDK','capabilityModelId')
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
    symmetric_key = config.get('DEVICE_SDK', 'sas_key')
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
    processorArchitecture= config.get('DEVICE_SDK','processorArchitecture')
    softwareVersion= config.get('DEVICE_SDK','softwareVersion')
    totalMemory= config.getint('DEVICE_SDK','totalMemory')
    totalStorage= config.getint('DEVICE_SDK','totalStorage')
    processorManufacturer= config.get('DEVICE_SDK','processorManufacturer')
    osName = config.get('DEVICE_SDK','osName')
    manufacturer= config.get('DEVICE_SDK','manufacturer')
    model= config.get('DEVICE_SDK','model')
    
    # update the device twin with store/customer information
    device_client.patch_twin_reported_properties({'storelocation':storeLocation})
    device_client.patch_twin_reported_properties({'customername':customerName})
    #device_client.patch_twin_reported_properties({'customercontact':customerContact})
    device_client.patch_twin_reported_properties({'storename':storeName})
    device_client.patch_twin_reported_properties({'devicename': deviceName})
    
    # update the device twin with device details
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
    sleep_time = config.getint("LOGGER", "PY_LOGGER_SLEEP")
    backlog_limit = config.getint("LOGGER", "BACKLOG_LIMIT")
    print("backlog_limit is ", backlog_limit)
    while True:
        try:  # keep on spinning this even in case of error
            with AtomicOpen(config.get("LOGGER", "LOG_FILE_PATH"), "r+") as fout:
                lines = [line.strip() for line in fout.readlines()]
                fout.seek(0)
                fout.write("")
                fout.truncate()
            
            if(len(lines) != 0):
                print(len(lines))
            
            messages = defaultdict(list)
            for x in lines:
                print("x =", x)
                if(len(x) != 0):
                    msg = json.loads(x)
                    _msg_type = list(msg.keys())[0]
                    messages[_msg_type].append(list(msg.values())[0])

            backlog = []
            for k, v in messages.items():
                if(k == "Liveness"):
                    v = v[-1:]  # for liveness keep the most recent ping
                
                for _msg in v:
                    msg_json = json.dumps({k : json.dumps(_msg)})
                    iot_message = Message(msg_json) 
                    print("iot_messsage =", iot_message)
                    
                    try:  # if error in sending, maintain backlog of all messages except liveness
                        iot_client.send_message(iot_message)
                    except:
                        if(k != "Liveness" and int(_msg["TimeStamp"]) >= int(time.time()) - backlog_limit):
                            backlog.append(json.dumps({k: _msg}))

            with AtomicOpen(config.get("LOGGER", "LOG_FILE_PATH"), "r+") as fout:
                for bl in backlog:
                    print(bl, file=fout)

        except Exception as ex:
            _msg = {"DeviceId": config.get("DEVICE_SDK", "deviceId"), 
                    "MessageSubType": "DeviceSDK", 
                    "MessageBody": {"Message": "exception in send_upstream_messages in deivce SDK: {}".format(ex)},
                    "TimeStamp": int(time.time())}
            message = Message(json.dumps({"Critical": json.dumps(_msg)}))
            print(message)
            iot_client.send_message(message)
        finally:
            time.sleep(sleep_time)

def getFileParams(param):
    fileparams = []
    if param is not None:
        files = [x for x in param.split(";") if len(x) > 0]
        for x in files:
            y = x.split(",")
            if(len(y) != 3):
                continue
            fileparams.append(commands_pb2.File(name=y[0], cdn=y[1], hashsum=y[2]))
    return fileparams

def command_listener(iot_client):
    print("Listening for device commands...")
    with grpc.insecure_channel('localhost:{}'.format(config.getint("GRPC", "DOWNSTREAM_PORT"))) as channel:
        stub = commands_pb2_grpc.RelayCommandStub(channel)
        while True:
            method_request = iot_client.receive_method_request()

            if method_request.name == "SetTelemetryInterval":  # this is there just for testing, not actually used
                try:
                    INTERVAL = int(method_request.payload)
                    print("Set Telemetry Interval : {}".format(INTERVAL))
                except ValueError:
                    response_payload = {"Response": "Invalid parameter"}
                    response_status = 400
                else:
                    response_payload = {"Response": "Executed direct method {}".format(method_request.name)}
                    response_status = 200
            elif(method_request.name == "Download"):
                print("Sending request to downlaoad", method_request.payload)
                payload = eval(method_request.payload)
                try:
                    _folder_path = payload['folder_path']
                    _metadata_files = getFileParams(payload['metadata_files'])
                    _bulk_files = getFileParams(payload['bulk_files'])
                    _channels = [commands_pb2.Channel(channelname=x) for x in payload['channels'].split(";")]
                    _deadline = int(payload['deadline'])
                    _add_to_existing = bool(payload["add_to_existing"])
                    print(dict(folderpath=_folder_path, metadatafiles=_metadata_files,
                    bulkfiles=_bulk_files, channels=_channels, deadline=_deadline,
                    addtoexisting=_add_to_existing))
                    
                    download_params = commands_pb2.DownloadParams(folderpath=_folder_path, metadatafiles=_metadata_files,
                    bulkfiles=_bulk_files, channels=_channels, deadline=_deadline)
                    response = stub.Download(download_params)
                    print(response)
                except Exception as ex:
                    print("Exception {}".format(ex))
                    response_payload = {"Response": "Error in sending from device SDK: {}".format(ex)}
                    response_status = 400
                else:
                    response_payload = {"Response": "Executed method call {}".format(method_request.name)}
                    response_status = 200
            elif(method_request.name == "Delete"):
                print("Sending request to delete", method_request.payload)
                payload = eval(method_request.payload)
                try:
                    _folder_path = payload['folder_path']
                    _recursive = bool(payload['recursive'])
                    _delete_after = int(payload['delete_after'])
                    delete_params = commands_pb2.DeleteParams(folderpath=_folder_path, recursive=_recursive,
                    deleteafter=_delete_after)
                    response = stub.Delete(delete_params)
                    print(response)
                except Exception as ex:
                    print("Exception {}".format(ex))
                    response_payload = {"Response": "Error in sending from device SDK: {}".format(ex)}
                    response_status = 400
                else:
                    response_payload = {"Response": "Executed method call {}".format(method_request.name)}
                    response_status = 200            
            elif(method_request.name == "AddNewPublicKey"):
                print("Sending request to add new public key", method_request.payload)
                payload = eval(method_request.payload)
                print(payload)
                try:
                    _public_key = payload["public_key"]
                    add_params = commands_pb2.AddNewPublicKeyParams(publickey=_public_key)
                    response = stub.AddNewPublicKey(add_params)
                    print(response)
                except Exception as ex:
                    response_payload = {"Response": "Error in sending from device SDK: {}".format(ex)}
                    response_status = 400
                else:
                    response_payload = {"Response": "Executed method call {}".format(method_request.name)}
                    response_status = 200
            else:
                response_payload = {"Response": "Method call {} not defined".format(method_request.name)}
                response_status = 404
            method_response = MethodResponse(method_request.request_id, response_status, payload=response_payload)
            iot_client.send_method_response(method_response)

if __name__ == '__main__':
    config.read('hub_config.ini')
    print(config.sections())

    # store details 
    config.read('customerdetails.ini')
    customerName = config.get('CUSTOMER_DETAILS', 'customer_name')
    #customerContact = config.get('CUSTOMER_DETAILS', 'customer_contact')
    storeName = config.get('CUSTOMER_DETAILS', 'store_name')
    storeLocation = config.get('CUSTOMER_DETAILS', 'store_location')
    deviceName = config.get('CUSTOMER_DETAILS', 'device_name')
    
    # capability Model
    capabilityModelId=config.get('DEVICE_SDK','capabilityModelId')
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
