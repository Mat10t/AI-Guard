# План задач MVP LLM Gateway

## Структура
- `00-planning`
- `01-infrastructure`
- `02-api-gateway`
- `03-auth-org-service`
- `04-project-key-service`
- `05-limits-service`
- `06-audit-analytics-service`
- `07-provider-catalog-service`
- `08-frontend`
- `09-testing`

## Порядок выполнения
1. `00-planning`
2. `01-infrastructure`
3. `03-auth-org-service` + `04-project-key-service`
4. `05-limits-service`
5. `07-provider-catalog-service`
6. `02-api-gateway`
7. `06-audit-analytics-service`
8. `08-frontend` (mobile-first -> desktop adaptation)
9. `09-testing`

## Ключевые зависимости
- Контракты из `00-planning` обязательны до реализации API и Kafka-событий.
- Инфраструктура из `01-infrastructure` обязательна для локального запуска и интеграционных тестов.
- `02-api-gateway` зависит от `04-project-key-service`, `05-limits-service`, `07-provider-catalog-service`.
- `06-audit-analytics-service` зависит от Kafka-событий из `02-api-gateway` и бизнес-событий из `04/05`.
- `08-frontend` зависит от готовых REST API из `03/04/05/06/07`.
- `09-testing` выполняется итеративно, финальный e2e проход после интеграции всех сервисов.

## Git Workflow (для новых задач)
- Правило: `1 задача = 1 ветка = 1 PR`.
- Нельзя начинать реализацию без task-id из имени файла задачи.
- Формат ветки: `feat/<task-id>-<slug>` или `fix/<task-id>-<slug>`.
- Формат commit message: `<task-id>: ...`.
- В PR запрещено смешивать несколько задач.
- В каждой новой задаче обязательны service-поля:
  - `Branch:`
  - `PR:`
  - `Merged at:`
- Шаблон новой задачи: `tasks/TEMPLATE.md`.
- Шаблон PR: `.github/pull_request_template.md`.

## MVP vs отложенное
### Обязательно для MVP
- Все задачи `00-01` .. `09-03`.
- Расширения MVP limits/budget:
  - `05-03-project-budget-sync.md`
  - `08-14-project-budget-sync-ui.md`
  - `09-10-budget-limit-regression.md`
- Расширения MVP UX/RBAC:
  - `03-03-members-soft-delete.md`
  - `08-12-project-delete-typed-confirm.md`
  - `08-13-member-delete-admin-flow.md`
  - `09-09-project-member-delete-regression.md`
- Расширения MVP key-UX:
  - `08-15-api-key-masked-display-copy-only.md`
  - `09-11-api-key-mask-copy-regression.md`
  - зависимость: `08-15 -> 09-11`
- Расширения MVP OpenAI 5.4:
  - `07-03-openai-54-model-catalog.md`
  - `05-04-default-billing-model-openai-54.md`
  - `02-05-dynamic-model-pricing-from-provider-models.md`
  - `09-12-openai-54-catalog-pricing-regression.md`
  - зависимости и порядок: `07-03 -> 05-04 -> 02-05 -> 09-12`
- Расширения MVP+ multi-key / analytics / gemini:
  - `00-03-requirements-update-multi-key.md`
  - `04-04-project-multi-key-management.md`
  - `06-03-analytics-sections-timeseries-and-csv.md`
  - `07-04-gemini-provider-adapter.md`
  - `08-16-analytics-sections-nav-and-charts.md`
  - `09-13-multi-key-analytics-gemini-regression.md`
  - зависимости и порядок: `00-03 -> 04-04 -> (06-03 + 07-04) -> 08-16 -> 09-13`
- Расширения MVP+ project-level fallback routing:
  - `04-05-project-fallback-routing-settings.md`
  - `07-05-route-resolution-with-project-fallback.md`
  - `02-06-fallback-model-override-by-project.md`
  - `08-17-project-fallback-settings-ui.md`
  - `09-14-project-fallback-routing-regression.md`
  - зависимости и порядок: `04-05 -> 07-05 -> 02-06 -> 08-17 -> 09-14`
- Расширения MVP+: UX/analytics/provider-status/pricing hardening:
  - `00-04-provider-status-pricing-and-analytics-contracts.md`
  - `08-18-destructive-actions-refresh-feedback.md`
  - `06-04-timeseries-input-output-tokens.md`
  - `08-19-usage-in-out-lines.md`
  - `07-06-live-provider-status-checks.md`
  - `07-07-catalog-pricing-admin-update.md`
  - `09-15-delete-usage-provider-pricing-regression.md`
  - зависимости и порядок: `00-04 -> (06-04 + 07-06 + 07-07) -> (08-18 + 08-19) -> 09-15`
