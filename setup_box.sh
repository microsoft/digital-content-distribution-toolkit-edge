echo "updating apt-get"
sudo apt-get update

echo "Installing tmux"
sudo apt install tmux

# golang
echo "installing grpc"
go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
go get -u google.golang.org/grpc
go get github.com/boltdb/bolt/...
go get -u github.com/gin-gonic/gin

# python
echo "Installing PIP"
sudo apt install python3-pip
echo "Installing python3 modules"
cd device_sdk
pip3 install --upgrade setuptools  # required for grpcio sometimes
pip3 install -r requirements.txt

cd ..
echo "=========================================="
echo "Done configuring development environment!!"
echo "Have fun hacking for BINE!!"
echo "=========================================="