import { describe, it, expect, beforeEach, mock } from "bun:test";
import { LspService } from "../LspService.js";
import type { LspManager } from "../LspManager.js";
import type { LspClientInfo } from "../LspClient.js";

const makeLocation = (uri: string, startLine: number) => ({
  uri,
  range: {
    start: { line: startLine, character: 0 },
    end: { line: startLine, character: 10 },
  },
});

function makeClient(sendRequestImpl: (method: string, params: unknown) => unknown): LspClientInfo {
  return {
    serverID: "test-server",
    root: "/test",
    diagnostics: new Map(),
    connection: {
      sendRequest: mock((_method: string, _params: unknown) => {
        try {
          return Promise.resolve(sendRequestImpl(_method, _params));
        } catch (err) {
          return Promise.reject(err);
        }
      }),
    } as any,
    notify: {
      open: mock(() => Promise.resolve()),
    },
    waitForDiagnostics: mock(() => Promise.resolve()),
    waitForReady: mock(() => Promise.resolve(true)),
    shutdown: mock(() => Promise.resolve()),
  };
}

function makeManager(clients: LspClientInfo[]): LspManager {
  return {
    getClients: mock(() => Promise.resolve(clients)),
  } as unknown as LspManager;
}

describe("LspService", () => {
  const filePath = "/project/src/index.ts";
  const line = 5;
  const character = 10;

  describe("definition", () => {
    it("returns locations from LSP server", async () => {
      const loc = makeLocation("file:///project/src/index.ts", 5);
      const client = makeClient(() => loc);
      const manager = makeManager([client]);
      const service = new LspService(manager);

      const result = await service.definition(filePath, line, character);

      expect(result).toHaveLength(1);
      expect(result[0].uri).toBe("file:///project/src/index.ts");
      expect(result[0].range.start.line).toBe(5);
    });

    it("handles array response from LSP", async () => {
      const locs = [
        makeLocation("file:///project/src/a.ts", 1),
        makeLocation("file:///project/src/b.ts", 2),
      ];
      const client = makeClient(() => locs);
      const manager = makeManager([client]);
      const service = new LspService(manager);

      const result = await service.definition(filePath, line, character);

      expect(result).toHaveLength(2);
    });

    it("opens file before sending request", async () => {
      const client = makeClient(() => null);
      const manager = makeManager([client]);
      const service = new LspService(manager);

      await service.definition(filePath, line, character);

      expect(client.notify.open).toHaveBeenCalledWith({ path: filePath });
    });

    it("returns empty array when no clients available", async () => {
      const manager = makeManager([]);
      const service = new LspService(manager);

      const result = await service.definition(filePath, line, character);

      expect(result).toEqual([]);
    });

    it("returns empty array when LSP server errors", async () => {
      const client = makeClient(() => {
        throw new Error("LSP server unavailable");
      });
      const manager = makeManager([client]);
      const service = new LspService(manager);

      const result = await service.definition(filePath, line, character);

      expect(result).toEqual([]);
    });
  });

  describe("references", () => {
    it("returns multiple locations", async () => {
      const locs = [
        makeLocation("file:///project/src/a.ts", 10),
        makeLocation("file:///project/src/b.ts", 20),
        makeLocation("file:///project/src/c.ts", 30),
      ];
      const client = makeClient(() => locs);
      const manager = makeManager([client]);
      const service = new LspService(manager);

      const result = await service.references(filePath, line, character);

      expect(result).toHaveLength(3);
      expect(result[2].range.start.line).toBe(30);
    });

    it("passes includeDeclaration context", async () => {
      let capturedParams: any = null;
      const client = makeClient((method, params) => {
        capturedParams = params;
        return [];
      });
      const manager = makeManager([client]);
      const service = new LspService(manager);

      await service.references(filePath, line, character);

      expect(capturedParams.context).toEqual({ includeDeclaration: true });
    });

    it("returns empty array when no clients available", async () => {
      const manager = makeManager([]);
      const service = new LspService(manager);

      const result = await service.references(filePath, line, character);

      expect(result).toEqual([]);
    });

    it("handles LSP server error gracefully", async () => {
      const client = makeClient(() => {
        throw new Error("connection refused");
      });
      const manager = makeManager([client]);
      const service = new LspService(manager);

      const result = await service.references(filePath, line, character);

      expect(result).toEqual([]);
    });
  });

  describe("implementation", () => {
    it("returns locations", async () => {
      const loc = makeLocation("file:///project/src/impl.ts", 42);
      const client = makeClient(() => loc);
      const manager = makeManager([client]);
      const service = new LspService(manager);

      const result = await service.implementation(filePath, line, character);

      expect(result).toHaveLength(1);
      expect(result[0].range.start.line).toBe(42);
    });

    it("returns empty array when no clients available", async () => {
      const manager = makeManager([]);
      const service = new LspService(manager);

      const result = await service.implementation(filePath, line, character);

      expect(result).toEqual([]);
    });

    it("handles LSP server error gracefully", async () => {
      const client = makeClient(() => {
        throw new Error("server crashed");
      });
      const manager = makeManager([client]);
      const service = new LspService(manager);

      const result = await service.implementation(filePath, line, character);

      expect(result).toEqual([]);
    });

    it("opens file before sending request", async () => {
      const client = makeClient(() => null);
      const manager = makeManager([client]);
      const service = new LspService(manager);

      await service.implementation(filePath, line, character);

      expect(client.notify.open).toHaveBeenCalledWith({ path: filePath });
    });
  });

  describe("merging results from multiple clients", () => {
    it("merges results from all matching LSP clients", async () => {
      const client1 = makeClient(() => makeLocation("file:///a.ts", 1));
      const client2 = makeClient(() => makeLocation("file:///b.ts", 2));
      const manager = makeManager([client1, client2]);
      const service = new LspService(manager);

      const result = await service.definition(filePath, line, character);

      expect(result).toHaveLength(2);
      expect(result.map((l) => l.uri)).toContain("file:///a.ts");
      expect(result.map((l) => l.uri)).toContain("file:///b.ts");
    });
  });
});
