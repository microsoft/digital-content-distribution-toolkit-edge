import asyncio
import os
import json
import datetime
import random
import configparser

from azure.iot.device.aio import ProvisioningDeviceClient
from azure.iot.device.aio import IoTHubDeviceClient
from azure.iot.device import MethodResponse
from azure.iot.device import Message


device_client=None
config = configparser.ConfigParser()



async def provision():
  # device and host configuration are stored in the config file
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

  async def blink_command(request):
    print('Received synchronous call to blink')
    response = MethodResponse.create_from_method_request(
      request, status = 200, payload = {'description': f'Blinking LED every {request.payload} seconds'}
    )
    await device_client.send_method_response(response)  # send response
    print(f'Blinking LED every {request.payload} seconds')

  async def diagnostics_command(request):
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

  async def turnon_command(request):
    print('Turning on the LED')
    response = MethodResponse.create_from_method_request(
      request, status = 200
    )
    await device_client.send_method_response(response)  # send response

  async def turnoff_command(request):
    print('Turning off the LED')
    response = MethodResponse.create_from_method_request(
      request, status = 200
    )
    await device_client.send_method_response(response)  # send response

  commands = {
    'blink': blink_command,
    'rundiagnostics': diagnostics_command,
    'turnon': turnon_command,
    'turnoff': turnoff_command,
  }

  # Define behavior for handling commands
  async def command_listener():
    while True:
      method_request = await device_client.receive_method_request()  # Wait for commands
      await commands[method_request.name](method_request)    


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
  

  # Define behavior for halting the application
  def stdin_listener():
    while True:
      selection = input('Press Q to quit\n')
      if selection == 'Q' or selection == 'q':
        print('Quitting...')
        break
  
  device_client = await connect_device()
  await init_device()
  
  if device_client is not None and device_client.connected:
        print('Send reported properties on startup')
        await device_client.patch_twin_reported_properties({'state': 'true'})
        
        tasks = asyncio.gather(
        command_listener(),
        twin_patch_listener(),
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

