# GitHub CI/CD

## CI Scope

GitHub Actions workflow: `.github/workflows/ci-cd.yml`.

CI runs on:
- `push` to `feature/ci-cd` while the CI/CD task is being prepared;
- `pull_request` to `main`;
- `push` to `main`;
- manual `workflow_dispatch`.

Required merge checks match `CONTRIBUTING.md`:
- `make test`;
- `make integration`;
- `make e2e`;
- `npm --prefix frontend run build`.

The workflow also runs the local quality gate, backend build, Docker Compose config validation, container stack build, and smoke flow.

## CD Scope

Minimal CD publishes service images to GitHub Container Registry and deploys them to a Docker Compose host over SSH.

CD runs only after CI jobs pass and only for:
- `push` to `main`;
- manual `workflow_dispatch`.

Deployment artifacts:
- `docker-compose.prod.yml`;
- `db/init.sql`;
- GHCR images tagged with both the commit SHA and `latest`.

## Required GitHub Secrets

- `DEPLOY_HOST` - SSH host/IP of the deployment server.
- `DEPLOY_USER` - SSH user with Docker permissions.
- `DEPLOY_SSH_KEY` - private SSH key for the deploy user.
- `DEPLOY_ENV_FILE` - production `.env` content written to the remote deploy directory.
- `DEPLOY_PATH` - optional remote path, default is `/opt/llm-gateway`.

Example `DEPLOY_ENV_FILE`:

```dotenv
POSTGRES_DB=llm_gateway
POSTGRES_USER=postgres
POSTGRES_PASSWORD=change-me
DATABASE_URL=postgres://postgres:change-me@postgres:5432/llm_gateway?sslmode=disable
JWT_SECRET=change-me
OPENAI_API_KEY=
GEMINI_API_KEY=
GATEWAY_PORT=8080
AUTH_PORT=8081
PROJECT_PORT=8082
LIMITS_PORT=8083
CATALOG_PORT=8084
ANALYTICS_PORT=8085
FRONTEND_PORT=3001
```

## Server Requirements

- Linux host with Docker and Docker Compose plugin installed.
- Deploy user can run Docker commands.
- Required ports are open or routed through a reverse proxy.
- The deployment directory is writable by the deploy user.

## Verification

1. Push the CI/CD branch and check the `CI/CD` workflow in GitHub Actions.
2. Open a PR to `main`; `Quality, Tests, Build` and `Container Build and Smoke` must pass.
3. Configure branch protection according to `docs/github-branch-protection.md`.
4. Merge to `main` or run `workflow_dispatch`.
5. Check that GHCR contains images for all services.
6. Check the deploy job health checks for backend services and frontend.
