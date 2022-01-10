using System;
using System.Collections;
using System.Collections.Generic;
using Grpc.Net.Client;


namespace HubEdgeProxyModule
{
    public class GrpcCommandClientHelper
    {
        private Dictionary<string, GrpcChannel> ChannelList {get;set;} = new Dictionary<string, GrpcChannel>();

        /// <summary>
        /// Returns the command client from cache.
        /// Since there can be no connectivity for a longer period of time, avoiding the usage of cache for now
        /// </summary>
        /// <param name="applicationName"></param>
        /// <returns></returns>
        public Command.CommandClient GetGrpcCommandClientFromCache(  Dictionary<string,string> channelDetails,
                                                            string applicationName)
        {
            GrpcChannel channel  = null;

            if (ChannelList.ContainsKey(applicationName))
            {
                channel =  ChannelList[applicationName];

            }else
            {
                if (channelDetails.ContainsKey(applicationName))
                {
                    AppContext.SetSwitch("System.Net.Http.SocketsHttpHandler.Http2UnencryptedSupport", true);
                    
                    channel = GrpcChannel.ForAddress(channelDetails[applicationName]);

                    ChannelList.Add(applicationName,channel);
                }
            }

            if (channel != null)
            {
                return new Command.CommandClient(channel);
            }else
            {
                return null;    
            }
           
        }

        /// <summary>
        /// Returns the new Command Client always
        /// </summary>
        /// <param name="channelDetails"></param>
        /// <param name="applicationName"></param>
        /// <returns></returns>
        public Command.CommandClient GetGrpcCommandClient(  Dictionary<string,string> channelDetails,
                                                            string applicationName)
        {
            GrpcChannel channel  = null;

            if (channelDetails.ContainsKey(applicationName))
            {
                AppContext.SetSwitch("System.Net.Http.SocketsHttpHandler.Http2UnencryptedSupport", true);
                    
                channel = GrpcChannel.ForAddress(channelDetails[applicationName]);

                return new Command.CommandClient(channel);
            }

            return null;    
        }

    }

}