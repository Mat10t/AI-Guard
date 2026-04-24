package e2e

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"llm-gateway-mvp/tests/testutil"
)

func TestCriticalMVPFlows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e in short mode")
	}
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run e2e tests")
	}
	env := testutil.LoadEnv()
	client := testutil.NewClient()

	email := testutil.RandomEmail("e2e")
	password := "password123"

	status, reg, raw := testutil.RequestJSON(t, client, http.MethodPost, env.AuthURL+"/auth/register", "", map[string]any{
		"org_name": "E2E Org",
		"email":    email,
		"password": password,
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	token := testutil.MustString(t, reg, "access_token")
	adminUserID := testutil.MustString(t, reg, "user_id")

	status, prj1, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects", token, map[string]any{"name": "E2E Main"})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	project1 := testutil.MustString(t, prj1, "id")

	status, key1Resp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+project1+"/keys", token, map[string]any{
		"name": "E2E primary key",
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	apiKey1 := testutil.MustString(t, key1Resp, "api_key")
	key1ID := testutil.MustString(t, key1Resp, "id")
	if gotName := testutil.MustString(t, key1Resp, "name"); gotName != "E2E primary key" {
		t.Fatalf("expected key name to match request, body=%s", string(raw))
	}

	status, key1bResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+project1+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	apiKey1b := testutil.MustString(t, key1bResp, "api_key")
	key1bID := testutil.MustString(t, key1bResp, "id")

	status, keyList1, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects/"+project1+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasKeyID(keyList1, key1ID) || !hasKeyID(keyList1, key1bID) {
		t.Fatalf("expected both keys in list for project1, body=%s", string(raw))
	}
	if !hasKeyName(keyList1, key1ID, "E2E primary key") {
		t.Fatalf("expected named key in list for project1, body=%s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey1, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "hello"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey1b, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "hello from second key"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, limitByTokensResp, raw := testutil.RequestJSON(t, client, http.MethodPut, env.LimitsURL+"/limits/projects/"+project1, token, map[string]any{
		"token_limit":   10,
		"billing_model": "gpt-5.4-mini",
		"period":        "day",
		"sync_source":   "tokens",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	usdPerToken, ok := limitByTokensResp["usd_per_token"].(float64)
	if !ok || usdPerToken <= 0 {
		t.Fatalf("expected usd_per_token in budget sync response, body=%s", string(raw))
	}

	status, getLimitResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.LimitsURL+"/limits/projects/"+project1, token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if gotModel, _ := getLimitResp["billing_model"].(string); gotModel != "gpt-5.4-mini" {
		t.Fatalf("expected billing_model=gpt-5.4-mini, got=%q body=%s", gotModel, string(raw))
	}

	status, limitByBudgetResp, raw := testutil.RequestJSON(t, client, http.MethodPut, env.LimitsURL+"/limits/projects/"+project1, token, map[string]any{
		"budget_limit_usd": usdPerToken,
		"billing_model":    "gpt-5.4-mini",
		"period":           "day",
		"sync_source":      "budget",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if tokenLimit, _ := limitByBudgetResp["token_limit"].(float64); int(tokenLimit) != 1 {
		t.Fatalf("expected token_limit=1 after budget sync, got body=%s", string(raw))
	}

	status, limitResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey1, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "this request should be blocked by project limit"}},
	})
	if status != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d body=%s", status, string(raw))
	}
	if code, _ := limitResp["code"].(string); code != "limit_exceeded" {
		t.Fatalf("expected code=limit_exceeded, got body=%s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+project1+"/keys/"+key1ID+"/revoke", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, revokedResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey1, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "must fail after revoke"}},
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked key, got %d body=%s", status, string(raw))
	}
	if code, _ := revokedResp["code"].(string); code != "revoked_api_key" {
		t.Fatalf("expected revoked_api_key, got body=%s", string(raw))
	}

	status, prj2, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects", token, map[string]any{"name": "E2E Fallback"})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	project2 := testutil.MustString(t, prj2, "id")

	status, routingResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects/"+project2+"/routing", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if v, ok := routingResp["fallback_model_id"]; ok && v != nil {
		t.Fatalf("expected empty fallback_model_id by default, got body=%s", string(raw))
	}

	status, routingResp, raw = testutil.RequestJSON(t, client, http.MethodPut, env.ProjectURL+"/projects/"+project2+"/routing", token, map[string]any{
		"fallback_model_id": "mock-fast",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if got, _ := routingResp["fallback_model_id"].(string); got != "mock-fast" {
		t.Fatalf("expected fallback_model_id=mock-fast, got body=%s", string(raw))
	}

	status, key2Resp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+project2+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	apiKey2 := testutil.MustString(t, key2Resp, "api_key")
	key2ID := testutil.MustString(t, key2Resp, "id")

	status, key3Resp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+project2+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	apiKey3 := testutil.MustString(t, key3Resp, "api_key")
	key3ID := testutil.MustString(t, key3Resp, "id")

	status, keyList2, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects/"+project2+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasKeyID(keyList2, key2ID) || !hasKeyID(keyList2, key3ID) {
		t.Fatalf("expected both keys in list for project2, body=%s", string(raw))
	}

	status, prj3, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects", token, map[string]any{"name": "E2E Default Route"})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	project3 := testutil.MustString(t, prj3, "id")

	status, key4Resp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+project3+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	apiKey4 := testutil.MustString(t, key4Resp, "api_key")

	status, fallbackOverrideResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey2, map[string]any{
		"model":    "gpt-5.4-mini",
		"messages": []map[string]any{{"role": "user", "content": "fallback please"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, fallbackDefaultResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey4, map[string]any{
		"model":    "gpt-5.4-mini",
		"messages": []map[string]any{{"role": "user", "content": "fallback default route please"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	overrideContent := extractAssistantContent(fallbackOverrideResp)
	defaultContent := extractAssistantContent(fallbackDefaultResp)
	if strings.Contains(overrideContent, "mock-fallback") && strings.Contains(defaultContent, "mock-fallback") {
		if model, _ := fallbackOverrideResp["model"].(string); model != "mock-fast" {
			t.Fatalf("expected override fallback response model=mock-fast, got body=%s", string(raw))
		}
		if model, _ := fallbackDefaultResp["model"].(string); model != "gpt-5.4-mini" {
			t.Fatalf("expected default fallback response model=gpt-5.4-mini, got body=%s", string(raw))
		}
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+project2+"/keys/"+key2ID+"/revoke", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey2, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "must fail after revoke key2"}},
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked key2, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey3, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "second key still works"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	time.Sleep(700 * time.Millisecond)

	status, logsResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/logs/technical?api_key_id="+key2ID+"&limit=50", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	fallbackObserved := hasFallbackLog(logsResp)
	if fallbackObserved {
		if !hasFallbackLogWithModel(logsResp, "mock-fast") {
			t.Fatalf("expected fallback_model=mock-fast when fallback is used, got body=%s", string(raw))
		}
	} else if !hasCompletedLogForModel(logsResp, "gpt-5.4-mini") {
		t.Fatalf("expected either fallback log or successful primary log for gpt-5.4-mini, got body=%s", string(raw))
	}

	status, auditResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/audit?limit=300", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasAuditAction(auditResp, "api_key.created") || !hasAuditAction(auditResp, "limit.updated") {
		t.Fatalf("expected api_key.created and limit.updated in audit, got body=%s", string(raw))
	}

	status, projectScopedAuditResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/audit?scope=project&project_id="+project2+"&limit=200", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasAuditAction(projectScopedAuditResp, "project.created") {
		t.Fatalf("expected project-scoped audit to include project.created for project2, got body=%s", string(raw))
	}

	status, keyScopedAuditResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/audit?scope=key&project_id="+project2+"&api_key_id="+key2ID+"&limit=200", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasAuditAction(keyScopedAuditResp, "api_key.created") {
		t.Fatalf("expected key-scoped audit to include api_key.created for key2, got body=%s", string(raw))
	}
	if hasAuditAction(keyScopedAuditResp, "project.created") {
		t.Fatalf("key-scoped audit must not include project.created, got body=%s", string(raw))
	}

	status, usageResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/analytics/usage?group_by=model", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasPositiveUsage(usageResp) {
		t.Fatalf("expected positive usage data, got body=%s", string(raw))
	}
	if !hasUsageForModelWithPositiveCost(usageResp, "gpt-5.4-mini") {
		t.Fatalf("expected positive cost usage for model=gpt-5.4-mini, got body=%s", string(raw))
	}
	if !hasUsageGroup(usageResp, "mock-fast") {
		t.Fatalf("expected usage grouped by effective fallback model=mock-fast, got body=%s", string(raw))
	}

	status, _, csvRaw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/reports/csv/usage", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, csvRaw)
	if !strings.Contains(string(csvRaw), "day,project_id,model,requested_model,effective_model,total_tokens,total_cost") {
		t.Fatalf("unexpected csv header: %s", string(csvRaw))
	}
	status, _, auditCSVRaw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/reports/csv/audit", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, auditCSVRaw)
	if !strings.Contains(string(auditCSVRaw), "created_at,actor_user_id,action,object_type,object_id,details_json") {
		t.Fatalf("unexpected audit csv header: %s", string(auditCSVRaw))
	}
	status, _, logsCSVRaw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/reports/csv/logs", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, logsCSVRaw)
	if !strings.Contains(string(logsCSVRaw), "created_at,request_id,project_id,api_key_id,model,status,error_code,retries,fallback_used,fallback_model,input_tokens,output_tokens") {
		t.Fatalf("unexpected logs csv header: %s", string(logsCSVRaw))
	}
	status, _, raw = testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/analytics/timeseries?metric=tokens&bucket=day", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	status, _, raw = testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/analytics/timeseries?metric=input_tokens&bucket=day", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	status, _, raw = testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/analytics/timeseries?metric=output_tokens&bucket=day", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	status, _, raw = testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/analytics/timeseries?metric=cost&bucket=all", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	status, _, raw = testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/analytics/timeseries?metric=error_rate&bucket=day", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	status, _, raw = testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/analytics/timeseries?metric=fallback_rate&bucket=day", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	inviteEmail := testutil.RandomEmail("e2e-dev")
	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.AuthURL+"/org/members/invite", token, map[string]any{
		"email": inviteEmail,
		"role":  "Dev",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for Dev invite without project_ids, got %d body=%s", status, string(raw))
	}
	status, inviteResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.AuthURL+"/org/members/invite", token, map[string]any{
		"email":       inviteEmail,
		"role":        "Dev",
		"project_ids": []string{project3},
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	inviteToken := testutil.MustString(t, inviteResp, "invitation_token")

	inviteeClient := testutil.NewClient()
	status, acceptResp, raw := testutil.RequestJSON(t, inviteeClient, http.MethodPost, env.AuthURL+"/org/members/accept", "", map[string]any{
		"token":    inviteToken,
		"password": "password123",
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	devToken := testutil.MustString(t, acceptResp, "access_token")
	devUserID := testutil.MustString(t, acceptResp, "user_id")
	if role := testutil.MustString(t, acceptResp, "role"); role != "Dev" {
		t.Fatalf("expected invited role=Dev, got=%q body=%s", role, string(raw))
	}

	status, refreshedDev, raw := testutil.RequestJSON(t, inviteeClient, http.MethodPost, env.AuthURL+"/auth/refresh", "", nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	_ = testutil.MustString(t, refreshedDev, "access_token")

	status, projectMembersResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects/"+project3+"/members", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasProjectMember(projectMembersResp, devUserID) {
		t.Fatalf("expected Dev membership in assigned project3, body=%s", string(raw))
	}

	status, devProjectsResp, raw := testutil.RequestJSON(t, inviteeClient, http.MethodGet, env.ProjectURL+"/projects", devToken, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !projectExists(devProjectsResp, project3) {
		t.Fatalf("expected Dev to see assigned project3, body=%s", string(raw))
	}
	if projectExists(devProjectsResp, project1) {
		t.Fatalf("expected Dev not to see project1, body=%s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodPost, env.ProjectURL+"/projects", devToken, map[string]any{
		"name": "Dev Should Not Create",
	})
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for Dev create project, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodGet, env.AnalyticsURL+"/analytics/usage?group_by=model", devToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for Dev org-scope usage, got %d body=%s", status, string(raw))
	}
	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodGet, env.AnalyticsURL+"/analytics/usage?group_by=model&scope=project&project_id="+project3, devToken, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodGet, env.AnalyticsURL+"/audit?limit=100", devToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for Dev audit access, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodGet, env.AnalyticsURL+"/reports/csv/usage", devToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for Dev csv export, got %d body=%s", status, string(raw))
	}

	status, providerStatusResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.CatalogURL+"/catalog/providers/status?refresh=1", "", nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasProviderStatusFields(providerStatusResp, "mock") {
		t.Fatalf("expected provider status fields for mock provider, body=%s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodDelete, env.AuthURL+"/org/members/"+devUserID, token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, client, http.MethodDelete, env.AuthURL+"/org/members/"+adminUserID, token, nil)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for self-delete attempt, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPut, env.AuthURL+"/org/members/"+adminUserID+"/role", token, map[string]any{
		"role": "PM",
	})
	if status != http.StatusConflict {
		t.Fatalf("expected 409 on last-admin demotion, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodPost, env.AuthURL+"/auth/refresh", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized refresh after member delete, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodPost, env.AuthURL+"/auth/login", "", map[string]any{
		"email":    inviteEmail,
		"password": "password123",
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized login after member delete, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodDelete, env.ProjectURL+"/projects/"+project2, token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, client, http.MethodDelete, env.ProjectURL+"/projects/"+project3, token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, projectsResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if projectExists(projectsResp, project2) {
		t.Fatalf("deleted project still present in list, project_id=%s body=%s", project2, string(raw))
	}

	status, auditAfterDeleteResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/audit?limit=400", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasAuditAction(auditAfterDeleteResp, "member.deleted") || !hasAuditAction(auditAfterDeleteResp, "project.deleted") {
		t.Fatalf("expected member.deleted and project.deleted in audit, got body=%s", string(raw))
	}

	if os.Getenv("VERIFY_KAFKA_EVENTS") != "0" {
		wait := 12 * time.Second
		mustKafkaEvent(t, env.KafkaBrokers, "project.created", wait, func(payload map[string]any) bool {
			return str(payload, "project_id") == project1
		}, "project.created for first project")

		mustKafkaEvent(t, env.KafkaBrokers, "api_key.created", wait, func(payload map[string]any) bool {
			return str(payload, "key_id") == key1ID && str(payload, "project_id") == project1
		}, "api_key.created for first key")

		mustKafkaEvent(t, env.KafkaBrokers, "limit.updated", wait, func(payload map[string]any) bool {
			return str(payload, "scope_type") == "project" && str(payload, "scope_id") == project1
		}, "limit.updated for first project")

		mustKafkaEvent(t, env.KafkaBrokers, "request.accepted", wait, func(payload map[string]any) bool {
			return str(payload, "project_id") == project1 || str(payload, "project_id") == project2 || str(payload, "project_id") == project3
		}, "request.accepted")

		mustKafkaEvent(t, env.KafkaBrokers, "request.rejected", wait, func(payload map[string]any) bool {
			return str(payload, "project_id") == project1 && str(payload, "reason") != ""
		}, "request.rejected")

		mustKafkaEvent(t, env.KafkaBrokers, "api_key.revoked", wait, func(payload map[string]any) bool {
			return str(payload, "key_id") == key1ID
		}, "api_key.revoked")

		mustKafkaEvent(t, env.KafkaBrokers, "request.completed", wait, func(payload map[string]any) bool {
			return str(payload, "project_id") == project2 || str(payload, "project_id") == project3
		}, "request.completed")

		if fallbackObserved {
			mustKafkaEvent(t, env.KafkaBrokers, "fallback.used", wait, func(payload map[string]any) bool {
				return str(payload, "project_id") == project2 && str(payload, "fallback_model") == "mock-fast"
			}, "fallback.used")
		}

		mustKafkaEvent(t, env.KafkaBrokers, "usage.recorded", wait, func(payload map[string]any) bool {
			return str(payload, "project_id") == project2 &&
				str(payload, "api_key_id") == key2ID &&
				str(payload, "requested_model") != "" &&
				str(payload, "effective_model") != "" &&
				str(payload, "model") == str(payload, "effective_model")
		}, "usage.recorded")
	}
}

func extractAssistantContent(resp map[string]any) string {
	choices, ok := resp["choices"].([]any)
	if !ok || len(choices) == 0 {
		return ""
	}
	first, ok := choices[0].(map[string]any)
	if !ok {
		return ""
	}
	msg, ok := first["message"].(map[string]any)
	if !ok {
		return ""
	}
	content, _ := msg["content"].(string)
	return content
}

func hasFallbackLog(resp map[string]any) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if used, ok := m["fallback_used"].(bool); ok && used {
			return true
		}
	}
	return false
}

func hasFallbackLogWithModel(resp map[string]any, model string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		used, _ := m["fallback_used"].(bool)
		fallbackModel, _ := m["fallback_model"].(string)
		if used && fallbackModel == model {
			return true
		}
	}
	return false
}

func hasCompletedLogForModel(resp map[string]any, model string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		status, _ := m["status"].(string)
		gotModel, _ := m["model"].(string)
		if status == "completed" && gotModel == model {
			return true
		}
	}
	return false
}

func hasAuditAction(resp map[string]any, action string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if got, _ := m["action"].(string); got == action {
			return true
		}
	}
	return false
}

