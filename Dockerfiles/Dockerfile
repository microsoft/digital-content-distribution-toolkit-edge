FROM arm64v8/ubuntu:latest

RUN apt-get update
RUN apt-get install wget -y

WORKDIR /bine/
COPY bine_arm ./
COPY *.ini ./
<<<<<<< HEAD
COPY alive.sh ./
COPY start_hub.sh ./
COPY run_go_python_sides.sh ./
EXPOSE 5000, 5002
=======
COPY start.sh ./
EXPOSE 5000
>>>>>>> upstream/dev

CMD ["./start.sh"]
