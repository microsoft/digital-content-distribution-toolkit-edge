WHERE go
IF %ERRORLEVEL% NEQ 0 echo "Please install go from https://golang.org/dl/" && exit

echo "Installing go-gin"
go get -u github.com/gin-gonic/gin
echo "Installing boltdb"
go get github.com/boltdb/bolt/...
echo "Downloading testing content from blob"
curl "https://blendnetapp.blob.core.windows.net/installer/static.zip?sp=r&st=2020-03-29T19:19:34Z&se=2020-07-01T03:19:34Z&spr=https&sv=2019-02-02&sr=b&sig=vyq%%2Fg3E9vfQUmmWEGUfKfqlFxMed%%2BkOq%%2FkGP4DN2Px4%%3D" --output static.zip
echo "Please extract the static.zip file in the same directory"
echo "Folder structure needs to be ./static/MSR/**"