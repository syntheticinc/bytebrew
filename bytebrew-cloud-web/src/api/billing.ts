import { api } from './client';

export async function createCheckout(
  plan: string,
  period: string,
): Promise<{ checkout_url: string }> {
  return api.request<{ checkout_url: string }>('POST', '/api/v1/billing/checkout', {
    plan,
    period,
  });
}

export async function createPortal(): Promise<{ portal_url: string }> {
  return api.request<{ portal_url: string }>('POST', '/api/v1/billing/portal', {});
}
