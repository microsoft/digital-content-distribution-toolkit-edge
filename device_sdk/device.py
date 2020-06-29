import asyncio
import os
import json
import datetime
import random
import configparser
from concurrent import futures
import time

from azure.iot.device.aio import ProvisioningDeviceClient
from azure.iot.device.aio import IoTHubDeviceClient
from azure.iot.device import MethodResponse
from azure.iot.device import Message

import grpc

import logger_pb2
import logger_pb2_grpc
import commands_pb2
import commands_pb2_grpc


device_client=None
config = configparser.ConfigParser()

class LogServicer(logger_pb2_grpc.LogServicer):
    """Provides methods that implement functionality of logging server."""

    def __init__(self, device_client):
      print(f'Assigning iot_client')
      self.iot_client = device_client

    def SendSingleLog(self, request, context):
        print(f'Preparing to send telemetry from the provisioned device')
        message = Message("[{}]{}".format(request.logtype, request.logstring))
        print(message)
        print(f'Sending telemetry from the provisioned device')
        self.iot_client.send_message(message)

        return logger_pb2.Empty()

async def provision():
  # device and host configuration are stored in the config file
  config.read('hub_config.ini')
  config.read('device.ini')
  provisioning_host = config.get('host', 'provisioningHost')
  symmetric_key = config.get('section_device', 'sas_key')
  registration_id = config.get('section_device','deviceId')
  id_scope = config.get('section_device','scope')
  
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
  # capability Model
  capabilityModelId=config.get('section_device','capabilityModelId')
  capabilityModel = "{iotcModelId : '" + capabilityModelId + "'}" 
  

  # All the remaining code is nested within this main function

  async def register_device():
    provisioning_device_client = ProvisioningDeviceClient.create_from_symmetric_key(
      provisioning_host=provisioning_host,
      registration_id=registration_id,
      id_scope=id_scope,
      symmetric_key=symmetric_key,
    )
    provisioning_device_client.provisioning_payload = capabilityModel
    registration_result = await provisioning_device_client.register()

    print(f'Registration result: {registration_result.status}')

    return registration_result

  async def init_device():
    await device_client.patch_twin_reported_properties({'location':location})
    await device_client.patch_twin_reported_properties({'name':customerName})
    await device_client.patch_twin_reported_properties({'model':model})
    await device_client.patch_twin_reported_properties({'swVersion':softwareVersion})
    await device_client.patch_twin_reported_properties({'osName':osName})
    await device_client.patch_twin_reported_properties({'processorManufacturer':processorManufacturer})
    await device_client.patch_twin_reported_properties({'totalMemory':totalMemory})
    await device_client.patch_twin_reported_properties({'totalStorage':totalStorage})
    await device_client.patch_twin_reported_properties({'manufacturer':manufacturer})
    await device_client.patch_twin_reported_properties({'processorArchitecture': processorArchitecture})


  async def connect_device():
    device_client = None
    try:
      registration_result = await register_device()
      if registration_result.status == 'assigned':
        device_client = IoTHubDeviceClient.create_from_symmetric_key(
          symmetric_key=symmetric_key,
          hostname=registration_result.registration_state.assigned_hub,
          device_id=registration_result.registration_state.device_id,
        )
        # Connect the client.
        await device_client.connect()
        print('Device connected successfully')
    finally:
      return device_client

  async def blink_command(request, stub):
    print('Received synchronous call to blink')
    response = MethodResponse.create_from_method_request(
      request, status = 200, payload = {'description': f'Blinking LED every {request.payload} seconds'}
    )
    await device_client.send_method_response(response)  # send response
    print(f'Blinking LED every {request.payload} seconds')

  async def diagnostics_command(request, stub):
    print('Starting asynchronous diagnostics run...')
    response = MethodResponse.create_from_method_request(
      request, status = 202
    )
    await device_client.send_method_response(response)  # send response
    print('Generating diagnostics...')
    await asyncio.sleep(2)
    print('Generating diagnostics...')
    await asyncio.sleep(2)
    print('Generating diagnostics...')
    await asyncio.sleep(2)
    print('Sending property update to confirm command completion')
    await device_client.patch_twin_reported_properties({'rundiagnostics': {'value': f'Diagnostics run complete at {datetime.datetime.today()}.'}})

  async def turnon_command(request, stub):
    print('Turning on the LED')
    response = MethodResponse.create_from_method_request(
      request, status = 200
    )
    await device_client.send_method_response(response)  # send response

  async def turnoff_command(request, stub):
    print('Turning off the LED')
    response = MethodResponse.create_from_method_request(
      request, status = 200
    )
    await device_client.send_method_response(response)  # send response
  
  async def setTelemetryInterval(request, stub):
    print('starting setTelemetryInterval')
    try:
        INTERVAL = int(request.payload)
    except ValueError:
        response_payload = {"Response": "Invalid parameter"}
        response_status = 400
    else:
        response_payload = {"Response": "Executed direct method {}".format(request.name)}
        response_status = 200
    method_response = MethodResponse(request.request_id, response_status, payload=response_payload)
    await device_client.send_method_response(method_response)
    
  async def delete(request, stub):
    print("Sending request to delete", request.payload)
    payload = eval(request.payload)

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
        response_payload = {"Response": "Executed method  call {}".format(request.name)}
        response_status = 200
    method_response = MethodResponse(request.request_id, response_status, payload=response_payload)
    await device_client.send_method_response(method_response)
    
  async def download(request, stub):
    print("Sending request to download")
    payload = eval(request.payload)
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
        response_payload = {"Response": "Executed method  call {}".format(request.name)}
        response_status = 200
    
    method_response = MethodResponse(request.request_id, response_status, payload=response_payload)
    await device_client.send_method_response(method_response)

  commands = {
    'blink': blink_command,
    'rundiagnostics': diagnostics_command,
    'turnon': turnon_command,
    'turnoff': turnoff_command,
    'setTelemetryInterval': setTelemetryInterval,
    'download' : download,
    'delete' : delete
  }
  # Define behavior for handling commands
  async def command_listener():
    with grpc.insecure_channel('localhost:{}'.format(config.getint("GRPC", "DOWNSTREAM_PORT"))) as channel:
        stub = commands_pb2_grpc.RelayCommandStub(channel)
    while True:
      print("command_listener started...")
      method_request = await device_client.receive_method_request()  # Wait for commands
      await commands[method_request.name](method_request, stub)    


  async def name_setting(value, version):
      await asyncio.sleep(1)
      print(f'Setting name value {value} - {version}')
      await device_client.patch_twin_reported_properties({'name' : {'value': value['value'], 'status': 'completed', 'desiredVersion': version}})

  async def location_setting(value, version):
      await asyncio.sleep(5)
      print(f'Setting location value {value} - {version}')
      await device_client.patch_twin_reported_properties({'location' : {'value': value['value'], 'status': 'completed', 'desiredVersion': version}})
      
  settings = {
      'name': name_setting,
      'location': location_setting
  }

    # define behavior for receiving a twin patch
  async def twin_patch_listener():
      while True:
        patch = await device_client.receive_twin_desired_properties_patch() # blocking
        to_update = patch.keys() & settings.keys()
        await asyncio.gather(
          *[settings[setting](patch[setting], patch['$version']) for setting in to_update]
        )
  
  def send_upstream_messages():
    print("Starting telemetry...")
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=5))
    logger_pb2_grpc.add_LogServicer_to_server(LogServicer(device_client), server)
    server.add_insecure_port('localhost:{}'.format(config.getint("GRPC", "UPSTREAM_PORT")))
    server.start()
    print("server started")
    time.sleep(1000)
    server.wait_for_termination()
  
    # Define behavior for halting the application
  def stdin_listener():
    while True:
      selection = input('Press Q to quit\n')
      if selection == 'Q' or selection == 'q':
        print('Quitting...')
        break
            
  # connect to the IOT device 
  device_client = await connect_device()
  # start sending telemetry 
  telemetry_pool = futures.ThreadPoolExecutor(1)
  telemetry_pool.submit(send_upstream_messages)
  # initialize the device by setting all the properties
  await init_device()
  
  if device_client is not None and device_client.connected:
        print('Send reported properties on startup')
        await device_client.patch_twin_reported_properties({'state': 'true'})
        tasks = asyncio.gather(
        command_listener(), # listens to incoming commands from IOT hub
        twin_patch_listener() # updates editable properties for a device in the IOT hub
        )

        # Run the stdin listener in the event loop
        loop = asyncio.get_running_loop()
        user_finished = loop.run_in_executor(None, stdin_listener)

        # Wait for user to indicate they are done listening for method calls
        await user_finished
        
        # Cancel tasks
        tasks.add_done_callback(lambda r: r.exception())
        tasks.cancel()
        
        await device_client.disconnect()

  else:
        print('Device could not connect')
        
  return device_client

if __name__ == '__main__':
  asyncio.run(provision())