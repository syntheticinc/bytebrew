// CLI setup with Commander
import { Command, Option } from 'commander';
import React from 'react';
import { render } from 'ink';
import path from 'path';
import * as fs from 'fs';
import { v4 as uuidv4 } from 'uuid';
import { App } from './presentation/app/App.js';
import { loadConfig, loadAndValidateConfig, AppConfig } from './config/index.js';
import { Indexer, IndexProgress } from './indexing/indexer.js';
import { initLogger } from './lib/logger.js';
import { initOutputWriter, stopOutputWriter } from './lib/outputWriter.js';
import { runHeadless, runHeadlessInteractive } from './headless/index.js';
import { SessionStore } from './infrastructure/persistence/SessionStore.js';
import { AuthStorage } from './infrastructure/auth/AuthStorage.js';
import { LicenseStorage } from './infrastructure/license/LicenseStorage.js';
import { CloudApiClient, CloudApiError } from './infrastructure/api/CloudApiClient.js';
import { parseJwtPayload, showLicenseInfo } from './infrastructure/license/parseJwt.js';
import { prompt, promptPassword } from './infrastructure/auth/prompt.js';
import { startBackgroundLicenseRefresh } from './infrastructure/license/backgroundRefresh.js';
import { runOnboardingWizard, checkLicenseStatus } from './presentation/onboarding/OnboardingWizard.js';
import { openBrowser } from './infrastructure/shell/openBrowser.js';
import { ServerConnectionOrchestrator, ServerConnection } from './infrastructure/server/ServerConnectionOrchestrator.js';
import { UpdateChecker } from './infrastructure/server/UpdateChecker.js';
import { UpdateApplier, isDevMode } from './infrastructure/server/UpdateApplier.js';
import { UpdateDownloader } from './infrastructure/server/UpdateDownloader.js';
import { VERSION } from './version.js';

const program = new Command();

/**
 * Resolve session ID based on CLI flags
 * --new: create new session
 * --resume <id>: use specific session
 * default: resume last session or create new
 */
function resolveSessionId(opts: { new?: boolean; resume?: string }, projectRoot: string): string {
  const store = new SessionStore(projectRoot);

  let sessionId: string;
  if (opts.new) {
    // --new flag: create new session
    sessionId = uuidv4();
  } else if (opts.resume) {
    // --resume <id> flag: use specific session
    sessionId = opts.resume;
  } else {
    // Default: resume last session or create new
    sessionId = store.getLastSessionId() ?? uuidv4();
  }

  // Save session ID for next time
  store.saveSessionId(sessionId);
  return sessionId;
}

interface CommandOptions {
  server?: string;
  project: string;
  debug: boolean;
}

/**
 * Connect to server and build AppConfig.
 * Shared by chat, ask, and session commands.
 */
async function connectAndConfigure(
  options: CommandOptions,
  globalOpts: { directory?: string; new?: boolean; resume?: string },
): Promise<{ config: AppConfig; connection: ServerConnection }> {
  const projectRoot = path.resolve(globalOpts.directory || process.cwd());
  const sessionId = resolveSessionId(globalOpts, projectRoot);

  const externalAddress = options.server || process.env.BYTEBREW_SERVER;
  const orchestrator = new ServerConnectionOrchestrator();
  const connection = await orchestrator.connect(externalAddress || undefined);

  const config = loadAndValidateConfig({
    serverAddress: connection.address,
    projectKey: options.project,
    projectRoot,
    sessionId,
    debug: options.debug,
  });

  return { config, connection };
}

/**
 * Check license status and run onboarding if needed.
 * Exits process if license is expired or user cancels onboarding.
 */
async function enforceLicense(connection: ServerConnection): Promise<void> {
  const licenseStatus = checkLicenseStatus();

  if (licenseStatus === 'missing') {
    const activated = await runOnboardingWizard();
    if (!activated) {
      await connection.cleanup();
      process.exit(0);
    }
    return;
  }

  if (licenseStatus === 'expired') {
    console.log('');
    console.log('Your subscription has expired.');
    console.log('Renew at https://app.bytebrew.ai or run "bytebrew login".');
    console.log('');
    await connection.cleanup();
    process.exit(1);
  }
}

