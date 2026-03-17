import { OnboardingBlock, OnboardingContext, OnboardingResult } from './OnboardingBlock.js';
import { OnboardingStateStore } from '../../infrastructure/onboarding/OnboardingStateStore.js';

export interface OnboardingRunResult {
  completed: string[];   // block IDs that completed
  skipped: string[];     // block IDs that were skipped
  failed: string[];      // block IDs that failed
  canProceed: boolean;   // false if a mandatory block failed
}

export class OnboardingOrchestrator {
  private blocks: OnboardingBlock[] = [];
  private stateStore: OnboardingStateStore;

  constructor(stateStore?: OnboardingStateStore) {
    this.stateStore = stateStore ?? new OnboardingStateStore();
  }

  register(block: OnboardingBlock): void {
    this.blocks.push(block);
  }

  async run(context: OnboardingContext): Promise<OnboardingRunResult> {
    const result: OnboardingRunResult = {
      completed: [],
      skipped: [],
      failed: [],
      canProceed: true,
    };

    // In headless mode, skip all onboarding
    if (context.headless) {
      return result;
    }

    this.stateStore.incrementLaunchCount();

    // Sort by priority (lower first)
    const sorted = [...this.blocks].sort((a, b) => a.priority - b.priority);

    for (const block of sorted) {
      const checkResult = block.check();

      // Block doesn't need onboarding
      if (!checkResult.needsOnboarding) {
        continue;
      }

      // Skippable block was previously skipped — check re-offer timing
      if (block.skippable && checkResult.skipCount > 0) {
        if (!this.stateStore.shouldReoffer(block.id)) {
          continue;
        }
      }

      // Run the block
      const blockResult = await block.run(context);

      switch (blockResult.status) {
        case 'completed':
          this.stateStore.markCompleted(block.id);
          result.completed.push(block.id);
          break;
        case 'skipped':
          this.stateStore.markSkipped(block.id, checkResult.skipCount === 0 ? 7 : 30);
          result.skipped.push(block.id);
          break;
        case 'failed':
          result.failed.push(block.id);
          if (!block.skippable) {
            result.canProceed = false;
            return result;  // Stop processing — mandatory block failed
          }
          break;
      }
    }

    return result;
  }
}
