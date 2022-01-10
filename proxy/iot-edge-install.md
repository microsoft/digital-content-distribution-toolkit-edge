# Install Iot Edge

#### References
*IoT Edge runtime installation*
> https://docs.microsoft.com/en-us/azure/iot-edge/how-to-install-iot-edge?view=iotedge-2018-06#option-1-authenticate-with-symmetric-keys
> https://docs.microsoft.com/en-us/azure/iot-central/core/tutorial-add-edge-as-leaf-device

### Import Repository
<code> curl https://packages.microsoft.com/config/debian/10/prod.list > ./microsoft-prod.list </code><br>
<code> sudo cp ./microsoft-prod.list /etc/apt/sources.list.d/ </code>

### Install the Microsoft GPG public key
<code> curl https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > microsoft.gpg </code><br>
<code> sudo cp ./microsoft.gpg /etc/apt/trusted.gpg.d/ </code>

### Install a container engine
<code> sudo apt-get update </code><br>
<code> sudo apt-get install moby-engine </code><br>

### Install the IoT Edge security daemon
<code> curl -L https://github.com/Azure/azure-iotedge/releases/download/1.1.0/libiothsm-std_1.1.0-1-1_debian9_arm64.deb -o libiothsm-std.deb && sudo dpkg -i ./libiothsm-std.deb</code><br>
<code> curl -L https://github.com/Azure/azure-iotedge/releases/download/1.1.0/iotedge_1.1.0-1_debian9_arm64.deb -o iotedge.deb && sudo dpkg -i ./iotedge.deb

# Configuration to connect to Iot Central
#### Scope ID for IOT-Central
> 0ne001E1BFE

### Copt the device certificates from the storage accounts to the device storage
<code>  cd /home </code><br>
<code>  mkdir iotedgecertificates </code><br>
<code>  cd iotedgecertificates </code><br>
<code>  curl -L <blob-url-for-iot-edge-device-identity-hub-<device_id>.key.pem> -o i-<device_id>.key.pem </code><br>
<code>  curl -L <blob-url-for-iot-edge-device-identity-hub-<device_id>-full-chain.cert.pem> -o i-<device_id>-fullchain.cert.pem </code><br>
<code>  curl -L <blob-url-for-iot-edge-device-hub-<device_id>.key.pem> -o d-<device_id>.key.pem </code><br>
<code>  curl -L <blob-url-for-iot-edge-device-hub-<device_id>-full-chain.cert.pem> -o d-<device_id>-fullchain.cert.pem </code><br>
<code> curl -L <blob-url-for-azure-iot-test-only.root.ca.cert.pem> -o root.ca.cert.pem </code><br>

<code>  cd .. </code><br>
<code>  cd .. </code><br>


# Connect to IOT Central
1) Create Device associate with a the correct template (capability model) with the exact device ID used while creation of the certificates <br>
2) Update the configuration file 

<code>sudo nano /etc/iotedge/config.yaml<code><br>
#### Update the provisioning section for x.509 certificate using dps
<pre>
provisioning: 
  source: "dps"
  global_endpoint: "https://global.azure-devices-provisioning.net"
  scope_id: "0ne001E1BFE" 
  attestation: 
    method: "x509"
    registration_id: "<OPTIONAL REGISTRATION ID. LEAVE COMMENTED OUT TO REGISTER WITH CN OF identity_cert>"
    identity_cert: "file:///home/iotedgecertificates/i-<device_id>-fullchain.cert.pem"
    identity_pk: "file:///home/iotedgecertificates/i-<device_id>.key.pem"
  dynamic_reprovisioning: false <br>
</pre>
<pre>
certificates:
  device_ca_cert: "file:///home/iotedgecertificates/d-<device_id>-fullchain.cert.pem"
  device_ca_pk: "file:///home/iotedgecertificates/d-<device_id>.key.pem"
  trusted_ca_certs: "file:///home/iotedgecertificates/root.ca.cert.pem"
</pre>

####  Make Sure below directories are empty. Delete its content if required.
> /var/lib/iotedge/hsm/certs  
> /var/lib/iotedge/hsm/cert_keys


<code> sudo systemctl restart iotedge </code><br>
<code> sudo systemctl status iotedge </code><br>

### Change access rights for the directory
<code> sudo chown 1000 /etc/iotedge/storageonhost </code><br>

# Verify the configuration and connection for iotedge
<code> journalctl -u iotedge </code><br>
<code> sudo iotedge check --verbose </code><br>
The above command should not give any error , apart from 2 warnings for  configurations related to production logs and DNS server.<br>
<code> sudo iotedge list </code><br>
This command list outs the running iot-edge containers as below :
<pre>
NAME                STATUS           DESCRIPTION      CONFIG
HubEdgeProxyModule  running          Up 23 minutes    bineiot.azurecr.io/hubedgeproxymodule:0.0.1-arm64v8
HubModule           running          Up 23 minutes    bineiot.azurecr.io/hub_arm:latest
edgeAgent           running          Up 24 minutes    mcr.microsoft.com/azureiotedge-agent:1.0
edgeHub             running          Up 23 minutes    mcr.microsoft.com/azureiotedge-hub:1.0
</pre>














