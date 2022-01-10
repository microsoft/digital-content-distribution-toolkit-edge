  
# Generate root and device certificates
## Reference
*Certificate based provisioning*
> https://docs.microsoft.com/en-us/azure/iot-edge/how-to-create-test-certificates?view=iotedge-2018-06#prepare-scripts-in-powershell

#### Install OpenSSL
#### Clone the code to download the library to generate certificates on your local machine
1. Open a PowerShell window in administrator mode <br>
2. Clone the IoT Edge git repo, which contains scripts to generate demo certificates. Use the git clone command or download the ZIP <br>
<code> git clone https://github.com/Azure/iotedge.git </code><br>
3. Open the <path>\iotedge\tools\CACertificates\ca-certs.ps1 file and replace line 32 <br>
<code> if (-not (Test-Path env:DEFAULT_VALIDITY_DAYS)) { $env:DEFAULT_VALIDITY_DAYS = 30 } </code> <br>
by <br>
<code> $env:DEFAULT_VALIDITY_DAYS = 365 </code> <br>
4. Navigate to the directory in which you want to work. Throughout this article, we'll call this directory <WRKDIR>. All certificates and keys will be created in this working directory <br> 
5. Copy the configuration and script files from the cloned repo into your working directory. If you downloaded the repo as a ZIP, then the folder name is iotedge-master and the rest of the path is the same <br>
<code> copy <path>\iotedge\tools\CACertificates\*.cnf . </code> <br>
<code> copy <path>\iotedge\tools\CACertificates\ca-certs.ps1 . </code> <br>
6. Enable PowerShell to run the scripts. <br>
<code> Set-ExecutionPolicy -ExecutionPolicy Unrestricted -Scope CurrentUser </code><br>
7. Bring the functions used by the scripts into PowerShell's global namespace. <br>
<code> . .\ca-certs.ps1 </code> <br>
8. Verify that OpenSSL has been installed correctly and make sure that there won't be name collisions with existing certificates. If there are problems, the script output should describe how to fix them on your system.<br>
<code> Test-CACertsPrerequisites </code> <br>

#### Create root CA certificate
1. Navigate to the working directory where you placed the certificate generation scripts. <br>
2. Create the root CA certificate and have it sign one intermediate certificate. The certificates are all placed in your working directory. <br>
<code> New-CACertsCertChain rsa </code><br>
This script command creates several certificate and key files, but when articles ask for the root CA certificate, use the following file:<br>
<code> <WRKDIR>\certs\azure-iot-test-only.root.ca.cert.pem </code><br>

#### Create IoT Edge device identity certificates
1. Create the IoT Edge device identity certificate and private key with the following command: <br>
<code> New-CACertsEdgeDeviceIdentity "<name>" </code>
The name that you pass in to this command will be the device ID for the IoT Edge device in IoT Hub. <br>
The new device identity command creates several certificate and key files, including three that you'll use when creating an individual enrollment in DPS and installing the IoT Edge runtime: <br>
  * <WRKDIR>\certs\iot-edge-device-identity-<name>-full-chain.cert.pem
  * <WRKDIR>\certs\iot-edge-device-identity-<name>.cert.pem
  * <WRKDIR>\private\iot-edge-device-identity-<name>.key.pem

#### Create IoT Edge device CA certificates
1. Navigate to the working directory that has the certificate generation scripts and root CA certificate. <br>
2. Create the IoT Edge device CA certificate and private key with the following command. Provide a name for the CA certificate.<br>
<code> New-CACertsEdgeDevice "<CA cert name>" </code> <br>
This command creates several certificate and key files. The following certificate and key pair needs to be copied over to an IoT Edge device and referenced in the config.yaml file: <br>
  * <WRKDIR>\certs\iot-edge-device-<CA cert name>-full-chain.cert.pem
  * <WRKDIR>\private\iot-edge-device-<CA cert name>.key.pem <br>
<br>
The name passed to the New-CACertsEdgeDevice command should not be the same as the hostname parameter in config.yaml, or the device's ID in IoT Hub.

#### Upload the Root certificate to IOT-C and verify it
1. Upload the root CA certificate file from your working directory, <WRKDIR>\certs\azure-iot-test-only.root.ca.cert.pem, to your IoT hub. <br>
2. Use the code provided in the Azure portal to verify that you own that root CA certificate. <br>
<code> New-CACertsVerificationCert "<verification code>" </code> <br>

#### Copy all the device identity and CA certificates along with the trusted CA certificate to azure storage container.
> https://binestorageaccount.blob.core.windows.net/iotedge-certs <br>
