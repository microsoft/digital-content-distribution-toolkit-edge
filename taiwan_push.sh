mkdir temp
cp -r ./test/ ./temp/
cp ./*.go ./temp/
cp ./setup_box.sh ./temp/
scp -P 8222 -r ./temp/ root@noovo-rd.loginto.me:/root/compile
rm -rf temp