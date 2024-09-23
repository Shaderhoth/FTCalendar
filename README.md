# FunTech Scraper and Google Calendar Sync

This project scrapes lesson data from the FunTech website and syncs it with a Google Calendar.  
Shoutout to [Weetile](https://github.com/Weetile/FunTechTutorScraper) for creating the original version in Python.  
This is a rewrite in Golang with several updates.

## Prerequisites

- Go 1.16 or later
- A Google account with access to the Google Calendar API

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
  "google_client_id": "your_google_client_id",
  "google_client_secret": "your_google_client_secret",
  "google_redirect_uri": "http://localhost:8080"
}
```

### Step 3: Get Google Calendar API Credentials

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project.
3. Enable the **Google Calendar API** for your project.
4. In the "Credentials" section, create OAuth 2.0 Client IDs.
5. Download the `credentials.json` file.
6. Use the contents of this file to fill in the `google_client_id`, `google_client_secret`, and `google_redirect_uri` fields in your `config.json`.

### Step 4: Deploy to alwaysdata

#### On Windows

Here's a batch script (`.bat`) I use to automate the deployment process. It uses **WinSCP** for file transfers and **Plink** for setting file permissions via SSH. Let me know if you need help setting up these tools.

```batch
@echo off
REM Set environment variables for cross-compiling to Linux
set GOOS=linux
set GOARCH=amd64

REM Build the Go applications
go build -o funtech-web-server funtech_web_server.go
go build -o funtech-daemon funtech_daemon.go

REM Common credentials
set USER=--USERNAME HERE--
set PASS=--PASSWORD HERE--

REM Generic host for both FTP and SSH
set HOST=%USER%.alwaysdata.net

REM Directory on the remote server
set REMOTE_DIR=/home/%USER%/FTCalendar

REM Use WinSCP to transfer files
echo option batch on> ftpcmd.dat
echo option confirm off>> ftpcmd.dat
echo open ftps://%USER%:%PASS%@ftp-%HOST%:990 -implicit>> ftpcmd.dat

REM Navigate to remote directory
echo cd %REMOTE_DIR%>> ftpcmd.dat

REM Remove existing files
echo rm funtech-web-server>> ftpcmd.dat

REM Upload files and directories
echo put funtech-web-server>> ftpcmd.dat
echo put funtech-daemon>> ftpcmd.dat
echo put config\common_config.json ./config/>> ftpcmd.dat
echo put config\user_configs\*.json ./config/user_configs/>> ftpcmd.dat
echo put site\templates\* ./site/templates/>> ftpcmd.dat
echo put cert.pem>> ftpcmd.dat
echo put key.pem>> ftpcmd.dat
echo exit>> ftpcmd.dat

.\bin\WinSCP.com /script=ftpcmd.dat

REM Check if WinSCP succeeded
if %ERRORLEVEL% neq 0 (
    echo WinSCP upload failed
    exit /b 1
)

REM Clean up WinSCP script file
del ftpcmd.dat

REM Set executable permissions for the files using SSH
echo chmod +x %REMOTE_DIR%/funtech-web-server %REMOTE_DIR%/funtech-daemon > sshcmd.sh

REM Use plink to set permissions
.\bin\plink.exe -ssh %USER%@ssh-%HOST% -pw %PASS% -v -m sshcmd.sh

REM Check if plink succeeded
if %ERRORLEVEL% neq 0 (
    echo Plink command failed
    exit /b 1
)

REM Clean up SSH script file
del sshcmd.sh

echo Deployment completed.
```

### Step 5: Folder Structure on alwaysdata

After deployment, you should have the following file structure on your alwaysdata server:

```
.
├── cert.pem
├── config
│   ├── common_config.json
│   └── user_configs
│       └── dkuc.json
├── funtech-daemon
├── funtech-web-server
├── key.pem
└── site
    └── templates
        ├── auth.html
        ├── dashboard.html
        └── style.css
```

- The **user configs** are generated automatically upon registration, so no need to worry about those.
- The **key.pem** and **cert.pem** are for SSL encryption and are used with the batch script.
- Everything else is uploaded from the GitHub repository (except the **common_config.json** file, which you'll need to create).

### Step 6: Start the FunTech Services

1. Set up a service for `funtech-daemon`.
2. Set up a process for `funtech-web-server`.
3. Ensure the command for each executable is set as: `./{executable_name}`.

---

With this setup, your FunTech scraper and Google Calendar synchronization should be working smoothly!

Let me know if you run into any issues during deployment!
