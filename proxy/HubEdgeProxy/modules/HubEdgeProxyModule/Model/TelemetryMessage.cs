using System;
using Newtonsoft.Json;

namespace HubEdgeProxyModule
{
    public class TelemetryMessage
    {
        public string Error { get; set; }
        
        public string Critical { get; set; }

        public string Info { get; set; }

        public string ApplicationName { get; set; } = "HubEdgeProxyModule";

        /// <summary>
        /// https://github.com/Azure/iotedgehubdev/issues/175
        /// https://github.com/Azure/azure-iot-sdk-csharp/issues/515
        /// </summary>
        /// <value></value>
        public string DeviceIdInData { get; set; }

        public DateTime TimeStamp { get; set; } = DateTime.Now;

        public string GetDataAsJson()
        {
            return  JsonConvert.SerializeObject(this,SerializationHelper.GetJsonSerializerSettings());
        }

    }
}
