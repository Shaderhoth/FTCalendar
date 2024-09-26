@echo off

REM Build the Go applications
go build -o funtech-web-server.exe funtech_web_server.go
go build -o funtech-daemon.exe funtech_daemon.go

REM Check if the build was successful
if not exist funtech-daemon.exe (
    echo "funtech-daemon.exe was not found. Build failed."
    pause
    exit /b 1
)

REM Run the application
funtech-daemon.exe

REM pause if you want to see the output in case of a failure
pause
