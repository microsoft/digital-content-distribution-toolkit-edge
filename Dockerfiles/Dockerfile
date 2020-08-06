FROM binelogin.azurecr.io/hub_base_arm:latest
 
RUN apt-get update
RUN apt install nano -y
WORKDIR /usr/bine/
COPY device_sdk ./device_sdk
COPY hub_authentication ./hub_authentication

## Adding depnedencies
RUN pip3 install -r device_sdk/requirements.txt
RUN apt-get install tmux -y
RUN apt-get install psmisc -y
RUN rm -rf ~/.cache/

## Environment variables
ENV AZURE_CLIENT_ID="adb2c_client_id"
ENV AZURE_CLIENT_SECRET="adb2c_client_secret"
ENV AZURE_TENANT_ID="adb2c_tenant_id"
ENV KEY_VAULT_NAME="key_vault_name"

COPY bine_arm ./
COPY *.ini ./
COPY alive.sh ./
COPY start_hub.sh ./
EXPOSE 5000

CMD ["./alive.sh"]
