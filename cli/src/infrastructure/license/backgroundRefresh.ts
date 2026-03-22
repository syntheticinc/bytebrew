import { AuthStorage, type AuthTokens } from '../auth/AuthStorage.js';
import { LicenseStorage } from './LicenseStorage.js';
import { CloudApiClient } from '../api/CloudApiClient.js';

/**
 * Attempts to refresh the license JWT in the background.
 * Non-blocking: errors are silently ignored.
 * Called once at startup of chat/ask commands.
 */
export function startBackgroundLicenseRefresh(): void {
  // Fire and forget — no await
  void refreshLicenseBackground(
    new AuthStorage(),
    new LicenseStorage(),
    (tokens) => new CloudApiClient(tokens),
  ).catch(() => {
    // Silent failure — don't show errors to user
  });
}

export async function refreshLicenseBackground(
  auth: AuthStorage,
  licenseStore: LicenseStorage,
  clientFactory: (tokens: AuthTokens) => CloudApiClient,
): Promise<void> {
  const tokens = auth.load();
  if (!tokens) return; // Not logged in — skip

  const currentJwt = licenseStore.load();
  if (!currentJwt) return; // No license — skip (onboarding will handle)

  const client = clientFactory(tokens);

  const newJwt = await client.refreshLicense(currentJwt);

  // Only save if we got a different JWT
  if (newJwt && newJwt !== currentJwt) {
    licenseStore.save(newJwt);
  }
}
