using Microsoft.Extensions.Logging;
using Microsoft.Azure.Devices.Client;
using Microsoft.Azure.Devices.Client.Transport.Mqtt;
using Microsoft.Azure.Devices.Shared;
using System.Threading.Tasks;
using System.Text;
using System.Collections.Generic;
using System.Collections;
using Newtonsoft.Json;
using System;
using Polly;
using Polly.Retry;
using Grpc.Core;
using Microsoft.Extensions.Options;

namespace HubEdgeProxyModule
{
    /// <summary>
    /// Module Client Helper
    /// Also contains the handler for commands from IOT Hub / Central
    /// </summary>
    public class ModuleClientHelper
    {
        private AppSettings _appSettings;

        private readonly ILogger<ModuleClientHelper> _logger;

        private readonly string _outputName = "output1";

        private readonly string  _moduleId;

        private readonly string _deviceId;

        private const double C_CONNECTION_TIME_OUT = 2.0; 
        
        /// <summary>
        /// GRPC Client Helper
        /// </summary>
        private GrpcCommandClientHelper _grpcCommandClientHelper;

        private ModuleClient _ioTHubModuleClient;

        /// <summary>
        /// Contains the details from Module Twins
        /// </summary>
        /// <value></value>
        public Dictionary<string,string> ProxyModuleClientEndpoints {get;set;}

        public ModuleClientHelper(  ILogger<ModuleClientHelper> logger, 
                                    GrpcCommandClientHelper grpcCommandClientHelper,
                                    IOptionsMonitor<AppSettings> options)
        {
            _logger = logger;

            _grpcCommandClientHelper = grpcCommandClientHelper;

             _moduleId = Environment.GetEnvironmentVariable("IOTEDGE_MODULEID");

             _deviceId = Environment.GetEnvironmentVariable("IOTEDGE_DEVICEID");

            _appSettings = options.CurrentValue;
        }

        /// <summary>
        /// Setup module client
        /// </summary>
        /// <returns></returns>
        public async Task SetupModuleClient()
        {
            _logger.LogInformation("Inside SetupModuleClient.");

            MqttTransportSettings mqttSetting = new MqttTransportSettings(TransportType.Mqtt_Tcp_Only);

            ITransportSettings[] settings = { mqttSetting };

            // Open a connection to the Edge runtime
            _ioTHubModuleClient = await ModuleClient.CreateFromEnvironmentAsync(settings);

            await _ioTHubModuleClient.OpenAsync();

            _ioTHubModuleClient.SetConnectionStatusChangesHandler(ConnectionStatusChangesHandler);
            
            _logger.LogInformation("IoT Hub module client initialized.");

            await _ioTHubModuleClient.SetMethodHandlerAsync("proxyModuleCommand",
                                                            ExecuteProxyModuleCommand,
                                                            "proxyModuleCommand");

            Microsoft.Azure.Devices.Shared.Twin moduleTwin = await _ioTHubModuleClient.GetTwinAsync();

            _logger.LogInformation($"Desired property count: {moduleTwin.Properties.Desired.Count}");

            ProxyModuleClientEndpoints = ((Newtonsoft.Json.Linq.JObject)moduleTwin.Properties.Desired["ProxyModuleClientEndpoints"]).ToObject<Dictionary<string,string>>();
        }

        /// <summary>
        /// Execute Proxy Consumer Command
        /// </summary>
        /// <param name="methodRequest"></param>
        /// <param name="userContext"></param>
        /// <returns></returns>
        async Task<MethodResponse> ExecuteProxyModuleCommand(     MethodRequest methodRequest, 
                                                                    object userContext)
        {
            TelemetryMessage telemetryMessage = null;

            MethodResponse methodResponse = null;

