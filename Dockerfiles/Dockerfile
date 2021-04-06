FROM arm64v8/ubuntu:latest

RUN apt-get update
RUN apt-get install wget -y

WORKDIR /bine/
COPY bine_arm ./
COPY *.ini ./
COPY start.sh ./
RUN chmod +x ./start.sh

EXPOSE 5000

CMD ["./start.sh"]
