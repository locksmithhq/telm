FROM postgres:16

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        postgresql-16-pgaudit \
        postgresql-contrib \
    && rm -rf /var/lib/apt/lists/*

# Create the log directory owned by the postgres user so that the
# logging_collector subprocess can write there when the pg-logs
# Docker volume is first mounted.
RUN mkdir -p /var/log/postgresql \
    && chown postgres:postgres /var/log/postgresql \
    && chmod 755 /var/log/postgresql
