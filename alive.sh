#!/bin/bash
crontab /etc/cron.d/alive-cron
cron
./start_hub.sh
sleep infinity