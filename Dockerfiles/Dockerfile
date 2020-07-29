FROM binelogin.azurecr.io/hub_base_arm:latest
 
RUN apt-get update
RUN apt install nano -y
WORKDIR /usr/bine/
COPY device_sdk ./device_sdk

## Adding depnedencies
RUN apt-get install python3-pip -y
RUN pip3 install -r device_sdk/requirements.txt
RUN apt-get remove python3-pip -y

RUN rm -rf ~/.cache/

COPY bine_arm ./
COPY *.ini ./
COPY alive.sh ./
COPY start_hub.sh ./
EXPOSE 5000

CMD ["./alive.sh"]
