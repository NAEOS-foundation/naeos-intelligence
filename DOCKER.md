# Docker

Build and start NAEOS Intelligence with its PostgreSQL development database:

```sh
docker compose build naeos-intelligence
docker compose up -d
```

The API listens on `http://localhost:8080`. The compose file also defines the
database migration job. Copy `.env.example` to `.env` and set any optional
connector credentials before starting the stack. Docker excludes local
environment files from the image build context.
