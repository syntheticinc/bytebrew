const API_BASE = import.meta.env.VITE_API_URL || '';

export interface PriceDetail {
  price_id: string;
  amount: number;   // in smallest currency unit (cents)
  currency: string; // e.g. "usd"
  interval: string; // "month" or "year"
}

export interface PlanPricing {
  monthly?: PriceDetail;
  annual?: PriceDetail;
}

export type PricingData = Record<string, PlanPricing>;

/**
 * Fetches current pricing from the API.
 * This is a public endpoint — no auth required.
 */
export async function getPricing(): Promise<PricingData> {
  const res = await fetch(`${API_BASE}/api/v1/pricing`);

  if (!res.ok) {
    throw new Error(`Failed to fetch pricing: ${res.status}`);
  }

  const json = await res.json();
  return json.data as PricingData;
}

/**
 * Formats a price amount (in cents) to a display string.
 * Example: formatPrice(49900, "usd") => "$499"
 * Example: formatPrice(4990, "usd") => "$49.90"
 */
export function formatPrice(amount: number, currency: string): string {
  const dollars = amount / 100;

  const formatter = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
    minimumFractionDigits: dollars === Math.floor(dollars) ? 0 : 2,
    maximumFractionDigits: 2,
  });

  return formatter.format(dollars);
}

/**
 * Returns a formatted price string with interval suffix.
 * Example: formatPriceWithInterval(49900, "usd", "month") => "$499/mo"
 */
export function formatPriceWithInterval(
  amount: number,
  currency: string,
  interval: string,
): string {
  const formatted = formatPrice(amount, currency);
  const suffix = interval === 'month' ? '/mo' : interval === 'year' ? '/yr' : '';
  return `${formatted}${suffix}`;
}
