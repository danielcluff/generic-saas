// Centralized application configuration for frontend
export interface AppConfig {
  // Brand and identity
  appName: string;
  appDisplayName: string;
  companyName: string;
  supportEmail: string;

  // URLs
  appBaseUrl: string;
  dashboardUrl: string;

  // Metadata
  description: string;
  version: string;

  // API configuration
  apiBaseUrl: string;
}

// Load configuration from environment variables with sensible defaults
const config: AppConfig = {
  // Brand and identity - can be overridden with environment variables
  appName: import.meta.env.PUBLIC_APP_NAME || "SaaSPlatform",
  appDisplayName: import.meta.env.PUBLIC_APP_DISPLAY_NAME || "SaaSPlatform",
  companyName: import.meta.env.PUBLIC_COMPANY_NAME || "SaaSPlatform Inc.",
  supportEmail: import.meta.env.PUBLIC_SUPPORT_EMAIL || "support@saasplatform.com",

  // URLs - customizable for different environments
  appBaseUrl: import.meta.env.PUBLIC_APP_BASE_URL || "https://app.saasplatform.com",
  dashboardUrl: import.meta.env.PUBLIC_DASHBOARD_URL || "/dashboard",

  // Metadata
  description: import.meta.env.PUBLIC_APP_DESCRIPTION || "A modern SaaS platform built with cutting-edge technology",
  version: import.meta.env.PUBLIC_APP_VERSION || "1.0.0",

  // API configuration
  apiBaseUrl: import.meta.env.PUBLIC_API_BASE_URL || "http://localhost:8080",
};

export default config;

// Helper functions for common branding needs
export const getBrandingTitle = (suffix?: string): string => {
  return suffix ? `${suffix} - ${config.appDisplayName}` : config.appDisplayName;
};

export const getPageTitle = (pageName: string): string => {
  return `${pageName} - ${config.appDisplayName}`;
};

export const getCopyright = (): string => {
  const year = new Date().getFullYear();
  return `© ${year} ${config.companyName}. All rights reserved.`;
};