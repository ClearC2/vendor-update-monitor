# vendor-update-monitor

An http applicaton that accepts GitHub webhook push events, inspects them for changes to files matching configured patterns, and sends a slack message if changes are found.


## Installation
Copy the`vendor-update-monitor.example.json` file to `vendor-update-monitor.json` and replace the relevant values.

```bash
# download the prebuilt executable
wget https://github.com/ClearC2/vendor-update-monitor/releases/download/<release-tag>/vendor-update-monitor.linux-amd64 

# or clone the repo and build yourself
GOOS=linux GOARCH=amd64 go build -o vendor-update-monitor.linux-amd64 vendor-update-monitor.go

# deploy
scp ./vendor-update-monitor.linux-amd64 user@server:/srv/vendor-update-monitor/
scp ./vendor-update-monitor.json user@server:/srv/vendor-update-monitor/vendor-update-monitor.json
```

Create a service file on the target server to run the application:

```service
# /etc/systemd/system/vendor-update-monitor.service
[Unit]
Description=Go vendor update monitor
After=network-online.target
[Service]
User=root
Restart=on-failure
ExecStart=/srv/vendor-update-monitor/vendor-update-monitor.linux-amd64 /srv/vendor-update-monitor/vendor-update-monitor.json
[Install]
WantedBy=multi-user.target
```
Enable and start the service:
```bash
systemctl enable vendor-update-monitor.service
service vendor-update-monitor start
```

The vendor-update-monitor will be running on port the configured port.

#### Running locally
Create a local config file first.
```bash
# run locally
go run vendor-update-monitor.go ./vendor-update-monitor.json
```
