using System.Threading;
using System.Threading.Tasks;
using Microsoft.Extensions.Logging;
using Grpc.Core;
using System;
using System.Linq;

namespace HubEdgeProxyModule
{
    /// <summary>
    /// Proxy service to send the telemetry 
    /// Telemetry service to send the data to Azure IOT HUB
    /// </summary>
    public class TelemetryService : Telemetry.TelemetryBase
    {
        private readonly ILogger<TelemetryService> _logger;

        private ModuleClientHelper _moduleClient;

        public TelemetryService(ILogger<TelemetryService> logger, 
                                ModuleClientHelper moduleClient)
        {
            _logger = logger;

            _moduleClient = moduleClient;
        }

        /// <summary>
        /// Sends the telemetry to the IOT Edge HUB
        /// </summary>
        /// <param name="request"></param>
        /// <param name="context"></param>
        /// <returns></returns>
        public override async Task<TelemetryResponse> SendTelemetry(  TelemetryRequest request, 
                                                                ServerCallContext context)
        {
            TelemetryResponse response = new TelemetryResponse() { Code = 0};

            try
            {
                if (request != null &&  !string.IsNullOrEmpty(request.TelemetryData)) {
                    
                    _logger.LogDebug($"SendTelemetry: Recieved the message to send {request.ApplicationName}. Data - {request.TelemetryData}");

                    await _moduleClient.SendMessage(request.TelemetryData);

                    response.Message = "SendTelemetry: Message sent sucessfully";

                    response.Code = 1;
                    
                }else {
                    
                    response.Message = "SendTelemetry: Recieved data is not in correct format.";
                }

            }catch(Exception ex)
            {
                _logger.LogError(ex,ex.Message);                

                response.Message = ex.Message;
            }

            return response;
        }

        /// <summary>
        /// Sends the Telemetry Batch Data.
        /// </summary>
        /// <param name="request"></param>
        /// <param name="context"></param>
        /// <returns></returns>
        public override async Task<TelemetryResponse> SendTelemetryBatch(TelemetryBatchRequest request, ServerCallContext context)
        {
            TelemetryResponse response = new TelemetryResponse() { Code = 0};

            try
            {
                if (request != null && request.TelemetryData != null && request.TelemetryData.Count > 0) {
                    
                    _logger.LogDebug($"SendTelemetryBatch: Recieved the message to send {request.ApplicationName}. Data Count - {request.TelemetryData.Count}");

                    await _moduleClient.SendMessageBatch(request.TelemetryData.ToArray<string>());

                    response.Message = $"SendTelemetryBatch: Message sent sucessfully. Message Count - {request.TelemetryData.Count}";

                    response.Code = 1;
                    
                }else {
                    
                    response.Message = "SendTelemetryBatch: Recieved data is not in correct format.";
                }

            }catch(Exception ex)
            {
                _logger.LogError(ex,ex.Message);                

                response.Message = ex.Message;
            }

            return response;
        }

    }
}