- Расширения MVP+: autosync цен и project limits:
  - `05-05-project-limit-sync-source-persistence.md`
  - `07-08-pricing-auto-sync-project-limits.md`
  - `08-20-pricing-update-autosync-feedback.md`
  - `09-16-pricing-limit-autosync-regression.md`
  - зависимости и порядок: `05-05 -> 07-08 -> 08-20 -> 09-16`
- Расширения MVP+: pricing controls только для Admin:
  - `08-21-providers-pricing-admin-only-controls.md`
  - `09-17-providers-pricing-visibility-regression.md`
  - зависимость: `08-21 -> 09-17`
- Расширения MVP+: scoped analytics + project assignment:
  - `00-05-scoped-analytics-and-project-assignment-contracts.md`
  - `03-04-invite-with-project-assignment.md`
  - `04-06-project-members-management-and-access-filtering.md`
  - `06-05-scoped-analytics-org-project-key.md`
  - `08-22-analytics-scope-switcher-and-project-member-ui.md`
  - `09-18-scoped-analytics-project-assignment-regression.md`
  - зависимости и порядок: `00-05 -> 03-04 -> 04-06 -> 06-05 -> 08-22 -> 09-18`
- Расширения MVP+: strict audit scope (hard precision):
  - `00-06-audit-hard-scope-columns-contract.md`
  - `06-06-audit-hard-scope-columns-implementation.md`
  - `09-19-audit-hard-scope-regression.md`
  - зависимости и порядок: `00-06 -> 06-06 -> 09-19`
- Расширения MVP+: actual model accounting in usage (fallback-aware):
  - `00-07-effective-model-accounting-contract.md`
  - `02-07-effective-model-in-usage-recording.md`
  - `06-07-usage-effective-model-storage-and-queries.md`
  - `09-20-effective-model-usage-regression.md`
  - зависимости и порядок: `00-07 -> 02-07 -> 06-07 -> 09-20`
- Расширения MVP+: AI Studio-inspired frontend redesign:
  - `08-23-ai-studio-inspired-design-tokens-and-primitives.md`
  - `08-24-shell-sidebar-drawer-navigation.md`
  - `08-25-projects-table-and-detail-panels.md`
  - `08-26-analytics-sections-ai-studio-layout.md`
  - `08-27-auth-and-landing-dark-ai-studio-flow.md`
  - `09-21-frontend-ai-studio-regression-checklist.md`
  - зависимости и порядок: `08-23 -> 08-24 -> (08-25 + 08-26 + 08-27) -> 09-21`
- Расширения MVP+: AI Guard UI refactor (sidebar + API keys hub + page split):
  - `00-08-ai-guard-navigation-and-keys-page-contracts.md`
  - `04-07-api-key-name-support.md`
  - `08-28-sidebar-fix-and-ai-guard-brand.md`
  - `08-29-api-keys-page-modal-grouping-and-key-scope-jump.md`
  - `08-30-projects-page-slim-fallback-limits-delete.md`
  - `08-31-organization-members-grouped-view-and-invite-modal.md`
  - `08-32-analytics-nav-split-and-console-removal.md`
  - `09-22-ai-guard-ui-structure-regression.md`
  - зависимости и порядок: `00-08 -> 04-07 -> (08-28 + 08-29 + 08-30 + 08-31 + 08-32) -> 09-22`
- Расширения MVP+: AI Guard frontend fix (nested modal + projects action modals + style tuning):
  - `00-09-keys-projects-modal-flow-and-style-contracts.md`
  - `08-33-api-keys-nested-project-create-modal.md`
  - `08-34-projects-list-summary-and-action-modals.md`
  - `08-35-ai-studio-colors-buttons-layout-tuning.md`
  - `09-23-frontend-keys-projects-modal-regression.md`
  - зависимости и порядок: `00-09 -> (08-33 + 08-34 + 08-35) -> 09-23`
- Расширения MVP+: copy active key anytime + hide revoked keys:
  - `00-10-active-key-copy-and-revoked-visibility-contracts.md`
  - `04-08-list-keys-with-secret-for-active.md`
  - `08-36-copy-anytime-and-hide-revoked-everywhere.md`
  - `09-24-active-key-copy-hide-revoked-regression.md`
  - зависимости и порядок: `00-10 -> 04-08 -> 08-36 -> 09-24`
- Расширения MVP+: Git workflow enforcement (`1 задача = 1 ветка = 1 PR`):
  - `00-11-git-workflow-branch-pr-enforcement.md`
  - зависимости и порядок: `00-11 -> применяется как process-gate ко всем новым задачам`

- Расширения MVP+: Postman collections для API-задач:
  - `09-25-postman-collections-api-contracts.md`

### Можно отложить после MVP
- `01-04-observability-dashboards-advanced.md`
- `02-04-gateway-streaming-response.md`
- `07-02-multi-fallback-chain.md`
- `09-04-load-tests-k6.md`
