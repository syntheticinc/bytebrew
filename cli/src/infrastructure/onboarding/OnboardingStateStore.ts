import * as fs from 'fs';
import * as path from 'path';
import { ByteBrewHome } from '../config/ByteBrewHome.js';

interface BlockState {
  completedAt?: string; // ISO date
  lastSkippedAt?: string; // ISO date
  skipCount: number;
  reofferAfter?: string; // ISO date
}

interface OnboardingState {
  version: number;
  blocks: Record<string, BlockState>;
  launchCount: number;
}

const DEFAULT_STATE: OnboardingState = {
  version: 1,
  blocks: {},
  launchCount: 0,
};

export class OnboardingStateStore {
  private readonly filePath: string;

  constructor() {
    this.filePath = path.join(ByteBrewHome.dir(), 'onboarding.json');
  }

  load(): OnboardingState {
    try {
      const raw = fs.readFileSync(this.filePath, 'utf-8');
      const parsed = JSON.parse(raw) as OnboardingState;
      return parsed;
    } catch {
      return { ...DEFAULT_STATE, blocks: {} };
    }
  }

  save(state: OnboardingState): void {
    ByteBrewHome.ensureDir();
    fs.writeFileSync(this.filePath, JSON.stringify(state, null, 2), 'utf-8');
  }

  getBlockState(blockId: string): BlockState | undefined {
    const state = this.load();
    return state.blocks[blockId];
  }

  markCompleted(blockId: string): void {
    const state = this.load();
    state.blocks[blockId] = {
      ...state.blocks[blockId],
      completedAt: new Date().toISOString(),
      skipCount: state.blocks[blockId]?.skipCount ?? 0,
    };
    this.save(state);
  }

  markSkipped(blockId: string, reofferDays: number): void {
    const state = this.load();
    const now = new Date();
    const reofferAfter = new Date(now.getTime() + reofferDays * 24 * 60 * 60 * 1000);
    const currentBlock = state.blocks[blockId];
    state.blocks[blockId] = {
      ...currentBlock,
      lastSkippedAt: now.toISOString(),
      skipCount: (currentBlock?.skipCount ?? 0) + 1,
      reofferAfter: reofferAfter.toISOString(),
    };
    this.save(state);
  }

  shouldReoffer(blockId: string): boolean {
    const block = this.getBlockState(blockId);
    if (!block?.reofferAfter) {
      return false;
    }
    return new Date() >= new Date(block.reofferAfter);
  }

  incrementLaunchCount(): void {
    const state = this.load();
    state.launchCount += 1;
    this.save(state);
  }
}
