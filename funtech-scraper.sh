#!/bin/bash

# Clear the log file
: > /home/shaderhoth/FTCalendar/scraper.log

while true; do
    /home/shaderhoth/FTCalendar/funtech-scraper >> /home/shaderhoth/FTCalendar/scraper.log 2>&1
    sleep 600  # Sleep for 1 hour
done
