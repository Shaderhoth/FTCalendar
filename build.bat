@echo off
REM Set environment variables for cross-compiling to Linux
set GOOS=linux
set GOARCH=amd64

REM Build the Go application
go build -o funtech-scraper main.go

REM Common credentials
set USER=shaderhoth
set PASS=CukDivad1

REM Generic host for both FTP and SSH
set HOST=shaderhoth.alwaysdata.net

REM Directory on remote server
set REMOTE_DIR=/home/shaderhoth/FTCalendar

REM Use WinSCP to transfer files
echo option batch on> ftpcmd.dat
echo option confirm off>> ftpcmd.dat
echo open ftps://%USER%:%PASS%@ftp-%HOST%:990 -implicit>> ftpcmd.dat
echo mkdir %REMOTE_DIR%>> ftpcmd.dat
echo cd %REMOTE_DIR%>> ftpcmd.dat
echo put funtech-scraper>> ftpcmd.dat
echo put config.json>> ftpcmd.dat
echo put funtech-scraper.sh>> ftpcmd.dat
echo put run_scraper.sh>> ftpcmd.dat
echo put token.json>> ftpcmd.dat
echo exit>> ftpcmd.dat

.\bin\WinSCP.com /script=ftpcmd.dat

REM Clean up WinSCP script file
del ftpcmd.dat

REM Cache the SSH host key
echo y | .\bin\plink -ssh -pw %PASS% %USER%@ssh-%HOST% exit

REM Stop any running versions of the script
.\bin\plink -ssh -batch -pw %PASS% %USER%@ssh-%HOST% "pkill -f funtech-scraper || true"

REM Use plink to set permissions and run the script
.\bin\plink -ssh -batch -pw %PASS% %USER%@ssh-%HOST% "cd %REMOTE_DIR% && chmod +x run_scraper.sh && chmod +x funtech-scraper.sh && ./run_scraper.sh"

echo Deployment completed.
REM pause
