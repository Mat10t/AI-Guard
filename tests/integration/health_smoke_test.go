package integration

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"llm-gateway-mvp/tests/testutil"
)

func TestHealthAndAPISmoke(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to run integration tests")
	}
	env := testutil.LoadEnv()
	client := testutil.NewClient()

	healthURLs := []string{
		env.GatewayURL + "/healthz",
		env.AuthURL + "/healthz",
		env.ProjectURL + "/healthz",
		env.LimitsURL + "/healthz",
		env.CatalogURL + "/healthz",
		env.AnalyticsURL + "/healthz",
	}
	for _, u := range healthURLs {
		resp, err := client.Get(u)
		if err != nil {
			t.Fatalf("health check failed for %s: %v", u, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unexpected health status for %s: %d", u, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}

	status, modelsResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.CatalogURL+"/catalog/models", "", nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasCatalogModel(modelsResp, "gpt-5.4-mini") || !hasCatalogModel(modelsResp, "gpt-5.4") || !hasCatalogModel(modelsResp, "gemini-2.5-flash") || !hasCatalogModel(modelsResp, "mock-fast") {
		t.Fatalf("catalog models mismatch, expected gpt-5.4-mini/gpt-5.4/gemini-2.5-flash/mock-fast, got body=%s", string(raw))
	}
	if !hasCatalogPricingFields(modelsResp, "gpt-5.4-mini") {
		t.Fatalf("expected pricing_source/pricing_updated_at in catalog models response, got body=%s", string(raw))
	}

	status, providerStatusResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.CatalogURL+"/catalog/providers/status?refresh=1", "", nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasProviderStatusFields(providerStatusResp, "mock") {
		t.Fatalf("expected provider live-status fields, got body=%s", string(raw))
	}

	email := testutil.RandomEmail("int")
	password := "password123"

	status, reg, raw := testutil.RequestJSON(t, client, http.MethodPost, env.AuthURL+"/auth/register", "", map[string]any{
		"org_name": "Integration Org",
		"email":    email,
		"password": password,
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	token := testutil.MustString(t, reg, "access_token")
	adminUserID := testutil.MustString(t, reg, "user_id")

	status, login, raw := testutil.RequestJSON(t, client, http.MethodPost, env.AuthURL+"/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	_ = testutil.MustString(t, login, "access_token")

	status, pricingResp, raw := testutil.RequestJSON(t, client, http.MethodPut, env.CatalogURL+"/catalog/models/gpt-5.4-mini/pricing", token, map[string]any{
		"input_cost":     0.00021,
		"output_cost":    0.00081,
		"pricing_source": "manual-smoke",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if got, _ := pricingResp["pricing_source"].(string); got != "manual-smoke" {
		t.Fatalf("expected pricing_source=manual-smoke after update, got body=%s", string(raw))
	}

	status, inviteProjectResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects", token, map[string]any{
		"name": "Integration Invite Scope Project",
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	inviteProjectID := testutil.MustString(t, inviteProjectResp, "id")

	inviteEmail := testutil.RandomEmail("invite-dev")
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
		"project_ids": []string{inviteProjectID},
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	inviteToken := testutil.MustString(t, inviteResp, "invitation_token")

	inviteeClient := testutil.NewClient()
	inviteePassword := "password123"
	status, acceptResp, raw := testutil.RequestJSON(t, inviteeClient, http.MethodPost, env.AuthURL+"/org/members/accept", "", map[string]any{
		"token":    inviteToken,
		"password": inviteePassword,
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	devToken := testutil.MustString(t, acceptResp, "access_token")
	devUserID := testutil.MustString(t, acceptResp, "user_id")
	if role := testutil.MustString(t, acceptResp, "role"); role != "Dev" {
		t.Fatalf("expected invited role=Dev, got %q body=%s", role, string(raw))
	}

	status, refreshedDev, raw := testutil.RequestJSON(t, inviteeClient, http.MethodPost, env.AuthURL+"/auth/refresh", "", nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	_ = testutil.MustString(t, refreshedDev, "access_token")

	status, projectMembersResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects/"+inviteProjectID+"/members", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasProjectMember(projectMembersResp, devUserID) {
		t.Fatalf("expected invited Dev membership in assigned project, body=%s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodPost, env.AuthURL+"/org/members/accept", "", map[string]any{
		"token":    "invalid-token",
		"password": "password123",
	})
	if status != http.StatusNotFound {
		t.Fatalf("expected not_found for invalid invite token, got %d body=%s", status, string(raw))
	}

	status, devProjectsResp, raw := testutil.RequestJSON(t, inviteeClient, http.MethodGet, env.ProjectURL+"/projects", devToken, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !projectExists(devProjectsResp, inviteProjectID) {
		t.Fatalf("expected Dev to see assigned project, body=%s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodPost, env.ProjectURL+"/projects", devToken, map[string]any{
		"name": "Dev Cannot Create",
	})
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for Dev project create, got %d body=%s", status, string(raw))
	}
	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodPut, env.ProjectURL+"/projects/00000000-0000-0000-0000-000000000000/routing", devToken, map[string]any{
		"fallback_model_id": "mock-fast",
	})
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for Dev project routing update, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodGet, env.AnalyticsURL+"/analytics/usage?group_by=model", devToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for Dev org-scope usage, got %d body=%s", status, string(raw))
	}
	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodGet, env.AnalyticsURL+"/analytics/usage?group_by=model&scope=project&project_id="+inviteProjectID, devToken, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, inviteeClient, http.MethodGet, env.AnalyticsURL+"/audit?limit=20", devToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403 for Dev audit access, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodDelete, env.AuthURL+"/org/members/"+devUserID, token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, client, http.MethodDelete, env.AuthURL+"/org/members/"+adminUserID, token, nil)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 on self-delete, got %d body=%s", status, string(raw))
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
		"password": inviteePassword,
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized login after member delete, got %d body=%s", status, string(raw))
	}

	status, reinviteResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.AuthURL+"/org/members/invite", token, map[string]any{
		"email": inviteEmail,
		"role":  "Finance",
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	reinviteToken := testutil.MustString(t, reinviteResp, "invitation_token")

	reinviteClient := testutil.NewClient()
	status, reinviteAcceptResp, raw := testutil.RequestJSON(t, reinviteClient, http.MethodPost, env.AuthURL+"/org/members/accept", "", map[string]any{
		"token":    reinviteToken,
		"password": "password123",
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	if role := testutil.MustString(t, reinviteAcceptResp, "role"); role != "Finance" {
		t.Fatalf("expected re-invited role=Finance, got %q body=%s", role, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.AuthURL+"/auth/refresh", "", nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.AuthURL+"/auth/logout", "", nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.AuthURL+"/auth/refresh", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized after logout refresh, got %d body=%s", status, string(raw))
	}

	status, prj, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects", token, map[string]any{
		"name": "Integration Project",
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	projectID := testutil.MustString(t, prj, "id")

	status, routingResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects/"+projectID+"/routing", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if v, ok := routingResp["fallback_model_id"]; ok && v != nil {
		t.Fatalf("expected empty fallback_model_id by default, got body=%s", string(raw))
	}

	status, routingResp, raw = testutil.RequestJSON(t, client, http.MethodPut, env.ProjectURL+"/projects/"+projectID+"/routing", token, map[string]any{
		"fallback_model_id": "mock-fast",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if got, _ := routingResp["fallback_model_id"].(string); got != "mock-fast" {
		t.Fatalf("expected fallback_model_id=mock-fast, got body=%s", string(raw))
	}

	status, routingResp, raw = testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects/"+projectID+"/routing", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if got, _ := routingResp["fallback_model_id"].(string); got != "mock-fast" {
		t.Fatalf("expected stored fallback_model_id=mock-fast, got body=%s", string(raw))
	}

	status, keyResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+projectID+"/keys", token, map[string]any{
		"name": "Integration primary key",
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	apiKey := testutil.MustString(t, keyResp, "api_key")
	keyID := testutil.MustString(t, keyResp, "id")
	if gotName := testutil.MustString(t, keyResp, "name"); gotName != "Integration primary key" {
		t.Fatalf("expected key name to be preserved, got body=%s", string(raw))
	}

	status, secondKeyResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+projectID+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	secondAPIKey := testutil.MustString(t, secondKeyResp, "api_key")
	secondKeyID := testutil.MustString(t, secondKeyResp, "id")
	if gotName := testutil.MustString(t, secondKeyResp, "name"); !strings.HasPrefix(gotName, "Key ") {
		t.Fatalf("expected default generated key name, got body=%s", string(raw))
	}

	status, compareProjectResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects", token, map[string]any{
		"name": "Integration Fallback Compare",
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	compareProjectID := testutil.MustString(t, compareProjectResp, "id")

	status, compareKeyResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+compareProjectID+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	compareAPIKey := testutil.MustString(t, compareKeyResp, "api_key")

	status, keyInfo, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects/"+projectID+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasKeyID(keyInfo, keyID) || !hasKeyID(keyInfo, secondKeyID) {
		t.Fatalf("expected both keys in list response, body=%s", string(raw))
	}
	if !hasKeyName(keyInfo, keyID, "Integration primary key") {
		t.Fatalf("expected named key in list response, body=%s", string(raw))
	}
	if !hasKeyAPIValue(keyInfo, keyID, apiKey) {
		t.Fatalf("expected active key to include api_key in list response, body=%s", string(raw))
	}
	if !hasKeyAPIValue(keyInfo, secondKeyID, secondAPIKey) {
		t.Fatalf("expected second active key to include api_key in list response, body=%s", string(raw))
	}

	status, routeOverrideResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey, map[string]any{
		"model":    "gpt-5.4-mini",
		"messages": []map[string]any{{"role": "user", "content": "usage pricing check"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, routeDefaultResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", compareAPIKey, map[string]any{
		"model":    "gpt-5.4-mini",
		"messages": []map[string]any{{"role": "user", "content": "routing compare request"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	overrideContent := extractAssistantContent(routeOverrideResp)
	defaultContent := extractAssistantContent(routeDefaultResp)
	if strings.Contains(overrideContent, "mock-fallback") && strings.Contains(defaultContent, "mock-fallback") {
		if modelA, _ := routeOverrideResp["model"].(string); modelA != "mock-fast" {
			t.Fatalf("expected override fallback model=mock-fast, got body=%s", string(raw))
		}
		if modelB, _ := routeDefaultResp["model"].(string); modelB != "gpt-5.4-mini" {
			t.Fatalf("expected default fallback to keep requested model, got body=%s", string(raw))
		}
	}

	status, autosyncProjectResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects", token, map[string]any{
		"name": "Integration Autosync",
	})
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	autosyncProjectID := testutil.MustString(t, autosyncProjectResp, "id")

	status, autosyncKeyResp, raw := testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+autosyncProjectID+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusCreated, raw)
	autosyncAPIKey := testutil.MustString(t, autosyncKeyResp, "api_key")

	status, autosyncLimitTokensResp, raw := testutil.RequestJSON(t, client, http.MethodPut, env.LimitsURL+"/limits/projects/"+autosyncProjectID, token, map[string]any{
		"token_limit":   10,
		"billing_model": "gpt-5.4-mini",
		"period":        "day",
		"sync_source":   "tokens",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if syncSource, _ := autosyncLimitTokensResp["sync_source"].(string); syncSource != "tokens" {
		t.Fatalf("expected sync_source=tokens, got body=%s", string(raw))
	}
	budgetBeforeAutosync := asFloat(autosyncLimitTokensResp["budget_limit_usd"])

	status, autosyncPricing1Resp, raw := testutil.RequestJSON(t, client, http.MethodPut, env.CatalogURL+"/catalog/models/gpt-5.4-mini/pricing", token, map[string]any{
		"input_cost":     0.00040,
		"output_cost":    0.00120,
		"pricing_source": "manual-autosync-1",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if affected := asFloat(autosyncPricing1Resp["affected_project_limits"]); int(affected) < 1 {
		t.Fatalf("expected affected_project_limits >= 1, got body=%s", string(raw))
	}

	status, autosyncLimitAfterPricing1, raw := testutil.RequestJSON(t, client, http.MethodGet, env.LimitsURL+"/limits/projects/"+autosyncProjectID, token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if syncSource, _ := autosyncLimitAfterPricing1["sync_source"].(string); syncSource != "tokens" {
		t.Fatalf("expected sync_source=tokens after autosync, got body=%s", string(raw))
	}
	if tokenLimitAfterPricing1 := asFloat(autosyncLimitAfterPricing1["token_limit"]); int(tokenLimitAfterPricing1) != 10 {
		t.Fatalf("expected token_limit=10 after tokens autosync, got body=%s", string(raw))
	}
	budgetAfterPricing1 := asFloat(autosyncLimitAfterPricing1["budget_limit_usd"])
	if approxEqualFloat(budgetAfterPricing1, budgetBeforeAutosync, 1e-12) {
		t.Fatalf("expected budget_limit_usd to change after pricing autosync, got body=%s", string(raw))
	}

	status, autosyncLimitBudgetResp, raw := testutil.RequestJSON(t, client, http.MethodPut, env.LimitsURL+"/limits/projects/"+autosyncProjectID, token, map[string]any{
		"budget_limit_usd": budgetAfterPricing1,
		"billing_model":    "gpt-5.4-mini",
		"period":           "day",
		"sync_source":      "budget",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if syncSource, _ := autosyncLimitBudgetResp["sync_source"].(string); syncSource != "budget" {
		t.Fatalf("expected sync_source=budget, got body=%s", string(raw))
	}
	budgetPinned := asFloat(autosyncLimitBudgetResp["budget_limit_usd"])
	tokenBeforePricing2 := asFloat(autosyncLimitBudgetResp["token_limit"])

	status, autosyncPricing2Resp, raw := testutil.RequestJSON(t, client, http.MethodPut, env.CatalogURL+"/catalog/models/gpt-5.4-mini/pricing", token, map[string]any{
		"input_cost":     0.00600,
		"output_cost":    0.00600,
		"pricing_source": "manual-autosync-2",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if affected := asFloat(autosyncPricing2Resp["affected_project_limits"]); int(affected) < 1 {
		t.Fatalf("expected affected_project_limits >= 1 on second pricing update, got body=%s", string(raw))
	}

	status, autosyncLimitAfterPricing2, raw := testutil.RequestJSON(t, client, http.MethodGet, env.LimitsURL+"/limits/projects/"+autosyncProjectID, token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if syncSource, _ := autosyncLimitAfterPricing2["sync_source"].(string); syncSource != "budget" {
		t.Fatalf("expected sync_source=budget after second autosync, got body=%s", string(raw))
	}
	tokenAfterPricing2 := asFloat(autosyncLimitAfterPricing2["token_limit"])
	budgetAfterPricing2 := asFloat(autosyncLimitAfterPricing2["budget_limit_usd"])
	if int(tokenAfterPricing2) == int(tokenBeforePricing2) {
		t.Fatalf("expected token_limit to change for budget autosync, got body=%s", string(raw))
	}
	if !approxEqualFloat(budgetPinned, budgetAfterPricing2, 1e-12) {
		t.Fatalf("expected pinned budget_limit_usd for budget autosync, got body=%s", string(raw))
	}
	if int(tokenAfterPricing2) != 1 {
		t.Fatalf("expected token_limit=1 after second autosync for runtime check, got body=%s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", autosyncAPIKey, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "a"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", autosyncAPIKey, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "b"}},
	})
	if status != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after autosync-updated runtime token_limit, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", secondAPIKey, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "second key request"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, usageByModelResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/analytics/usage?group_by=model", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasUsageForModelWithPositiveCost(usageByModelResp, "gpt-5.4-mini") {
		t.Fatalf("expected positive cost usage for model=gpt-5.4-mini, got body=%s", string(raw))
	}
	if !hasUsageGroup(usageByModelResp, "mock-fast") {
		t.Fatalf("expected usage grouped by effective fallback model=mock-fast, got body=%s", string(raw))
	}

	status, limitByTokensResp, raw := testutil.RequestJSON(t, client, http.MethodPut, env.LimitsURL+"/limits/projects/"+projectID, token, map[string]any{
		"token_limit":   10,
		"billing_model": "gpt-5.4-mini",
		"period":        "day",
		"sync_source":   "tokens",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	usdPerToken, ok := limitByTokensResp["usd_per_token"].(float64)
	if !ok || usdPerToken <= 0 {
		t.Fatalf("expected usd_per_token in response, got body=%s", string(raw))
	}
	if _, ok = limitByTokensResp["budget_limit_usd"].(float64); !ok {
		t.Fatalf("expected budget_limit_usd in response, got body=%s", string(raw))
	}

	status, getLimitResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.LimitsURL+"/limits/projects/"+projectID, token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if gotModel, _ := getLimitResp["billing_model"].(string); gotModel != "gpt-5.4-mini" {
		t.Fatalf("expected billing_model=gpt-5.4-mini, got %q body=%s", gotModel, string(raw))
	}

	status, limitByBudgetResp, raw := testutil.RequestJSON(t, client, http.MethodPut, env.LimitsURL+"/limits/projects/"+projectID, token, map[string]any{
		"budget_limit_usd": usdPerToken,
		"billing_model":    "gpt-5.4-mini",
		"period":           "day",
		"sync_source":      "budget",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	tokenLimitFromBudget, ok := limitByBudgetResp["token_limit"].(float64)
	if !ok || int(tokenLimitFromBudget) != 1 {
		t.Fatalf("expected token_limit=1 after budget sync, got body=%s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPut, env.LimitsURL+"/limits/projects/"+projectID, token, map[string]any{
		"budget_limit_usd": 1,
		"billing_model":    "mock-fast",
		"period":           "day",
		"sync_source":      "budget",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for budget sync on mock-fast, got %d body=%s", status, string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "a"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "this will exceed project limit"}},
	})
	if status != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d body=%s", status, string(raw))
	}
	if !strings.Contains(string(raw), "limit") {
		t.Fatalf("expected limit reason in body: %s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.ProjectURL+"/projects/"+projectID+"/keys/"+keyID+"/revoke", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	status, keyInfoAfterRevoke, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects/"+projectID+"/keys", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasKeyStatus(keyInfoAfterRevoke, keyID, "revoked") {
		t.Fatalf("expected revoked status in key list, body=%s", string(raw))
	}
	if !hasKeyAPINull(keyInfoAfterRevoke, keyID) {
		t.Fatalf("expected revoked key api_key=null in list response, body=%s", string(raw))
	}
	if !hasKeyAPIValue(keyInfoAfterRevoke, secondKeyID, secondAPIKey) {
		t.Fatalf("expected second key to remain readable in list response, body=%s", string(raw))
	}

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", apiKey, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "test after revoke"}},
	})
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for revoked key, got %d body=%s", status, string(raw))
	}
	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", secondAPIKey, map[string]any{
		"model":    "mock-fast",
		"messages": []map[string]any{{"role": "user", "content": "other key must stay valid"}},
	})
	if status != http.StatusTooManyRequests {
		t.Fatalf("expected 429 from project limit for second key after first revoke, got %d body=%s", status, string(raw))
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

	status, scopedAuditProject, raw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/audit?scope=project&project_id="+projectID+"&limit=200", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasAuditAction(scopedAuditProject, "project.created") {
		t.Fatalf("expected project-scoped audit to include project.created, body=%s", string(raw))
	}

	status, scopedAuditKey, raw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/audit?scope=key&project_id="+projectID+"&api_key_id="+keyID+"&limit=200", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	if !hasAuditAction(scopedAuditKey, "api_key.created") {
		t.Fatalf("expected key-scoped audit to include api_key.created, body=%s", string(raw))
	}
	if hasAuditAction(scopedAuditKey, "project.created") {
		t.Fatalf("key-scoped audit must not include project.created, body=%s", string(raw))
	}

	status, _, usageCSVRaw := testutil.RequestJSON(t, client, http.MethodGet, env.AnalyticsURL+"/reports/csv/usage", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, usageCSVRaw)
	if !strings.Contains(string(usageCSVRaw), "day,project_id,model,requested_model,effective_model,total_tokens,total_cost") {
		t.Fatalf("unexpected usage csv header: %s", string(usageCSVRaw))
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

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPut, env.LimitsURL+"/limits/projects/"+projectID, token, map[string]any{
		"token_limit":   5000,
		"billing_model": "gpt-5.4-mini",
		"period":        "day",
		"sync_source":   "tokens",
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, client, http.MethodPost, env.GatewayURL+"/v1/chat/completions", secondAPIKey, map[string]any{
		"model":    "gemini-2.5-flash",
		"messages": []map[string]any{{"role": "user", "content": "gemini fallback check"}},
	})
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, _, raw = testutil.RequestJSON(t, client, http.MethodDelete, env.ProjectURL+"/projects/"+projectID, token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)

	status, projectsResp, raw := testutil.RequestJSON(t, client, http.MethodGet, env.ProjectURL+"/projects", token, nil)
	testutil.RequireStatus(t, status, http.StatusOK, raw)
	items, _ := projectsResp["items"].([]any)
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if rowID, _ := row["id"].(string); rowID == projectID {
			t.Fatalf("deleted project must not be present in list, project_id=%s body=%s", projectID, string(raw))
		}
	}
}

func hasCatalogModel(resp map[string]any, modelID string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := row["id"].(string); id == modelID {
			return true
		}
	}
	return false
}

func hasCatalogPricingFields(resp map[string]any, modelID string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := row["id"].(string); id != modelID {
			continue
		}
		_, hasSource := row["pricing_source"]
		_, hasUpdated := row["pricing_updated_at"]
		return hasSource && hasUpdated
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

func hasUsageForModelWithPositiveCost(resp map[string]any, modelID string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if group, _ := row["group"].(string); group == modelID {
			if cost, ok := row["total_cost"].(float64); ok && cost > 0 {
				return true
			}
		}
	}
	return false
}

func hasUsageGroup(resp map[string]any, modelID string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if group, _ := row["group"].(string); group == modelID {
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
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if got, _ := row["action"].(string); got == action {
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

func hasKeyStatus(resp map[string]any, keyID, status string) bool {
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
			if got, _ := row["status"].(string); got == status {
				return true
			}
		}
	}
	return false
}

func hasKeyAPIValue(resp map[string]any, keyID, expected string) bool {
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
			got, _ := row["api_key"].(string)
			return got == expected
		}
	}
	return false
}

func hasKeyAPINull(resp map[string]any, keyID string) bool {
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
			v, exists := row["api_key"]
			return exists && v == nil
		}
	}
	return false
}

func projectExists(resp map[string]any, projectID string) bool {
	items, ok := resp["items"].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := row["id"].(string); id == projectID {
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
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if got, _ := row["user_id"].(string); got == userID {
			return true
		}
	}
	return false
}

func approxEqualFloat(a, b, eps float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= eps
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	default:
		return 0
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
