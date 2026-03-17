import { api } from './client';

export interface LicenseStatus {
  tier: string;
  expires_at: string;
  grace_until?: string;
  features: string[];
}

export async function activateLicense(): Promise<{ license: string }> {
  return api.request<{ license: string }>('POST', '/api/v1/license/activate', {});
}

export async function getLicenseStatus(): Promise<LicenseStatus> {
  return api.request<LicenseStatus>('GET', '/api/v1/license/status');
}

export async function downloadLicense(): Promise<Blob> {
  const API_BASE = import.meta.env.VITE_API_URL || '';
  const token = api.getToken();
  const res = await fetch(`${API_BASE}/api/v1/license/download`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) {
    throw new Error('Failed to download license');
  }
  return res.blob();
}

export interface UsageInfo {
  tier: string;
  proxy_steps_used: number;
  proxy_steps_limit: number;
  proxy_steps_remaining: number;
  byok_enabled: boolean;
  current_period_end?: string;
}

export async function getUsage(): Promise<UsageInfo> {
  return api.request<UsageInfo>('GET', '/api/v1/subscription/usage');
}
