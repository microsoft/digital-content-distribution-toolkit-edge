#!/bin/bash
# check if log file size exceeded limit
size=$(stat -c %s $1)
if [[ $size -gt 1024*1024*10 ]]
then
	echo "" > $1
fi

# starting tmux if not running
if [ "$(tmux ls | grep -c "bine-session:")" -eq 0 ]; 
then
	/usr/bine/start_hub.sh
fi

# check if goserver is running; if not, start it in the tmux session
if [ -f "/media/sda1/MSR/retailer_detail.ini" ] && [ `pgrep -f ./bine_arm | wc -l` -eq 0 ];
then
    echo "goserver is not running. Starting it now..."
    tmux send -t bine-session:0.0 './bine_arm' ENTER
else
    echo "hub not authenticated/goserver is running"
fi

# check if python device SDK is running; if not, start it in the tmux session
if [ -f "/media/sda1/MSR/retailer_detail.ini" ] && [ `pgrep -f ./device_sdk/device.py | wc -l` -eq 0 ];
then
    echo "py_device_sdk is not running. Starting it now..."
    tmux send -t bine-session:0.2 'PYTHONPATH=./device_sdk/ python3 ./device_sdk/device.py' ENTER
else
    echo "hub not authenticated/py_device_sdk is running"
fi

# check if hub_authentication server is running
if [ `pgrep -f ./hub_authentication/main.py | wc -l` -eq 0 ];
then
    echo "hub authentication is not running, starting it"
    tmux send -t bine-session:0.1 'python3 ./hub_authentication/main.py' ENTER
else
    echo "hub authentication is running"
fi
echo "Done executing bine cron job"