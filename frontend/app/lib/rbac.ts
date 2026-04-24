export type AppRole = "Admin" | "PM" | "Dev" | "Finance";

export type UICapabilities = {
  showAPIKeys: boolean;
  showProjects: boolean;
  showAnalytics: boolean;
  showOrganization: boolean;
  canUseOrgScope: boolean;
  canInviteMembers: boolean;
  canMutateProjects: boolean;
  canUpdatePricing: boolean;
  canViewAudit: boolean;
  canExportCSV: boolean;
  canViewTechnicalLogs: boolean;
  canViewUsage: boolean;
};

function baseCaps(): UICapabilities {
  return {
    showAPIKeys: false,
    showProjects: false,
    showAnalytics: false,
    showOrganization: false,
    canUseOrgScope: false,
    canInviteMembers: false,
    canMutateProjects: false,
    canUpdatePricing: false,
    canViewAudit: false,
    canExportCSV: false,
    canViewTechnicalLogs: false,
    canViewUsage: false
  };
}

export function normalizeRole(role?: string): AppRole | "" {
  if (role === "Admin" || role === "PM" || role === "Dev" || role === "Finance") {
    return role;
  }
  return "";
}

export function capabilitiesForRole(role?: string): UICapabilities {
  const normalized = normalizeRole(role);
  const caps = baseCaps();

  switch (normalized) {
    case "Admin":
      return {
        ...caps,
        showAPIKeys: true,
        showProjects: true,
        showAnalytics: true,
        showOrganization: true,
        canUseOrgScope: true,
        canInviteMembers: true,
        canMutateProjects: true,
        canUpdatePricing: true,
        canViewAudit: true,
        canExportCSV: true,
        canViewTechnicalLogs: true,
        canViewUsage: true
      };
    case "PM":
      return {
        ...caps,
        showAPIKeys: true,
        showProjects: true,
        showAnalytics: true,
        showOrganization: true,
        canUseOrgScope: false,
        canInviteMembers: true,
        canMutateProjects: true,
        canUpdatePricing: false,
        canViewAudit: true,
        canExportCSV: true,
        canViewTechnicalLogs: true,
        canViewUsage: true
      };
    case "Dev":
      return {
        ...caps,
        showAPIKeys: true,
        showProjects: true,
        showAnalytics: true,
        canUseOrgScope: false,
        canUpdatePricing: false,
        canViewTechnicalLogs: true,
        canViewUsage: true
      };
    case "Finance":
      return {
        ...caps,
        showAnalytics: true,
        showOrganization: true,
        canUseOrgScope: true,
        canUpdatePricing: false,
        canViewAudit: true,
        canExportCSV: true,
        canViewTechnicalLogs: true,
        canViewUsage: true
      };
    default:
      return caps;
  }
}

export function firstAllowedRoute(role?: string): string {
  const caps = capabilitiesForRole(role);
  if (caps.showAPIKeys) {
    return "/api-keys";
  }
  if (caps.showProjects) {
    return "/projects";
  }
  if (caps.showAnalytics) {
    return "/analytics";
  }
  if (caps.showOrganization) {
    return "/auth";
  }
  return "/auth";
}
