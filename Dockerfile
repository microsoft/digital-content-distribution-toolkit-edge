FROM binelogin.azurecr.io/arm_hub_base:latest
 
WORKDIR /usr/bine/
COPY device_sdk ./device_sdk
# Any new pip3 requirements can be added here
# CMD pip3 install example
COPY bine_arm ./
COPY hub_config.ini ./
 
EXPOSE 5000

CMD ["/bin/bash"]
