# Postman Collections for API Tasks

This folder contains executable API contract artifacts for MVP tasks that use public APIs and internal service APIs.

## Files
- `LLM-Gateway-MVP.postman_collection.json` — main collection (public + internal contracts).
- `LLM-Gateway-Local.postman_environment.json` — local environment (`localhost` service URLs).

## Covered areas
- Health checks (`/healthz` for all services)
- Auth/org (`register/login/refresh`, members/invite/accept)
- Projects/keys/routing (`/projects`, `/projects/{id}/keys`, `/projects/{id}/routing`)
- Limits (`/limits/projects/{id}`, `/internal/limits/check`)
- Gateway (`/v1/chat/completions`, negative revoked-key scenario)
- Catalog/providers (`/catalog/models`, `/catalog/providers/status`, `/internal/route`)
- Analytics/reports (`/analytics/*`, `/audit`, `/logs/technical`, `/reports/csv/*`)
- Internal ingestion (`/internal/audit/event`, `/internal/usage/record`)

## Run in Postman UI
1. Import collection and environment from this folder.
2. Select environment `LLM Gateway Local`.
3. Ensure local stack is running (`make up`).
4. Run collection in order.

## Run via Newman
```bash
newman run postman/LLM-Gateway-MVP.postman_collection.json \
  -e postman/LLM-Gateway-Local.postman_environment.json
```

## Notes
- Collection auto-generates unique admin email on each run.
- Access/API key/project IDs are captured into collection variables.
- Some requests are intentionally negative (revoked key -> `401 revoked_api_key`).