            try
            {
                string jsonData = methodRequest.DataAsJson;

                _logger.LogInformation($"ExecuteProxyModuleCommand: Command recieved with Data - { jsonData} and Context {userContext}");
                
                ProxyModuleCommand proxyModuleCommand =  JsonConvert.DeserializeObject<ProxyModuleCommand>(jsonData,SerializationHelper.GetJsonSerializerSettings());

                if (string.IsNullOrEmpty(proxyModuleCommand.ModuleClientName) || 
                    string.IsNullOrEmpty(proxyModuleCommand.CommandName) ||
                    string.IsNullOrEmpty(proxyModuleCommand.Payload))
                    {
                        string validationErrorMessage = "ExecuteProxyModuleCommand: Invalid command data recieved. Null or empty not allowed for ModuleClientName, CommandName , Payload";

                        _logger.LogError(validationErrorMessage);
                  
                        methodResponse =  GetMethodResponse(validationErrorMessage,0,System.Net.HttpStatusCode.BadRequest);

                        telemetryMessage = new TelemetryMessage () {Error = validationErrorMessage , DeviceIdInData = _deviceId};

                    }else
                    {
                        _logger.LogInformation($"ExecuteProxyModuleCommand: Invoking gRPC enpoint for - { proxyModuleCommand.ModuleClientName }");

                        //invoke gRPC
                        Command.CommandClient grpcCommandClient = _grpcCommandClientHelper.GetGrpcCommandClient(ProxyModuleClientEndpoints, proxyModuleCommand.ModuleClientName);

                        if (grpcCommandClient is null)
                        {
                            string validationErrorMessage = $"ExecuteProxyModuleCommand: Invalid ModuleClientName recieved. No grpc client configured for the given module client name : {proxyModuleCommand.ModuleClientName}";

                            _logger.LogError(validationErrorMessage);
                  
                            methodResponse =  GetMethodResponse(validationErrorMessage,0,System.Net.HttpStatusCode.BadRequest);

                            telemetryMessage = new TelemetryMessage () {Error = validationErrorMessage, DeviceIdInData = _deviceId };

                        }else
                        {
                            CommandServiceRequest request = new CommandServiceRequest() 
                            { 
                                CommandName = proxyModuleCommand.CommandName,
                                Payload = proxyModuleCommand.Payload
                            };

                            AsyncRetryPolicy policyBuilder = GetRetryPolicy<RpcException>();

                            await policyBuilder.ExecuteAsync(async () => {

                                if (   !(proxyModuleCommand.ConnectionTimeOutInMts.HasValue)  ||
                                        proxyModuleCommand.ConnectionTimeOutInMts.Value <= 0)
                                {
                                    proxyModuleCommand.ConnectionTimeOutInMts = C_CONNECTION_TIME_OUT;
                                }

                                CallOptions callOptions = new CallOptions(deadline:DateTime.UtcNow.AddMinutes(proxyModuleCommand.ConnectionTimeOutInMts.Value));                                

                                CommandServiceResponse commandServiceResponse = await grpcCommandClient.ReceiveCommandAsync(request,callOptions);

                                string successMessage = $"ExecuteProxyModuleCommand: Executed direct method: User Context : {userContext} : code {commandServiceResponse.Code} : message : {commandServiceResponse.Message} ";

                                _logger.LogInformation(successMessage);

                                methodResponse =  GetMethodResponse(successMessage,commandServiceResponse.Code,System.Net.HttpStatusCode.OK);

                                telemetryMessage = new TelemetryMessage () {Info = successMessage , DeviceIdInData = _deviceId };

                            });

                        }
                    }
            }catch(Exception exception)
            {
                string exceptionMessage = $"ExecuteProxyConsumerCommand: Exception - { exception.ToString()}";

                //log the exception
                _logger.LogError(exceptionMessage);

                //send the telemetry message with error
                telemetryMessage = new TelemetryMessage () {Error = exceptionMessage , DeviceIdInData = _deviceId};

                methodResponse = GetMethodResponse(exceptionMessage,0,System.Net.HttpStatusCode.InternalServerError); 
            }

            await SendMessage(telemetryMessage.GetDataAsJson());

            return methodResponse;
        }

        /// <summary>
        /// Get Method Response
        /// </summary>
        /// <param name="message"></param>
        /// <param name="status"></param>
        /// <param name="httpStatusCode"></param>
        /// <returns></returns>
        private MethodResponse GetMethodResponse(string message, int status, System.Net.HttpStatusCode httpStatusCode)
        {
            ProxyModuleCommandResponse response = new ProxyModuleCommandResponse() { Result = message , Status = status};
                        
            return new MethodResponse(Encoding.UTF8.GetBytes(response.GetDataAsJson()), (int)httpStatusCode);
        }

        ///Logs the connection status
        ///ToDo : Check if initialization of  module client is required.
        private void ConnectionStatusChangesHandler( ConnectionStatus status, 
                                                    ConnectionStatusChangeReason reason)
        {
            string message = $"ConnectionStatusChangesHandler - Status - {status} - Reasons - {reason}";
            
            _logger.LogInformation(message);
        }

        /// <summary>
        /// Send Message to IOT Edge
        /// </summary>
        /// <param name="telemeteryMessage"></param>
        /// <returns></returns>
        public async Task SendMessage(string telemeteryMessage)
        {
            _logger.LogInformation($"SendMessage: Inside send message of ModuleClientHelper. - {telemeteryMessage}");

            await _ioTHubModuleClient.SendEventAsync(_outputName, GetMessage(telemeteryMessage));

            _logger.LogInformation("SendMessage : Message Sent to IOT hub.");
        }

        /// <summary>
        /// Send Batch of Messages to IOT Edge Hub
        /// </summary>
        /// <param name="telemeteryMessage"></param>
        /// <returns></returns>
        public async Task SendMessageBatch(string[] telemeteryMessages)
        {
            _logger.LogInformation($"SendMessageBatch: Inside send message batch of ModuleClientHelper. Count : {telemeteryMessages.Length}");

            List<Message> messagesToSend = new List<Message>();

            foreach(string telemeteryMessage in  telemeteryMessages)
            {
                messagesToSend.Add(GetMessage(telemeteryMessage));
            }

            await _ioTHubModuleClient.SendEventBatchAsync(_outputName, messagesToSend);

            _logger.LogInformation("SendMessageBatch : Message Sent to IOT Edge hub.");
        }

        /// <summary>
        /// Returns the device client message
        /// </summary>
        /// <param name="telemeteryMessage"></param>
        /// <returns></returns>
        private Message GetMessage(string telemeteryMessage)
        {
            byte[] messageBytes =  System.Text.Encoding.UTF8.GetBytes(telemeteryMessage);

             Message message = new Message(messageBytes) 
             {
                    ContentType = "application/json",
                    ContentEncoding = "utf-8"
             };

             return message;
        }

        /// <summary>
        /// Polly Retry Policy for RCP Call
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <returns></returns>
        private AsyncRetryPolicy GetRetryPolicy<T>() where T : System.Exception
        {
              var builder = Policy
              .Handle<T>()
              .WaitAndRetryAsync(
                _appSettings.RetryCount,
                retryAttempt => TimeSpan.FromSeconds(Math.Pow(_appSettings.ExponentialBase, retryAttempt)),
                onRetry: (exception, delay, retryCount, context) => 
                {
                    _logger.LogError(exception, $"Attempt : {retryCount} - Exception Message : {exception.Message} - ");
                }
              );

              return builder;
        }

   }
}