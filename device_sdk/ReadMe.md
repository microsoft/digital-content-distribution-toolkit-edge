#### Create SAS key using below command for a device.
   ```
   dps-keygen -mk:<group primary key> -di:<device ID>
   ```
   #### Example
   ```   
   dps-keygen -mk:4+IZ0emfOPVqs5Fq+v95Bke8WaGlqbh33UVzrLVu5dDlparcUXsfpMGofZhDbapJZlMqdOTiX3iB/W16JB2hiw== -di:blendnet-device-1
   ```

#### Use the key generated and the device ID used in earlier step in device.ini file
   ```  
   sas_key = <key>
   deviceId = <deviceID> 
   ```  
   #### Example
   ```  
   sas_key = i7Y5xzYuoA28gDryw/s6Z4C92vwp3OHKavuvimFEp9U=
   deviceId = blendnet-device-1 
   ```  

#### set the environment variables to access the Azure key-vault
   ``` 
   export AZURE_CLIENT_ID="22f54f66-df06-42c1-b689-f089aedf9d26"
   export AZURE_CLIENT_SECRET="6jC-D43p3om4_xlUHRX56R27~HVODd1BY"
   export AZURE_TENANT_ID="72f988bf-86f1-41af-91ab-2d7cd011db47"
   export KEY_VAULT_NAME="mishtu"
   ``` 
#### create self sign certificate for the bine portal application
   ```
   openssl req -newkey rsa:4096 -x509 -sha256 -days 3650 -nodes -out mishtu.crt -keyout mishtu.key
   ```