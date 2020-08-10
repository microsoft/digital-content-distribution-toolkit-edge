#!/bin/bash
# remove the database file and file system, may not be required in real life but for testing
# TODO: dynamically see the folder instead of hardcode zzzz/
#rm -rf test.db zzzz/


# close ports required for gRPC
fuser -k 50051/tcp
fuser -k 50052/tcp

# start a tmux session with one window[gohub, device_sdk]
cd /usr/bine
tmux new-session -d -s bine-session  # start new detached tmux session, run htop
sleep 2s
tmux split-window -h -t bine-session # split the detached tmux session (two vertical panes)
sleep 2s
tmux split-window -v -t bine-session # split the first tmux pane (two horizontal panes)
sleep 2s
# tmux send -t bine-session:0.right 'conda activate bine' ENTER
tmux send -t bine-session:0.2 'PYTHONPATH=./device_sdk/ python3 ./device_sdk/device.py' ENTER

# enusre that python server is started, hacky
# TODO: write to channel from python and read from that channel here, start gohub only is success 
sleep 5s

tmux send -t bine-session:0.0 './bine_arm' ENTER
tmux send -t bine-session:0.1 'python3 ./hub_authentication/main.py' ENTER