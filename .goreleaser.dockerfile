FROM scratch

COPY CalendarSync /calendarsync

ENTRYPOINT ["/calendarsync", "--config", "/etc/calendarsync/sync.yaml", "--port", "8080"]
