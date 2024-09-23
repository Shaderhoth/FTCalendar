# FunTech Scraper and Calendar Sync

This project scrapes lesson data from the FunTech website and syncs it with a Google Calendar.
Giant shoutout to Weetile for making the original version in Python.
This is a rewrite in Golang with a few updates.
[Original Python Version by Weetile](https://github.com/Weetile/FunTechTutorScraper)

## Prerequisites

- Go 1.16 or later
- A Google account with access to the Google Calendar API

## Setup

### Step 1: Clone the Repository

```
git clone https://github.com/Shaderhoth/FTCalendar.git
cd FTCalendar
```

### Step 2: Configure the Application

Create a `config.json` file in the project directory with the following content:

```
{
  "username": "your_funtech_username",
  "password": "your_funtech_password",
  "google_client_id": "your_google_client_id",
  "google_client_secret": "your_google_client_secret",
  "google_redirect_uri": "http://localhost:8080",
  "google_calendar_id": "your_google_calendar_id@group.calendar.google.com"
}
```

### Step 3: Get Google Calendar API Credentials

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project.
3. Enable the Google Calendar API for your project.
4. Create OAuth 2.0 Client IDs in the "Credentials" section.
5. Download the `credentials.json` file.
6. Use the contents of this file to fill in the `google_client_id`, `google_client_secret`, and `google_redirect_uri` fields in `config.json`.

### Step 4: Build and Run the Application

#### On Windows

1. Build the Go application:

```
go build -o funtech-scraper main.go
```

2. Run the application:

```
./funtech-scraper
```

### Step 5: Authentication

1. When running for the first time, the application will prompt you to authorize access to your Google Calendar.
2. Follow the link provided and authorize the application.
3. The token will be saved locally for future use.