/**
 * Register process exit handler to stop managed server.
 */
function registerExitCleanup(connection: ServerConnection): void {
  const cleanup = connection.cleanup;
  process.on('exit', () => { cleanup(); });
}

program
  .name('bytebrew')
  .description('ByteBrew CLI - AI-powered terminal client')
  .version(VERSION)
  .option('-C, --directory <path>', 'Run as if started in <path> (like git -C)')
  .addOption(new Option('--new', 'Start a new session (ignore previous)').conflicts('resume'))
  .addOption(new Option('--resume <id>', 'Resume specific session by ID').conflicts('new'));

// Chat command (default)
program
  .command('chat', { isDefault: true })
  .description('Start interactive chat session')
  .option('-s, --server <address>', 'Server address (connect to external server)')
  .option('-p, --project <key>', 'Project key', 'default')
  .option('-d, --debug', 'Enable debug mode', false)
  .action(async (options) => {
    const globalOpts = program.opts();
    initLogger(options.debug);

    // Apply pending update if available
    if (!process.env.BYTEBREW_DISABLE_AUTOUPDATE) {
      try {
        const applier = new UpdateApplier();
        await applier.cleanupOldFiles();
        const result = await applier.applyPending();
        if (result.applied) {
          console.log(`Update v${result.version} applied. Will take effect on next server start.`);
        }
      } catch (err) {
        // Don't block startup on update errors — log only in debug
        if (options.debug) {
          console.error(`Auto-update error: ${(err as Error).message}`);
        }
      }
    }

    let connection: ServerConnection | null = null;

    try {
      const result = await connectAndConfigure(options, globalOpts);
      connection = result.connection;

      await enforceLicense(connection);
      startBackgroundLicenseRefresh();

      // Background: check for updates, download silently
      if (!process.env.BYTEBREW_DISABLE_AUTOUPDATE) {
        void (async () => {
          try {
            const checker = new UpdateChecker(VERSION);
            const info = await checker.check();
            if (!info?.updateAvailable) return;

            const downloader = new UpdateDownloader();
            await downloader.download(info);
            // Staged silently — will apply on next launch via applyPending()
          } catch {
            // Non-blocking — don't disturb the user
          }
        })();
      }

      registerExitCleanup(connection);
      runChatApp(result.config);
    } catch (error) {
      console.error((error as Error).message);
      if (connection) await connection.cleanup();
      process.exit(1);
    }
  });

// Ask command - single question (interactive or headless mode)
program
  .command('ask <question>')
  .description('Start chat with an initial question')
  .option('-s, --server <address>', 'Server address (connect to external server)')
  .option('-p, --project <key>', 'Project key', 'default')
  .option('-d, --debug', 'Enable debug mode', false)
  .option('--headless', 'Run in headless mode (no UI, plain text output)', false)
  .option('-o, --output <file>', 'Write output to file (headless mode only)')
  .option(
    '--unknown-cmd <behavior>',
    'How to handle unknown commands in headless mode: block, allow-once, allow-remember',
    'block'
  )
  .action(async (question, options) => {
    // Validate empty question
    if (!question || !question.trim()) {
      console.error('Error: Question cannot be empty');
      process.exit(1);
    }

    const globalOpts = program.opts();
    initLogger(options.debug);

    let connection: ServerConnection | null = null;

    try {
      const result = await connectAndConfigure(options, globalOpts);
      connection = result.connection;

      await enforceLicense(connection);
      startBackgroundLicenseRefresh();
      registerExitCleanup(connection);

      if (options.headless) {
        // Initialize output writer only in headless mode
        if (options.output) {
          initOutputWriter(options.output);
        }

        // Validate unknown-cmd option
        const validBehaviors = ['block', 'allow-once', 'allow-remember'];
        const unknownCmdBehavior = options.unknownCmd || 'block';
        if (!validBehaviors.includes(unknownCmdBehavior)) {
          console.error(`Error: Invalid --unknown-cmd value: ${unknownCmdBehavior}`);
          console.error(`Valid options: ${validBehaviors.join(', ')}`);
          process.exit(1);
        }

        // Headless mode - plain text output, no Ink UI
        await runHeadless(result.config, question, {
          unknownCommandBehavior: unknownCmdBehavior as 'block' | 'allow-once' | 'allow-remember',
        });
        await stopOutputWriter();
        await connection.cleanup();
        process.exit(0);
      } else {
        // Interactive mode with Ink UI (--output not supported)
        runChatApp(result.config, question);
      }
    } catch (error) {
      console.error((error as Error).message);
      if (connection) await connection.cleanup();
      await stopOutputWriter();
      process.exit(1);
    }
  });

