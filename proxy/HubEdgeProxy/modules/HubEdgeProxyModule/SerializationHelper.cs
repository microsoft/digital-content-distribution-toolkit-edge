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
using Newtonsoft.Json.Serialization;

namespace HubEdgeProxyModule
{
    public static class SerializationHelper
    {

        /// <summary>
        /// Returns the serializer settings
        /// </summary>
        /// <returns></returns>
        public static JsonSerializerSettings GetJsonSerializerSettings()
        {
            DefaultContractResolver contractResolver = new DefaultContractResolver
            {
                NamingStrategy = new CamelCaseNamingStrategy()
            };

            return new JsonSerializerSettings
            {
                ContractResolver = contractResolver            
            };
        }
    }
}
