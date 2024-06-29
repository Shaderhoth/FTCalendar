# FunTech Scraper and Calendar Sync

This project scrapes lesson data from the FunTech website, generates an ICS file, uploads it to GitHub, and syncs it with a Google Calendar.

## Prerequisites

- Go 1.16 or later
- A GitHub account and repository
- A Google account with access to the Google Calendar API
- A hosting environment to run the compiled Go application (Debian in this example)

## Setup

### Step 1: Clone the Repository

```bash
git clone https://github.com/Shaderhoth/FTCalendar.git
cd FTCalendar
```

### Step 2: Configure the Application

Create a `config.json` file in the project directory with the following content:

```json
{
  "username": "your_funtech_username",
  "password": "your_funtech_password",
  "github_token": "your_github_token",
  "github_repo": "your_github_username/your_github_repo",
  "github_path": "out/calendar.ics",
  "google_client_id": "your_google_client_id",
  "google_client_secret": "your_google_client_secret",
  "google_redirect_uri": "http://localhost:8080",
  "google_calendar_id": "your_google_calendar_id@group.calendar.google.com"
}
```

### Step 3: Obtain GitHub Token

1. Go to your [GitHub settings](https://github.com/settings/tokens).
2. Click on "Generate new token".
3. Select the scopes for `repo` and `public_repo` to allow the token to read and write to your repositories.
4. Generate the token and copy it.
5. Paste this token in the `github_token` field in `config.json`.

### Step 4: Set GitHub Repository and Path

1. Create a new repository on GitHub if you don't have one already.
2. Set the `github_repo` field in `config.json` to `your_github_username/your_github_repo`.
3. Set the `github_path` field to the path where you want the `calendar.ics` file to be stored in your repository, e.g., `out/calendar.ics`.

### Step 5: Get Google Calendar API Credentials

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project.
3. Enable the Google Calendar API for your project.
4. Create OAuth 2.0 Client IDs in the "Credentials" section.
5. Download the `credentials.json` file.
6. Use the contents of this file to fill in the `google_client_id`, `google_client_secret`, and `google_redirect_uri` fields in `config.json`.

### Step 6: Build and Run the Application

#### On Windows

1. Build the Go application for Debian:

```bash
env GOOS=linux GOARCH=amd64 go build -o funtech-scraper main.go
```

2. Transfer the binary and `config.json` to your Debian server.

#### On Debian

1. Upload the files to your server:

```bash
scp funtech-scraper config.json user@your_debian_server:/path/to/directory
```

2. Connect to your server and navigate to the directory where you uploaded the files.
3. Ensure the `config.json` file has the correct permissions:

```bash
chmod 644 config.json
```

4. Run the application:

```bash
./funtech-scraper
```

### Step 7: Authentication

1. When running for the first time, the application will prompt you to authorize access to your Google Calendar.
2. Follow the link provided and authorize the application.
3. The token will be saved locally for future use.

## Troubleshooting

### Config File Not Found

Ensure that `config.json` is in the same directory as your binary and has the correct permissions.

### Permissions Issues

Make sure that the `config.json` file is readable by the user running the application.

### Duplicate Events

The application checks for existing events and avoids creating duplicates. If you encounter issues, make sure the ICS file is correctly generated.

## Contributing

Feel free to fork this repository and submit pull requests. Contributions are welcome!

## Hosting on Debian

1. Ensure you have a Debian server set up.
2. Transfer the compiled binary and `config.json` to your server.
3. Make sure you have the necessary permissions to execute the binary and read the config file.

### Example Commands

```bash
scp funtech-scraper config.json user@your_debian_server:/path/to/directory
ssh user@your_debian_server
cd /path/to/directory
chmod +x funtech-scraper
chmod 644 config.json
./funtech-scraper
```

### Setting Up as a Service

For continuous running, you can set up the scraper as a service using `systemd`.

1. Create a service file: `/etc/systemd/system/funtech-scraper.service`

```ini
[Unit]
Description=FunTech Scraper Service
After=network.target

[Service]
User=your_user
WorkingDirectory=/path/to/directory
ExecStart=/path/to/directory/funtech-scraper
Restart=always

[Install]
WantedBy=multi-user.target
```

2. Reload the systemd manager configuration:

```bash
sudo systemctl daemon-reload
```

3. Enable and start the service:

```bash
sudo systemctl enable funtech-scraper
sudo systemctl start funtech-scraper
```

4. Check the status of the service:

```bash
sudo systemctl status funtech-scraper
```

This will ensure that your scraper runs continuously and restarts automatically if it fails.
