package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	deliverygrpc "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/delivery/grpc"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/delivery/ws"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/testutil"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/tools"
	agentservice "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/agent"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/engine"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/eventstore"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/session_processor"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/config"
	"google.golang.org/grpc"
)

func main() {
	scenario := flag.String("scenario", "echo", "Scenario name (echo|server-tool|reasoning|error|proxied-read|proxied-write|proxied-exec|ask-user|multi-tool|tool-error|task-create|proxied-edit|proxied-tree|proxied-search|multi-agent|agent-interrupt|agent-failure|multi-agent-read)")
	port := flag.Int("port", 0, "Port (0 = random)")
	licenseStatus := flag.String("license", "active", "License status (active|grace|blocked)")
	flag.Parse()

	// 1. Create mock ChatModel
	chatModel := llm.NewMockChatModel(*scenario)

	// 2. Create Engine with in-memory repos
	snapshotRepo := testutil.NewMockSnapshotRepo()
	historyRepo := testutil.NewMockHistoryRepo()
	agentEngine := engine.New(snapshotRepo, historyRepo)

	// 3. Create FlowManager programmatically (no flows.yaml)
	flowsCfg, promptsCfg := testutil.TestFlowConfig()
	flowManager, err := agentservice.NewFlowManager(flowsCfg, promptsCfg)
	if err != nil {
		log.Fatalf("Failed to create flow manager: %v", err)
	}

	// 4. Create ToolResolver
	toolResolver := tools.NewDefaultToolResolver()

	// 5. Create AgentConfig
	agentConfig := &config.AgentConfig{
		MaxContextSize:     4000,
		MaxSteps:           10,
		ToolReturnDirectly: make(map[string]struct{}),
		Prompts:            promptsCfg,
	}

	// 6. Create mock managers
	subtaskMgr := testutil.NewMockSubtaskManager()
	taskMgr := testutil.NewMockTaskManager()

	// Pre-seed subtask for "multi-agent" scenario
	if *scenario == "multi-agent" {
		subtaskMgr.Subtasks["test-subtask-1"] = &domain.Subtask{
			ID:          "test-subtask-1",
			SessionID:   "",
			TaskID:      "test-task-1",
			Title:       "Implement greeting function",
			Description: "Create a greeting function that returns Hello World.",
			Status:      domain.SubtaskStatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	// Pre-seed subtask for "agent-interrupt" scenario
	if *scenario == "agent-interrupt" {
		subtaskMgr.Subtasks["test-subtask-1"] = &domain.Subtask{
			ID:          "test-subtask-1",
			SessionID:   "",
			TaskID:      "test-task-1",
			Title:       "Long running task",
			Description: "This task takes a long time to complete.",
			Status:      domain.SubtaskStatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	// Pre-seed subtask for "agent-failure" scenario
	if *scenario == "agent-failure" {
		subtaskMgr.Subtasks["test-subtask-1"] = &domain.Subtask{
			ID:          "test-subtask-1",
			SessionID:   "",
			TaskID:      "test-task-1",
			Title:       "Failing task",
			Description: "This task will fail.",
			Status:      domain.SubtaskStatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	// Pre-seed subtask for "multi-agent-read" scenario
	if *scenario == "multi-agent-read" {
		subtaskMgr.Subtasks["test-subtask-1"] = &domain.Subtask{
			ID:          "test-subtask-1",
			SessionID:   "",
			TaskID:      "test-task-1",
			Title:       "Read source file",
			Description: "Read the main source file.",
			Status:      domain.SubtaskStatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	// 7. Create ModelSelector and AgentPool for multi-agent support
	modelSelector := llm.NewModelSelector(chatModel, "mock-model")
	agentRunStorage := testutil.NewMockAgentRunStorage()
	agentPool := agentservice.NewAgentPool(agentservice.AgentPoolConfig{
		ModelSelector:   modelSelector,
		SubtaskManager:  subtaskMgr,
		AgentRunStorage: agentRunStorage,
		AgentConfig:     agentConfig,
		MaxConcurrent:   0,
	})
	agentPoolAdapter := agentservice.NewAgentPoolAdapter(agentPool)

	// Create ToolDepsProvider for AgentPool (code agents need tool deps)
	toolDepsProvider := tools.NewDefaultToolDepsProvider(
		nil,             // proxy — will be set per-session by FlowHandler
		taskMgr,
		subtaskMgr,
		agentPoolAdapter,
		nil, nil,        // webSearchTool, webFetchTool
	)

	// Set Engine deps on AgentPool so spawned agents can run
	agentPool.SetEngine(agentEngine, flowManager, toolResolver, toolDepsProvider)

	// 8. Create EngineTurnExecutorFactory (SAME as production!)
	factory := infrastructure.NewEngineTurnExecutorFactory(
		agentEngine,
		flowManager,
		toolResolver,
		modelSelector,
		agentConfig,
		taskMgr,
		subtaskMgr,
		agentPoolAdapter, // was nil
		nil,
		nil,
		nil, // contextRemindersGetter — not needed in test
	)

	// 9. Create FlowHandler (SAME as production!)
	flowRegistry := flow_registry.NewInMemoryRegistry()
	// Create in-memory event store for tests
	eventsDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open in-memory events db: %v", err)
	}
	eventsDB.SetMaxOpenConns(1)
	defer eventsDB.Close()

	evtStore, err := eventstore.New(eventsDB)
	if err != nil {
		log.Fatalf("Failed to create event store: %v", err)
	}

	sessionRegistry := flow_registry.NewSessionRegistry(evtStore)
	sessProcessor := session_processor.New(sessionRegistry, factory, evtStore)
	sessProcessor.SetAgentPoolRegistrar(agentPool)

	flowHandlerCfg := deliverygrpc.FlowHandlerConfig{
		AgentService:        &testutil.NoopAgentService{},
		TurnExecutorFactory: factory,
		PingInterval:        60 * time.Second, // Keep-alive ping every 60s
		FlowRegistry:        flowRegistry,
		SessionRegistry:     sessionRegistry,
		SessionProcessor:    sessProcessor,
		AgentPoolProxy:      agentPool,        // NEW: for proxy/callback management
		AgentPoolAdapter:    agentPoolAdapter, // NEW: for spawn_code_agent tool
	}

	flowHandler, err := deliverygrpc.NewFlowHandlerWithConfig(flowHandlerCfg)
	if err != nil {
		log.Fatalf("Failed to create flow handler: %v", err)
	}

	// 10. Build license info from CLI flag
	licenseInfo := &domain.LicenseInfo{Status: domain.LicenseActive}
	switch *licenseStatus {
	case "grace":
		licenseInfo.Status = domain.LicenseGrace
	case "blocked":
		licenseInfo.Status = domain.LicenseBlocked
	}

	// 11. Start gRPC server with license interceptors
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(deliverygrpc.LicenseUnaryInterceptor(licenseInfo)),
		grpc.StreamInterceptor(deliverygrpc.LicenseStreamInterceptor(licenseInfo)),
	)
	pb.RegisterFlowServiceServer(grpcServer, flowHandler)

	// 12. Create WS server (SAME as production!)
	wsHandler := ws.NewConnectionHandler(sessionRegistry, sessProcessor, &testutil.NoopAgentService{}, nil, &domain.LicenseInfo{Status: domain.LicenseActive})
	wsServer, err := ws.NewServer(wsHandler)
	if err != nil {
		log.Fatalf("Failed to create WS server: %v", err)
	}

	// Start WS server in goroutine
	go func() {
		if err := wsServer.Start(context.Background()); err != nil {
			log.Printf("WS server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		_ = wsServer.Shutdown(context.Background())
		grpcServer.GracefulStop()
	}()

	// Print READY:{grpc_port}:{ws_port} for client to parse
	actualPort := listener.Addr().(*net.TCPAddr).Port
	fmt.Printf("READY:%d:%d\n", actualPort, wsServer.Port())
	os.Stdout.Sync()

	// Serve (blocks until GracefulStop or error)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
