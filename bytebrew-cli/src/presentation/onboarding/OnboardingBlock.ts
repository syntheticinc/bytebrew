export interface OnboardingCheckResult {
  needsOnboarding: boolean;
  lastSkippedAt?: Date;
  skipCount: number;
}

export type OnboardingResult =
  | { status: 'completed' }
  | { status: 'skipped' }
  | { status: 'failed'; error: string };

export interface OnboardingContext {
  serverAddress?: string;
  headless: boolean;
}

export interface OnboardingBlock {
  readonly id: string;
  readonly displayName: string;
  readonly description: string;
  readonly priority: number; // lower = runs first
  readonly skippable: boolean;

  check(): OnboardingCheckResult;
  run(context: OnboardingContext): Promise<OnboardingResult>;
}
