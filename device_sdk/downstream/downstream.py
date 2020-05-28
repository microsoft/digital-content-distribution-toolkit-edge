from __future__ import print_function

import random
import logging

import grpc

import commands_pb2
import commands_pb2_grpc

from azure.iot.device import IoTHubDeviceClient, Message, MethodResponse

CONNECTION_STRING = "HostName=gohub.azure-devices.net;DeviceId=MyPythonDevice;SharedAccessKey=zA2DAirXTqJ0TGkpf+8fTLYVCxC3YlPJsO+UUO2QS98="

# def run():
#     with grpc.insecure_channel('localhost:50052') as channel:
#         stub = commands_pb2_grpc.RelayCommandStub(channel)

#         channels = [commands_pb2.Channel(channelname="wifi")];
#         metadata_files = [commands_pb2.File(name="cover.jpb", cdn="cdn1", hashsum="asdfasdf"), commands_pb2.File(name="rating.txt", cdn="cdn1", hashsum="asdfasdf")];
#         bulk_files = [commands_pb2.File(name="A song of ice and fire", cdn="cdn1", hashsum="asdfasdf")];

#         download_params = commands_pb2.DownloadParams(id="id1", mediahouse="HBO", channels=channels, 
#             metadatafiles=metadata_files, bulkfiles=bulk_files, deadline=100, hierarchy="HBO/GOT/");


#         response = stub.Download(download_params)

#         print(response)


# run()
def iothub_client_init():
    client = IoTHubDeviceClient.create_from_connection_string(CONNECTION_STRING)
    return client

device_client = iothub_client_init()
with grpc.insecure_channel('localhost:50052') as channel:
    stub = commands_pb2_grpc.RelayCommandStub(channel)

    while True:
        method_request = device_client.receive_method_request()
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
        device_client.send_method_response(method_response)