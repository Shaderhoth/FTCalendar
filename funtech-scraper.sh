#!/bin/bash
while true; do
    /home/shaderhoth/FTCalendar/funtech-scraper >> /home/shaderhoth/FTCalendar/scraper.log 2>&1
    sleep 3600  # Sleep for 1 hour
done
