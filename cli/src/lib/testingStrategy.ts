import { existsSync, readFileSync } from 'fs';
import { join } from 'path';

const TESTING_STRATEGY_FILE = '.testing-strategy.yaml';

export function readTestingStrategy(projectRoot: string): string | undefined {
  const filePath = join(projectRoot, TESTING_STRATEGY_FILE);
  if (!existsSync(filePath)) {
    return undefined;
  }
  try {
    return readFileSync(filePath, 'utf-8');
  } catch {
    return undefined;
  }
}
