echo "updating apt-get"
sudo apt-get update
echo "installing zeromq"
sudo apt-get install libzmq3-dev
echo "installing zeromq C wrappers"
sudo apt-get install libczmq-dev
echo "installing pkg-dev"
sudo apt-get install pkg-config
echo "installing libsodium"
sudo apt-get install libsodium-dev
echo "Installing PIP"
sudo apt install python3-pip
echo "Installing Go modules"
go get github.com/boltdb/bolt/...
go get -u github.com/gin-gonic/gin
go get gopkg.in/zeromq/goczmq.v4
echo "Installing python3 modules"
cd ZMQ-PY
pip3 install -r requirements.txt
cd ..
echo "Done configuring development environment!!"