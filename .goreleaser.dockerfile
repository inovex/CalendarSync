FROM scratch

COPY calendarsync /calendarsync

ENTRYPOINT ["/calendarsync", "--config", "/etc/calendarsync/sync.yaml", "--port", "8080"]