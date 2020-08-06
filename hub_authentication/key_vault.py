import os
import cmd
from azure.keyvault.secrets import SecretClient
from azure.identity import DefaultAzureCredential

def retrieve_client_secret():
    keyVaultName = os.environ["KEY_VAULT_NAME"]
    KVUri = "https://{}.vault.azure.net".format(keyVaultName)
    
    credential = DefaultAzureCredential()
    client = SecretClient(vault_url=KVUri, credential=credential)
    client_secret =  client.get_secret("ClientSecret")
    return client_secret.value
    