using Newtonsoft.Json;
using Newtonsoft.Json.Serialization;

namespace HubEdgeProxyModule
{
    /// <summary>
    /// Represents the proxy consumer command
    /// </summary>
    public class ProxyModuleCommand
    {
        public string ModuleClientName {get;set;}

        public string CommandName {get;set;}
        
        public string Payload {get;set;}

        public double? ConnectionTimeOutInMts {get;set;}

    }

    /// <summary>
    /// Returns the command response
    /// </summary>
    public class ProxyModuleCommandResponse
    {
        public int Status {get;set;}
        
        public string Result {get;set;}

        public string GetDataAsJson()
        {
        
            return JsonConvert.SerializeObject(this, SerializationHelper.GetJsonSerializerSettings());
        }
    }
}