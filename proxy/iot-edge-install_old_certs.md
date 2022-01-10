
# Generate root and device certificates

#### Clone the code to download the library to generate certificates on your local machine
<code> git clone https://github.com/Azure/azure-iot-sdk-node.git </code><br>
<code> cd azure-iot-sdk-node/provisioning/tools </code> <br>
<code> npm install </code> 

#### Generate Certificates
<code> node create_test_cert.js root blendnettestrootcert </code><br>
<code> node create_test_cert.js device <device_id> blendnettestrootcert </code>

#### Upload Certificate to IOT Central for Enrollment Group
<code>node create_test_cert.js verification --ca blendnettestrootcert_cert.pem --key blendnettestrootcert_key.pem --nonce  ACDC074456AF3746B5F3C7852E44131509501C2DFDFFBB31 </code>

#### Scope ID for IOT-C
> 0ne001E1BFE

#### Upload the device certificates to a storage container and then copy those to the device

<code>  cd /home </code><br>
<code>  mkdir iotedgecerts </code><br>
<code>  cd iotedgecerts </code><br>
<code>  curl -L <blob_url_cert.pem> -o <device_id>_cert.pem </code><br>
<code>  curl -L <blob_url_key.pem> -o <device_id>_key.pem </code><br>
<code>  curl -L <blob_url_fullchain.pem> -o <device_id>_fullchain.pem </code><br>
<code>  cd .. </code><br>
<code>  cd .. </code><br>

# Import Repository
<code> curl https://packages.microsoft.com/config/debian/10/prod.list > ./microsoft-prod.list </code><br>
<code> sudo cp ./microsoft-prod.list /etc/apt/sources.list.d/ </code>

# Install the Microsoft GPG public key
<code> curl https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > microsoft.gpg </code><br>
<code> sudo cp ./microsoft.gpg /etc/apt/trusted.gpg.d/ </code>

# Install a container engine
<code> sudo apt-get update </code><br>
<code> sudo apt-get install moby-engine </code><br>

# Install the IoT Edge security daemon
<code> curl -L https://github.com/Azure/azure-iotedge/releases/download/1.1.0/libiothsm-std_1.1.0-1-1_debian9_arm64.deb -o libiothsm-std.deb && sudo dpkg -i ./libiothsm-std.deb</code><br>
<code> curl -L https://github.com/Azure/azure-iotedge/releases/download/1.1.0/iotedge_1.1.0-1_debian9_arm64.deb -o iotedge.deb && sudo dpkg -i ./iotedge.deb

# Connect to IOT Central
1) Create Device associate with a the correct template (capability model)<br>
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
    identity_cert: "file:///home/iotedgecerts/<device_id>_fullchain.pem"
    identity_pk: "file:///home/iotedgecerts/<device_id>_key.pem"
  dynamic_reprovisioning: false <br>
</pre>

<code> sudo systemctl restart iotedge </code><br>
<code> sudo systemctl status iotedge </code><br>

### Change access rights for the directory
<code> sudo chown 1000 /etc/iotedge/storageonhost </code><br>

# Verify the configuration and connection for iotedge
<code> journalctl -u iotedge </code><br>
<code> sudo iotedge check --verbose </code><br>
<code> sudo iotedge list </code><br>

# References

*IoT Edge runtime installation*
> https://docs.microsoft.com/en-us/azure/iot-edge/how-to-install-iot-edge?view=iotedge-2018-06#option-1-authenticate-with-symmetric-keys
> https://docs.microsoft.com/en-us/azure/iot-central/core/tutorial-add-edge-as-leaf-device

*Certificate based provisioning*
> https://docs.microsoft.com/en-us/azure/iot-edge/how-to-auto-provision-x509-certs?view=iotedge-2018-06#linux-device













