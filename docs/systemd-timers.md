# Running CalendarSync periodically using systemd

To run CalendarSync periodically / automatically on specific times using systemd, two files are necessary.

- A [service unit](https://www.freedesktop.org/software/systemd/man/systemd.service.html) file, which we will call: `CalendarSync.service`
- A [timer unit](https://www.freedesktop.org/software/systemd/man/systemd.timer.html) file, which we will call: `CalendarSync.timer`

The following content should be placed under: `.config/systemd/user/CalendarSync.service`:

```systemd
[Unit]
Description=Run CalendarSync

[Service]
ExecStart=/path/to/binary/calendarsync --config path/to/your/sync.yaml --storage-encryption-key $key
```

The following content should be placed under: `.config/systemd/user/CalendarSync.timer`:

```systemd
[Unit]
Description=Run CalendarSync regularly

[Timer]
OnCalendar=Mon..Fri 09:30
OnCalendar=Mon..Fri 11:40
Unit=calendarsync.service

[Install]
WantedBy=default.target
```

For more information check the [systemd docs](https://www.freedesktop.org/software/systemd/man/systemd.timer.html) on timer unit configurations.


After creation of the files, reload systemd using:

```bash
systemctl daemon-reload
```

and enable the service and timer:

```bash
systemctl enable --user CalendarSync.timer
```

check the status with:

```bash
systemctl status --user CalendarSync.timer
# and
systemctl list-timers --user
```

Now at the configured times CalendarSync should run and sync the events without
further intervention 🤞
