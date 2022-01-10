namespace HubEdgeProxyModule
{
    /// <summary>
    /// App Settings
    /// </summary>
    public class AppSettings
    {
        /// <summary>
        /// Number of Retry to perform
        /// </summary>
        /// <value></value>
        public int RetryCount { get;set;} 

        /// <summary>
        /// Exponential Factor for Retry
        /// </summary>
        /// <value></value>
        public int ExponentialBase {get;set;}
    }
}