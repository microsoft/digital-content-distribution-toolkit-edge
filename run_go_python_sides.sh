
# check if goserver is running; if not, start it in the tmux session
if [ `pgrep -f ./bine | wc -l` -eq 0 ];
then
    echo "goserver is not running. Starting it now..."
    tmux send -t bine-session:0.left './bine' ENTER
else
    echo "goserver is running"
fi

# check if python device SDK is running; if not, start it in the tmux session
if [ `pgrep -f ./device_sdk/device.py | wc -l` -eq 0 ];
then
    echo "py_device_sdk is not running. Starting it now..."
    tmux send -t bine-session:0.right 'PYTHONPATH=./device_sdk/ python3 ./device_sdk/device.py' ENTER
else
    echo "py_device_sdk is running"
fi
