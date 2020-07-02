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

import logger_pb2
import logger_pb2_grpc
import commands_pb2
import commands_pb2_grpc

from azure.iot.device import IoTHubDeviceClient, Message
from azure.iot.device import IoTHubDeviceClient, Message, MethodResponse


config = configparser.ConfigParser()
# CONNECTION_STRING = "HostName=gohub.azure-devices.net;DeviceId=MyPythonDevice;SharedAccessKey=zA2DAirXTqJ0TGkpf+8fTLYVCxC3YlPJsO+UUO2QS98="


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

def iothub_client_init():
    # Create an IoT Hub client
    client = IoTHubDeviceClient.create_from_connection_string(config.get("DEVICE_INFO", "IOT_DEVICE_CONNECTION_STRING"))
    return client

# class LogServicer(logger_pb2_grpc.LogServicer):
#     """Provides methods that implement functionality of logging server."""
#     def __init__(self, iot_client):
#         self.iot_client = iot_client
#     def SendSingleLog(self, request, context):
#         message = Message("[{}][{}]{}".format(config.get("DEVICE_INFO", "DEVICE_NAME"), request.logtype, request.logstring))
#         print(message)
#         self.iot_client.send_message(message)
#         return logger_pb2.Empty()

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
                    print("here........")
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

def listen_for_method_calls(iot_client):
    print("Listening for method calls...")
    with grpc.insecure_channel('localhost:{}'.format(config.getint("GRPC", "DOWNSTREAM_PORT"))) as channel:
        stub = commands_pb2_grpc.RelayCommandStub(channel)
        while True:
            print("hello")
            # time.sleep(2)
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
                    _folder_path = payload["folder_path"]
                    # print(_folder_path)
                    _metadata_files = getFileParams(payload["metadata_files"]);
                    _bulk_files = getFileParams(payload["bulk_files"]);
                    # print(_metadata_files)
                    _channels = [commands_pb2.Channel(channelname=x) for x in payload["channels"].split(";")]
                    _deadline = int(payload["deadline"])
                    _add_to_existing = bool(payload["add_to_existing"])
                    print(dict(folderpath=_folder_path, metadatafiles=_metadata_files,
                    bulkfiles=_bulk_files, channels=_channels, deadline=_deadline, 
                    addtoexisting=_add_to_existing))
                    
                    download_params = commands_pb2.DownloadParams(folderpath=_folder_path, metadatafiles=_metadata_files,
                    bulkfiles=_bulk_files, channels=_channels, deadline=_deadline, addtoexisting=_add_to_existing)
                    response = stub.Download(download_params)
                    print(response)
                except Exception as ex:
                    response_payload = {"Response": "Error in sending from device SDK: {}".format(ex)}
                    response_status = 400
                else:
                    response_payload = {"Response": "Executed method  call {}".format(method_request.name)}
                    response_status = 200
            
            elif(method_request.name == "Delete"):
                print("Sending request to delete", method_request.payload)
                payload = eval(method_request.payload)
                try:
                    _folder_path = payload["folder_path"]
                    _recursive = bool(payload["recursive"])
                    _delete_after = int(payload["delete_after"])
                    delete_params = commands_pb2.DeleteParams(folderpath=_folder_path, recursive=_recursive,
                    delteafter=_delete_after)
                    response = stub.Delete(delete_params)
                    print(response)
                except Exception as ex:
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
    iot_client = iothub_client_init()
    
    # telemetry_pool = futures.ThreadPoolExecutor(1)
    # telemetry_pool.submit(listen_for_method_calls, iot_client)
    t = threading.Thread(target=listen_for_method_calls, args=[iot_client])
    t.daemon = True
    t.start()
    
    print("came here...")
    send_upstream_messages(iot_client)