FROM binelogin.azurecr.io/hub_base_arm:latest
 
RUN apt-get update
RUN apt install nano -y
WORKDIR /usr/bine/
COPY device_sdk ./device_sdk
# Any new pip3 requirements can be added here
# CMD pip3 install example
COPY bine_arm ./
COPY *.ini ./
COPY alive.sh ./
EXPOSE 5000

CMD ["./alive.sh"]
