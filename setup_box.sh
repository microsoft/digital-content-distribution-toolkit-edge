echo "updating apt-get"
apt-get update
echo "installing zeromq"
apt-get install libzmq3-dev
echo "installing zeromq C wrappers"
apt-get install libczmq-dev
echo "installing pkg-dev"
apt-get install pkg-config
echo "installing libsodium"
apt-get install libsodium-dev
echo "Installing go-dependencies"
go get
echo "Installing PIP"
apt install python3-pip