from __future__ import print_function

from concurrent import futures
from multiprocessing import Process
import time
import sys
import grpc

import logger_pb2
import logger_pb2_grpc
import commands_pb2
import commands_pb2_grpc

from azure.iot.device import IoTHubDeviceClient, Message
from azure.iot.device import IoTHubDeviceClient, Message, MethodResponse



CONNECTION_STRING = "HostName=gohub.azure-devices.net;DeviceId=MyPythonDevice;SharedAccessKey=zA2DAirXTqJ0TGkpf+8fTLYVCxC3YlPJsO+UUO2QS98="

def iothub_client_init():
    # Create an IoT Hub client
    client = IoTHubDeviceClient.create_from_connection_string(CONNECTION_STRING)
    return client


class LogServicer(logger_pb2_grpc.LogServicer):
    """Provides methods that implement functionality of logging server."""

    def __init__(self, iot_client):
        self.iot_client = iot_client

    def SendSingleLog(self, request, context):
        message = Message("[{}]{}".format(request.logtype, request.logstring))
        print(message)
        self.iot_client.send_message(message)

        return logger_pb2.Empty()


def send_upstream_messages(iot_client):
    print("Starting telemetry...")
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=5))
    logger_pb2_grpc.add_LogServicer_to_server(LogServicer(iot_client), server)
    server.add_insecure_port('localhost:50051')
    server.start()
    print("server started")
    time.sleep(1000)
    server.wait_for_termination()

def listen_for_method_calls(iot_client):
    print("Listening for method calls...")
    with grpc.insecure_channel('localhost:50052') as channel:
        stub = commands_pb2_grpc.RelayCommandStub(channel)

        while True:
            print("hello")
            time.sleep(2)
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
                    _metadata_files = [commands_pb2.File(name="cover.jpg", cdn="cdn1", hashsum="asdfasdf"), commands_pb2.File(name="rating.txt", cdn="cdn1", hashsum="asdfasdf")];
                    _bulk_files = [commands_pb2.File(name="A song of ice and fire", cdn="cdn1", hashsum="asdfasdf")];
                    
                    _channels = [commands_pb2.Channel(channelname=x) for x in payload["channels"].split(";")]
                    _deadline = int(payload["deadline"])

                    print(dict(folderpath=_folder_path, metadatafiles=_metadata_files,
                    bulkfiles=_bulk_files, channels=_channels, deadline=_deadline
                    ))
                    
                    download_params = commands_pb2.DownloadParams(folderpath=_folder_path, metadatafiles=_metadata_files,
                    bulkfiles=_bulk_files, channels=_channels, deadline=_deadline)

                    response = stub.Download(download_params)
                    print(response)
                except:
                    response_payload = {"Response": "Invalid parameter"}
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
                except:
                    response_payload = {"Response": "Invalid parameter"}
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
    iot_client = iothub_client_init()
    telemetry_pool = futures.ThreadPoolExecutor(1)
    telemetry_pool.submit(send_upstream_messages, iot_client)
    # p = Process(target=send_upstream_messages, args=(iot_client,))
    # p.start()
    # send_upstream_messages(iot_client)
    # time.sleep(10)
    print("came here...")
    # telemetry_pool.submit(listen_for_method_calls, iot_client)
    # telemetry_pool = futures.ThreadPoolExecutor(1)
    # telemetry_pool.submit(listen_for_method_calls, iot_client)
    listen_for_method_calls(iot_client)