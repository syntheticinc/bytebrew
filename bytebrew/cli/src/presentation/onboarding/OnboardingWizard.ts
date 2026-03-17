import { AuthStorage } from '../../infrastructure/auth/AuthStorage.js';
import { LicenseStorage } from '../../infrastructure/license/LicenseStorage.js';
import { CloudApiClient, CloudApiError } from '../../infrastructure/api/CloudApiClient.js';
import { prompt, promptPassword } from '../../infrastructure/auth/prompt.js';
import { parseJwtPayload } from '../../infrastructure/license/parseJwt.js';
import { openBrowser } from '../../infrastructure/shell/openBrowser.js';

/**
 * Check if the existing license is expired or expiring soon.
 * Returns 'valid' | 'expiring_soon' | 'grace' | 'expired' | 'missing'.
 * For 'grace', prints a warning with days remaining.
 * For 'expiring_soon', prints a warning about upcoming expiration.
 */
export function checkLicenseStatus(): 'valid' | 'expiring_soon' | 'grace' | 'expired' | 'missing' {
  const jwt = new LicenseStorage().load();
  if (!jwt) return 'missing';

  const claims = parseJwtPayload(jwt);
  if (!claims.exp) return 'valid';

  const now = Math.floor(Date.now() / 1000);

  // Check grace period: exp passed but grace_until not yet
  if (claims.exp < now) {
    const graceUntil = claims.grace_until as number | undefined;
    if (graceUntil && graceUntil > now) {
      const daysLeft = Math.ceil((graceUntil - now) / (60 * 60 * 24));
      console.log('');
      console.log(`Warning: Your subscription expired. Grace period ends in ${daysLeft} day${daysLeft === 1 ? '' : 's'}.`);
      console.log('Renew at https://bytebrew.ai or run "bytebrew status" for details.');
      console.log('');
      return 'grace';
    }
    return 'expired';
  }

  // Pre-expiry warnings: license is still active but expiring soon
  const daysLeft = Math.ceil((claims.exp - now) / (60 * 60 * 24));
  const tier = claims.tier as string | undefined;
  const isTrial = tier === 'trial';
  const threshold = isTrial ? 3 : 7;

  if (daysLeft <= threshold) {
    const label = isTrial ? 'trial' : 'subscription';
    console.log('');
    if (daysLeft <= 1) {
      console.log(`Warning: Your ${label} expires tomorrow.`);
    } else {
      console.log(`Warning: Your ${label} expires in ${daysLeft} days.`);
    }
    console.log('Renew at https://bytebrew.ai or run "bytebrew status" for details.');
    console.log('');
    return 'expiring_soon';
  }

  return 'valid';
}

/**
 * Onboarding wizard — runs when no license.jwt found.
 * Interactive console flow: login or register → activate license.
 * Returns true if license was activated, false if user cancelled.
 */
export async function runOnboardingWizard(): Promise<boolean> {
  console.log('');
  console.log('Welcome to ByteBrew!');
  console.log('');
  console.log('To use ByteBrew you need an account.');
  console.log('');

  const choice = await prompt('  1) Login to existing account\n  2) Register new account\n\nChoice (1/2): ');

  if (choice.trim() === '1') {
    return loginFlow();
  }
  if (choice.trim() === '2') {
    return registerFlow();
  }

  console.log('Invalid choice. Exiting.');
  return false;
}

async function loginFlow(): Promise<boolean> {
  const email = await prompt('Email: ');
  const password = await promptPassword('Password: ');

  const client = new CloudApiClient();
  try {
    const tokens = await client.login(email, password);
    new AuthStorage().save(tokens);
    console.log(`Logged in as ${tokens.email}`);
  } catch (err) {
    if (err instanceof CloudApiError) {
      console.error(`Login failed: ${err.message}`);
    } else {
      console.error(`Login failed: ${(err as Error).message}`);
    }
    return false;
  }

  return activateFlow(client);
}

async function registerFlow(): Promise<boolean> {
  const email = await prompt('Email: ');
  const password = await promptPassword('Password: ');
  const confirm = await promptPassword('Confirm password: ');

  if (password !== confirm) {
    console.error('Passwords do not match.');
    return false;
  }

  const client = new CloudApiClient();
  try {
    const tokens = await client.register(email, password);
    new AuthStorage().save(tokens);
    console.log(`Registered as ${tokens.email}`);
  } catch (err) {
    if (err instanceof CloudApiError) {
      console.error(`Registration failed: ${err.message}`);
    } else {
      console.error(`Registration failed: ${(err as Error).message}`);
    }
    return false;
  }

  // Start trial via Stripe Checkout (CC required)
  return startTrialCheckout(client);
}

async function startTrialCheckout(client: CloudApiClient): Promise<boolean> {
  try {
    const checkoutUrl = await client.createCheckout('personal', 'monthly');
    console.log('');
    console.log('Starting trial... Opening browser for payment setup.');
    console.log('');
    openBrowser(checkoutUrl);
    console.log(`If the browser didn't open, visit: ${checkoutUrl}`);
    console.log('');
    await prompt('Press Enter after completing checkout...');
  } catch {
    // Checkout creation failed — try activate anyway (may work without checkout)
  }

  return activateFlow(client);
}

async function activateFlow(client: CloudApiClient): Promise<boolean> {
  try {
    const jwt = await client.activateLicense();
    new LicenseStorage().save(jwt);

    const claims = parseJwtPayload(jwt);
    console.log('');
    console.log('License activated.');
    console.log(`  Tier: ${claims.tier ?? 'unknown'}`);
    if (claims.exp) {
      const exp = new Date(claims.exp * 1000);
      const days = Math.ceil((exp.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
      console.log(`  Expires: ${exp.toISOString().split('T')[0]} (in ${days} days)`);
    }
    console.log('');
    console.log('Starting ByteBrew...');
    console.log('');
    return true;
  } catch (err) {
    if (err instanceof CloudApiError) {
      console.log(`\nNo active subscription found: ${err.message}`);
      console.log("Subscribe at https://bytebrew.ai and run 'bytebrew activate'.");
    } else {
      console.error(`Activation failed: ${(err as Error).message}`);
    }
    return false;
  }
}