// Session command - interactive headless mode (multiple messages)
program
  .command('session')
  .description('Start interactive headless session (plain text, multi-message)')
  .option('-s, --server <address>', 'Server address (connect to external server)')
  .option('-p, --project <key>', 'Project key', 'default')
  .option('-d, --debug', 'Enable debug mode', false)
  .option('-o, --output <file>', 'Write output to file (in addition to console)')
  .action(async (options) => {
    const globalOpts = program.opts();
    initLogger(options.debug);

    // Initialize output writer if --output specified
    if (options.output) {
      initOutputWriter(options.output);
    }

    let connection: ServerConnection | null = null;

    try {
      const result = await connectAndConfigure(options, globalOpts);
      connection = result.connection;

      registerExitCleanup(connection);

      await runHeadlessInteractive(result.config);
      await stopOutputWriter();
      await connection.cleanup();
    } catch (error) {
      console.error((error as Error).message);
      if (connection) await connection.cleanup();
      await stopOutputWriter();
      process.exit(1);
    }
  });

// Login command
program
  .command('login')
  .description('Login to ByteBrew account')
  .option('-e, --email <email>', 'Email address')
  .option('-p, --password <password>', 'Password')
  .action(async (options) => {
    try {
      const email = options.email || (await prompt('Email: '));
      const password = options.password || (await promptPassword('Password: '));

      const client = new CloudApiClient();
      const tokens = await client.login(email, password);

      new AuthStorage().save(tokens);

      let jwt: string | null = null;
      try {
        jwt = await client.activateLicense();
        new LicenseStorage().save(jwt);
      } catch {
        // No active subscription yet — user can run 'bytebrew activate' later
      }

      if (jwt) {
        const claims = parseJwtPayload(jwt);
        const tierStr = claims.tier ?? 'unknown';
        let expiryStr = '';
        if (claims.exp) {
          const exp = new Date(claims.exp * 1000);
          expiryStr = `, expires ${exp.toISOString().split('T')[0]}`;
        }
        console.log(`Logged in as ${tokens.email} (${tierStr} tier${expiryStr})`);
      } else {
        console.log(`Logged in as ${tokens.email}`);
        console.log("No active subscription found. Run 'bytebrew activate' after subscribing.");
      }
    } catch (err) {
      if (err instanceof CloudApiError) {
        console.error(`Login failed: ${err.message}`);
      } else {
        console.error(`Login failed: ${(err as Error).message}`);
      }
      process.exit(1);
    }
  });

// Register command
program
  .command('register')
  .description('Register new ByteBrew account')
  .option('-e, --email <email>', 'Email address')
  .action(async (options) => {
    try {
      const email = options.email || (await prompt('Email: '));
      const password = await promptPassword('Password: ');
      const confirm = await promptPassword('Confirm password: ');

      if (password !== confirm) {
        console.error('Passwords do not match.');
        process.exit(1);
      }

      const client = new CloudApiClient();
      const tokens = await client.register(email, password);

      new AuthStorage().save(tokens);
      console.log(`Registered as ${tokens.email}`);

      // Open Stripe Checkout for trial (CC required)
      try {
        const checkoutUrl = await client.createCheckout('personal', 'monthly');
        console.log('\nStarting trial... Opening browser for payment setup.');
        openBrowser(checkoutUrl);
        console.log(`If the browser didn't open, visit: ${checkoutUrl}\n`);
        await prompt('Press Enter after completing checkout...');
      } catch {
        // Checkout not available — continue to activate
      }

      try {
        const jwt = await client.activateLicense();
        new LicenseStorage().save(jwt);
        console.log('License activated.');
        showLicenseInfo(jwt);
      } catch {
        console.log("Run 'bytebrew activate' after subscribing.");
      }
    } catch (err) {
      if (err instanceof CloudApiError) {
        console.error(`Registration failed: ${err.message}`);
      } else {
        console.error(`Registration failed: ${(err as Error).message}`);
      }
      process.exit(1);
    }
  });

