# systemd (Linux)

The Debian package installs a `systemd --user` unit by default. For a system-wide install or a custom setup, drop the unit below into place and reload.

## User-level service (recommended)

A user service starts when you log in and stops when you log out. No root required.

**`~/.config/systemd/user/stratafs.service`**

```ini
[Unit]
Description=StrataFS semantic filesystem daemon
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/stratafs serve
Restart=on-failure
RestartSec=5
Environment=STRATAFS_LOG_LEVEL=info

[Install]
WantedBy=default.target
```

Enable and start:

```bash
systemctl --user daemon-reload
systemctl --user enable --now stratafs
systemctl --user status stratafs
```

Logs:

```bash
journalctl --user -u stratafs -f
```

To keep the service running after logout (so an idle search agent can still query the API):

```bash
sudo loginctl enable-linger "$USER"
```

## System-wide service

For a multi-user host or a server-style install:

**`/etc/systemd/system/stratafs.service`**

```ini
[Unit]
Description=StrataFS semantic filesystem daemon
After=network-online.target

[Service]
Type=simple
User=stratafs
Group=stratafs
ExecStart=/usr/bin/stratafs serve --config-dir /var/lib/stratafs
Restart=on-failure
RestartSec=5
Environment=STRATAFS_LOG_LEVEL=info

# Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=/var/lib/stratafs

[Install]
WantedBy=multi-user.target
```

Prepare the service account and data directory:

```bash
sudo useradd --system --home /var/lib/stratafs --shell /usr/sbin/nologin stratafs
sudo install -d -o stratafs -g stratafs /var/lib/stratafs
sudo -u stratafs stratafs --config-dir /var/lib/stratafs config init
sudo systemctl daemon-reload
sudo systemctl enable --now stratafs
```

`ProtectHome=read-only` means StrataFS can still **read** user home directories (necessary for indexing local sources) but cannot write to them. The only writable path is `/var/lib/stratafs`.

## Auto-restart on config changes

For a service that picks up config edits without a manual restart, add a path unit:

**`~/.config/systemd/user/stratafs-config.path`**

```ini
[Unit]
Description=Watch StrataFS config

[Path]
PathChanged=%h/.stratafs/config.json

[Install]
WantedBy=default.target
```

**`~/.config/systemd/user/stratafs-config.service`**

```ini
[Unit]
Description=Restart StrataFS on config change

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl --user restart stratafs.service
```

```bash
systemctl --user enable --now stratafs-config.path
```

## Verifying it's healthy

```bash
curl http://localhost:8080/health
curl -s http://localhost:8080/queue/stats | jq
```

If `pending_jobs` stays flat, the worker pool may be saturated — bump `worker.count`.
