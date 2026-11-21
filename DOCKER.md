# Docker

Build and start NAEOS Intelligence with its PostgreSQL development database:

```sh
docker compose build naeos-intelligence
docker compose up -d
```

The API listens on `http://localhost:8080`. The compose file also defines the
database migration job. Copy `.env.example` to `.env` and configure the
inbound JWKS issuer, audience, and URL plus the Core service token. For the
local Core container, Compose uses
`NAEOS_INTELLIGENCE_DOCKER_CORE_API_BASE_URL=http://host.docker.internal:8081/api`;
the regular `NAEOS_INTELLIGENCE_CORE_API_BASE_URL` remains for running
Intelligence directly on the host. Docker maps `host.docker.internal` to the
host gateway for Linux compatibility. Docker excludes local environment files
from the image build context.