// Activate command
program
  .command('activate')
  .description('Activate or import license')
  .option('-f, --file <path>', 'Import license from file')
  .action(async (options) => {
    if (options.file) {
      try {
        const jwt = fs.readFileSync(path.resolve(options.file), 'utf-8').trim();
        new LicenseStorage().save(jwt);
        console.log('License activated from file.');
        showLicenseInfo(jwt);
      } catch (err) {
        console.error(`Failed to read license file: ${(err as Error).message}`);
        process.exit(1);
      }
      return;
    }

    const tokens = new AuthStorage().load();
    if (!tokens) {
      console.error('Not logged in. Run "bytebrew login" first.');
      process.exit(1);
    }

    try {
      const client = new CloudApiClient({
        accessToken: tokens.accessToken,
        refreshToken: tokens.refreshToken,
      });
      const jwt = await client.activateLicense();
      new LicenseStorage().save(jwt);
      console.log('License activated.');
      showLicenseInfo(jwt);
    } catch (err) {
      if (err instanceof CloudApiError) {
        console.error(`Activation failed: ${err.message}`);
      } else {
        console.error(`Activation failed: ${(err as Error).message}`);
      }
      process.exit(1);
    }
  });

// Status command
program
  .command('status')
  .description('Show account and license status')
  .action(async () => {
    const tokens = new AuthStorage().load();
    const jwt = new LicenseStorage().load();

    if (!jwt) {
      console.log('No license found. Run "bytebrew login" or "bytebrew activate".');
      return;
    }

    const claims = parseJwtPayload(jwt);

    console.log(`Account: ${tokens?.email ?? 'unknown'}`);
    console.log(`Tier:    ${claims.tier ?? 'unknown'}`);
    console.log(`Status:  ${claims.status ?? 'unknown'}`);
    if (claims.exp) {
      const exp = new Date(claims.exp * 1000);
      const days = Math.ceil((exp.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
      console.log(`Expires: ${exp.toISOString().split('T')[0]} (in ${days} days)`);
    }

    if (!tokens) return;

    try {
      const client = new CloudApiClient({
        accessToken: tokens.accessToken,
        refreshToken: tokens.refreshToken,
      });
      const usage = await client.getUsage();
      console.log('');
      if (usage.proxyStepsLimit > 0) {
        console.log(`Proxy steps: ${usage.proxyStepsUsed}/${usage.proxyStepsLimit} used`);
      } else {
        console.log('Proxy steps: unlimited (trial)');
      }
      console.log(`BYOK:        ${usage.byokEnabled ? 'Enabled' : 'Disabled'}`);
    } catch {
      // API not available — skip usage display
    }
  });

// Logout command
program
  .command('logout')
  .description('Logout and remove license')
  .action(() => {
    new AuthStorage().clear();
    new LicenseStorage().clear();
    console.log('Logged out. License removed.');
  });

// Update command - check and install updates
program
  .command('update')
  .description('Check and install updates')
  .option('--check', 'Only check for updates, do not download or apply')
  .action(async (options) => {
    console.log(`Current version: v${VERSION}`);
    console.log('Checking for updates...');

    const checker = new UpdateChecker(VERSION);
    const info = await checker.check();

    if (!info) {
      console.log('Unable to check for updates (network error).');
      return;
    }

    if (!info.updateAvailable) {
      console.log('You are on the latest version.');
      return;
    }

    console.log(`New version available: v${info.latestVersion}`);

    if (options.check) return;

    // Download
    console.log('Downloading...');
    const downloader = new UpdateDownloader();
    await downloader.download(info, ({ component, phase }) => {
      console.log(`  ${component}: ${phase}`);
    });

    // Apply
    console.log('Applying update...');
    const applier = new UpdateApplier();
    const result = await applier.applyPending();

    if (result.applied) {
      console.log(`Updated to v${info.latestVersion}!`);
      if (isDevMode()) {
        console.log('Note: CLI binary not updated (dev mode).');
      }
      console.log('Restart to use the new version.');
    }
  });

// Index command - index codebase for semantic search
program
  .command('index [path]')
  .description('Index codebase for semantic search')
  .option('-r, --reindex', 'Force full reindex (clear existing index)', false)
  .option('--status', 'Show index status only', false)
  .option('-d, --debug', 'Enable debug mode', false)
  .action(async (targetPath, options) => {
    initLogger(options.debug);

    const rootPath = path.resolve(targetPath || process.cwd());

    const indexer = new Indexer({
      rootPath,
      onProgress: (progress: IndexProgress) => {
        printProgress(progress);
      },
    });

    if (options.status) {
      // Just show status
      try {
        const status = await indexer.getStatus();
        console.log('\nIndex Status:');
        console.log(`  Location: ${rootPath}/.bytebrew/`);
        console.log(`  Total chunks: ${status.totalChunks}`);
        console.log(`  Files indexed: ${status.filesCount}`);
        console.log(`  Languages: ${status.languages.join(', ') || 'none'}`);
        console.log(`  Status: ${status.isStale ? 'Empty/Stale' : 'Ready'}`);
      } catch (error) {
        console.error('\nError:', (error as Error).message);
        process.exit(1);
      }
      return;
    }

    console.log(`\nIndexing: ${rootPath}`);
    console.log(`Storage: ${rootPath}/.bytebrew/`);
    console.log(`Mode: ${options.reindex ? 'Full reindex' : 'Index'}\n`);

    try {
      const status = await indexer.index(options.reindex);
      console.log('\n\nIndexing complete!');
      console.log(`  Total chunks: ${status.totalChunks}`);
      console.log(`  Files indexed: ${status.filesCount}`);
      console.log(`  Languages: ${status.languages.join(', ') || 'none'}`);
    } catch (error) {
      console.error('\n\nIndexing failed:', (error as Error).message);
      process.exit(1);
    }
  });

function printProgress(progress: IndexProgress): void {
  const { phase, filesScanned, totalFiles, chunksProcessed, totalChunks, error } = progress;

  if (error) {
    console.error(`Error: ${error}`);
    return;
  }

  switch (phase) {
    case 'scanning':
      if (totalFiles) {
        process.stdout.write(`\rScanning... Found ${totalFiles} files`);
      } else {
        process.stdout.write('\rScanning...');
      }
      break;
    case 'parsing':
      if (filesScanned && totalFiles) {
        const pct = Math.round((filesScanned / totalFiles) * 100);
        process.stdout.write(`\rParsing: ${filesScanned}/${totalFiles} files (${pct}%) - ${chunksProcessed || 0} chunks`);
      }
      break;
    case 'embedding':
      process.stdout.write(`\rGenerating embeddings for ${totalChunks} chunks...`);
      break;
    case 'storing':
      if (chunksProcessed && totalChunks) {
        const pct = Math.round((chunksProcessed / totalChunks) * 100);
        process.stdout.write(`\rStoring: ${chunksProcessed}/${totalChunks} chunks (${pct}%)`);
      }
      break;
    case 'complete':
      // Final message handled in action
      break;
  }
}

function runChatApp(config: AppConfig, initialQuestion?: string) {
  // Ink 6 options to reduce flickering
  const { waitUntilExit } = render(<App config={config} initialQuestion={initialQuestion} />, {
    // Only update changed lines instead of redrawing entire output
    incrementalRendering: true,
  });

  // Handle process signals
  process.on('SIGINT', () => {
    process.exit(0);
  });

  process.on('SIGTERM', () => {
    process.exit(0);
  });

  // Catch unhandled errors to prevent silent crashes (e.g. native addon failures)
  process.on('uncaughtException', (err) => {
    console.error('[FATAL] Uncaught exception:', err.message);
    process.exit(1);
  });

  process.on('unhandledRejection', (reason) => {
    console.error('[FATAL] Unhandled rejection:', reason);
    process.exit(1);
  });

  waitUntilExit().then(() => {
    process.exit(0);
  });
}

export { program };
