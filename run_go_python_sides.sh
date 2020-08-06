# check if goserver is running; if not, start it in the tmux session
if [ -f "/media/sda1/MSR/retailer_detail.ini" ] && [ `pgrep -f ./bine_arm | wc -l` -eq 0 ];
then
    echo "goserver is not running. Starting it now..."
    tmux send -t bine-session:0.left './bine_arm' ENTER
else
    echo "hub not authenticated/goserver is running"
fi

# check if python device SDK is running; if not, start it in the tmux session
if [ -f "/media/sda1/MSR/retailer_detail.ini" ] && [ `pgrep -f ./device_sdk/device.py | wc -l` -eq 0 ];
then
    echo "py_device_sdk is not running. Starting it now..."
    tmux send -t bine-session:0.right 'PYTHONPATH=./device_sdk/ python3 ./device_sdk/device.py' ENTER
else
    echo "hub not authenticated/py_device_sdk is running"
fi