FROM binelogin.azurecr.io/hub_base_arm:latest
 
WORKDIR /usr/bine/
RUN apt install nano
COPY device_sdk ./device_sdk
# Any new pip3 requirements can be added here
# CMD pip3 install example
COPY bine_arm ./
COPY hub_config.ini ./
COPY alive.sh ./
EXPOSE 5000

CMD ["./alive.sh"]
