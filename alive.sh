#!/bin/bash
crontab /etc/cron.d/alive-cron
./start_hub.sh
sleep infinity