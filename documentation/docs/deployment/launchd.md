# launchd (macOS)

The macOS `.pkg` installer drops a LaunchAgent that starts StrataFS at login. To set one up by hand:

## LaunchAgent (per-user)

**`~/Library/LaunchAgents/org.stratafs.daemon.plist`**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>org.stratafs.daemon</string>

  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/stratafs</string>
    <string>serve</string>
  </array>

  <key>RunAtLoad</key>
  <true/>

  <key>KeepAlive</key>
  <dict>
    <key>SuccessfulExit</key>
    <false/>
  </dict>

  <key>StandardOutPath</key>
  <string>/tmp/stratafs.stdout.log</string>

  <key>StandardErrorPath</key>
  <string>/tmp/stratafs.stderr.log</string>

  <key>EnvironmentVariables</key>
  <dict>
    <key>STRATAFS_WORKERS</key>
    <string>4</string>
  </dict>
</dict>
</plist>
```

Load it:

```bash
launchctl load ~/Library/LaunchAgents/org.stratafs.daemon.plist
launchctl list | grep stratafs
```

## LaunchDaemon (system-wide)

For a kiosk or server, install a LaunchDaemon instead. The shape is identical, but the file lives under `/Library/LaunchDaemons/`, owned by `root:wheel` and mode `0644`:

```bash
sudo cp org.stratafs.daemon.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/org.stratafs.daemon.plist
sudo chmod 0644 /Library/LaunchDaemons/org.stratafs.daemon.plist
sudo launchctl load /Library/LaunchDaemons/org.stratafs.daemon.plist
```

Add a `UserName` key to run as a dedicated service account rather than `root`.

## Unloading

```bash
launchctl unload ~/Library/LaunchAgents/org.stratafs.daemon.plist
```

## Logs

`launchctl` redirects to the paths you specified in the plist:

```bash
tail -f /tmp/stratafs.stdout.log /tmp/stratafs.stderr.log
```

For long-lived deployments, redirect both streams to files under `~/Library/Logs/stratafs/` via `StandardOutPath` / `StandardErrorPath` and rotate them with `newsyslog` or `logrotate`.

## Verifying it's healthy

```bash
curl http://localhost:8080/health
```

If the menu bar icon is present, the `.pkg` installer's bundled Wails app is also running. Quit it from the menu bar to disable the UI without affecting the daemon.
