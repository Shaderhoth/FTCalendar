#!/bin/bash

# Name of the screen session
SESSION_NAME="funtech-scraper"

# Path to your script
SCRIPT_PATH="/home/shaderhoth/FTCalendar/funtech-scraper.sh"

# Ensure the script and the binary are executable
chmod +x $SCRIPT_PATH
chmod +x /home/shaderhoth/FTCalendar/funtech-scraper

# Start a new screen session and run the script
screen -dmS $SESSION_NAME $SCRIPT_PATH

echo "Started funtech-scraper in a new screen session named '$SESSION_NAME'."