func hasPositiveUsage(resp map[string]any) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if v, ok := m["total_tokens"].(float64); ok && v > 0 {
			return true
		}
	}
	return false
}

func hasUsageForModelWithPositiveCost(resp map[string]any, model string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		group, _ := row["group"].(string)
		totalCost, _ := row["total_cost"].(float64)
		if group == model && totalCost > 0 {
			return true
		}
	}
	return false
}

func hasUsageGroup(resp map[string]any, model string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if group, _ := row["group"].(string); group == model {
			return true
		}
	}
	return false
}

func hasKeyID(resp map[string]any, keyID string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := row["id"].(string); id == keyID {
			return true
		}
	}
	return false
}

func hasKeyName(resp map[string]any, keyID, name string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := row["id"].(string); id == keyID {
			if got, _ := row["name"].(string); got == name {
				return true
			}
		}
	}
	return false
}

func hasProviderStatusFields(resp map[string]any, provider string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if got, _ := row["provider"].(string); got != provider {
			continue
		}
		_, hasStatus := row["status"]
		_, hasCheckedAt := row["checked_at"]
		_, hasLatency := row["latency_ms"]
		_, hasError := row["error"]
		return hasStatus && hasCheckedAt && hasLatency && hasError
	}
	return false
}

func projectExists(resp map[string]any, projectID string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := m["id"].(string); id == projectID {
			return true
		}
	}
	return false
}

func hasProjectMember(resp map[string]any, userID string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := m["user_id"].(string); id == userID {
			return true
		}
	}
	return false
}

func mustKafkaEvent(
	t *testing.T,
	brokers string,
	topic string,
	timeout time.Duration,
	match func(map[string]any) bool,
	description string,
) {
	t.Helper()
	if !testutil.WaitKafkaEvent(t, brokers, topic, timeout, match) {
		t.Fatalf("expected kafka event not observed: topic=%s (%s)", topic, description)
	}
}

func str(payload map[string]any, key string) string {
	v, _ := payload[key].(string)
	return v
}
