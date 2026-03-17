import { OnboardingBlock, OnboardingCheckResult, OnboardingContext, OnboardingResult } from '../OnboardingBlock.js';
import { checkLicenseStatus, runOnboardingWizard } from '../OnboardingWizard.js';

export class LicenseBlock implements OnboardingBlock {
  readonly id = 'license';
  readonly displayName = 'License';
  readonly description = 'Login or register to activate your ByteBrew license';
  readonly priority = 0;
  readonly skippable = false;

  check(): OnboardingCheckResult {
    const status = checkLicenseStatus();
    return {
      needsOnboarding: status === 'missing',
      skipCount: 0,
    };
  }

  async run(_context: OnboardingContext): Promise<OnboardingResult> {
    const activated = await runOnboardingWizard();
    if (activated) {
      return { status: 'completed' };
    }
    return { status: 'failed', error: 'License activation cancelled' };
  }
}
