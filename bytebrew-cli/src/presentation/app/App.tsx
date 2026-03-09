// Root application component
import React, { useMemo, useEffect } from 'react';
import { Box } from 'ink';
import { ChatApp } from './ChatApp.js';
import { createContainer, resetContainer } from '../../config/container.js';
import { AppConfig } from '../../config/index.js';
import { registerToolDefinitions } from '../../tools/definitions/registerTools.js';
import { createInteractiveAskUserCallback } from '../../tools/askUser.js';

// Register tool definitions once at module load
registerToolDefinitions();

interface AppProps {
  config: AppConfig;
  initialQuestion?: string;
}

export const App: React.FC<AppProps> = ({ config, initialQuestion }) => {
  // Create DI container
  const container = useMemo(() => {
    return createContainer({
      projectRoot: config.projectRoot,
      serverAddress: config.serverAddress,
      wsAddress: config.wsAddress,
      projectKey: config.projectKey,
      sessionId: config.sessionId,
      askUserCallback: createInteractiveAskUserCallback(),
    });
  }, [config.projectRoot, config.serverAddress, config.projectKey, config.sessionId]);

  // Cleanup container on unmount
  useEffect(() => {
    return () => {
      resetContainer();
    };
  }, []);

  return (
    <Box flexDirection="column" width="100%">
      <ChatApp
        container={container}
        initialQuestion={initialQuestion}
      />
    </Box>
  );
};
