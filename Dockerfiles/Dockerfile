FROM binelogin.azurecr.io/hub_base_arm:latest
 
RUN apt-get update
RUN apt install nano -y
WORKDIR /usr/bine/
COPY device_sdk ./device_sdk
# Any new pip3 requirements can be added here
RUN pip3 install -r device_sdk/requirements.txt
COPY bine_arm ./
COPY *.ini ./
COPY alive.sh ./
COPY start_hub.sh ./
EXPOSE 5000

CMD ["./alive.sh"]
