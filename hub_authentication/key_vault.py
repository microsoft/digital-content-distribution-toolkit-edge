import os
import cmd
from azure.keyvault.secrets import SecretClient
from azure.identity import DefaultAzureCredential

def retrieve_client_secret():
    os.environ["AZURE_CLIENT_ID"] = "22f54f66-df06-42c1-b689-f089aedf9d26"
    os.environ["AZURE_CLIENT_SECRET"] = "6jC-D43p3om4_xlUHRX56R2.7~HVODd1BY"
    os.environ["AZURE_TENANT_ID"] = "72f988bf-86f1-41af-91ab-2d7cd011db47"
    KVUri = "https://mishtu.vault.azure.net"
    
    credential = DefaultAzureCredential()
    client = SecretClient(vault_url=KVUri, credential=credential)
    client_secret =  client.get_secret("ClientSecret")
    return client_secret.value
    