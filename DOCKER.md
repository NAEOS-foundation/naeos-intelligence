# Docker

Build the service and start the local PostgreSQL database:

```sh
docker compose build naeos-intelligence
docker compose up -d postgres
```

Apply migrations explicitly using the migrator role:

```sh
docker compose --profile migrate run --rm naeos-intelligence-migrate
```

Then start the API:

```sh
docker compose up -d naeos-intelligence
```

The API listens on `http://localhost:8080`. PostgreSQL is exposed on
`localhost:5433` for local development. The first-time database initialization
provisions separate runtime and migrator roles. Local default passwords are for
development only; configure `RUNTIME_DB_PASSWORD` and
`MIGRATOR_DB_PASSWORD` before initializing a new volume.

The API authenticates as `naeos_intelligence_runtime`; the explicit migration
command authenticates as `naeos_intelligence_migrator`. Never share credentials
or run migrations as the runtime role. If an existing data volume predates role
provisioning, back up local data before recreating it; init scripts do not rerun
automatically.

Copy `.env.example` to `.env` and set optional connector credentials. Docker
excludes local environment files from the image build context.
