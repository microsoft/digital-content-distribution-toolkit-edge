from concurrent import futures
import time
import sys

import grpc

import logger_pb2
import logger_pb2_grpc

from azure.iot.device import IoTHubDeviceClient, Message


CONNECTION_STRING = "HostName=gohub.azure-devices.net;DeviceId=MyPythonDevice;SharedAccessKey=zA2DAirXTqJ0TGkpf+8fTLYVCxC3YlPJsO+UUO2QS98="

def iothub_client_init():
    # Create an IoT Hub client
    client = IoTHubDeviceClient.create_from_connection_string(CONNECTION_STRING)
    return client


class LogServicer(logger_pb2_grpc.LogServicer):
    """Provides methods that implement functionality of logging server."""

    def __init__(self):
        self.client = iothub_client_init()
        pass

    def SendSingleLog(self, request, context):
        message = Message("[{}]{}".format(request.logtype, request.logstring))
        self.client.send_message(message)

        return logger_pb2.Empty()


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    logger_pb2_grpc.add_LogServicer_to_server(LogServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    time.sleep(1000)
    server.wait_for_termination()

if __name__ == '__main__':
    serve()