#!/bin/bash
# Бенчмарк: grep+glob vs grep+glob+lsp (прогон 3)
# Запуск: cd vector-cli-node && bash scripts/benchmark-tools.sh

set -e

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SRV="$ROOT/vector-srv"
CLI="$ROOT/vector-cli-node"
TEST_PROJECT="$ROOT/test-project"
RESULTS="$CLI/test-output/benchmark"
FLOWS="$SRV/flows.yaml"
FLOWS_BACKUP="$SRV/flows.yaml.bak"
PORT=60401

# Queries (навигационные — LSP-friendly)
Q1="Найди определение структуры AgentEvent. Покажи все её поля."
Q2="Покажи все места где используется FlowHandler.HandleStream."
Q3="Какие типы реализуют интерфейс Tool в проекте?"

# Configs: name -> search tools
declare -A CONFIGS
CONFIGS[G-grep_glob]="grep_search glob"
CONFIGS[H-grep_glob_lsp]="grep_search glob lsp"

# Base tools: ONLY read_file — forces LLM to use the search tool
BASE_TOOLS="read_file"

echo "=== Tool Benchmark (Прогон 3: LSP) ==="
echo "Test project: $TEST_PROJECT"
echo "Results: $RESULTS"
echo ""

# Prepare
mkdir -p "$RESULTS"
cp "$FLOWS" "$FLOWS_BACKUP"

generate_flows() {
    local search_tools="$1"
    local all_tools="$BASE_TOOLS $search_tools"

    # Write YAML line by line
    {
        echo "flows:"
        echo "  supervisor:"
        echo "    name: \"Supervisor Agent\""
        echo "    system_prompt_ref: \"supervisor_prompt\""
        echo "    tools:"
        for tool in $all_tools; do
            echo "      - $tool"
        done
        echo "    max_steps: 0"
        echo "    max_context_size: 16000"
        echo "    lifecycle:"
        echo "      suspend_on:"
        echo "        - final_answer"
        echo "        - ask_user"
        echo "      report_to: user"
        echo "    spawn_policy:"
        echo "      allowed_flows: []"
        echo "      max_concurrent: 0"
    } > "$FLOWS"
}

start_server() {
    echo "  Starting server..."
    cd "$SRV"
    rm -rf logs/*
    ./server.exe &
    SERVER_PID=$!
    cd "$CLI"
    sleep 3
    # Check if server is up
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        echo "  ERROR: Server failed to start"
        return 1
    fi
    echo "  Server started (PID: $SERVER_PID)"
}

stop_server() {
    echo "  Stopping server..."
    if [ -n "$SERVER_PID" ] && kill -0 $SERVER_PID 2>/dev/null; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    # Also kill by port
    local pids=$(netstat -ano 2>/dev/null | grep ":$PORT" | grep LISTEN | awk '{print $5}' | sort -u)
    for pid in $pids; do
        taskkill //F //PID "$pid" >/dev/null 2>&1 || true
    done
    sleep 2
    echo "  Server stopped"
}

run_query() {
    local config_name="$1"
    local query_num="$2"
    local query="$3"
    local output_file="$RESULTS/${config_name}-q${query_num}.txt"

    echo "  Running Q${query_num}..."
    local start_time=$(date +%s)

    cd "$CLI"
    timeout 180 bun dist/index.js -C "$TEST_PROJECT" ask --headless --new "$query" --output "$output_file" 2>/dev/null || true

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    echo "  Q${query_num} done (${duration}s) -> $output_file"
}

save_logs() {
    local config_name="$1"
    local logs_dir="$RESULTS/${config_name}-logs"
    mkdir -p "$logs_dir"

    # Copy all session directories from logs
    if [ -d "$SRV/logs" ]; then
        cp -r "$SRV/logs/"* "$logs_dir/" 2>/dev/null || true
    fi
    echo "  Logs saved to $logs_dir"
}

# Kill any existing server
stop_server 2>/dev/null || true

# Run benchmark for each config
for config_key in G-grep_glob H-grep_glob_lsp; do
    search_tools="${CONFIGS[$config_key]}"
    echo ""
    echo "=== Config: $config_key (tools: $search_tools) ==="

    # Generate flows.yaml
    generate_flows "$search_tools"
    echo "  flows.yaml generated"

    # Start server
    start_server

    # Run 3 queries
    run_query "$config_key" 1 "$Q1"
    run_query "$config_key" 2 "$Q2"
    run_query "$config_key" 3 "$Q3"

    # Save logs
    save_logs "$config_key"

    # Stop server
    stop_server
done

# Restore original flows.yaml
cp "$FLOWS_BACKUP" "$FLOWS"
rm "$FLOWS_BACKUP"
echo ""
echo "=== Benchmark complete ==="
echo "Results in: $RESULTS"
echo "flows.yaml restored"

# Summary
echo ""
echo "=== Output files ==="
ls -la "$RESULTS"/G-*.txt "$RESULTS"/H-*.txt 2>/dev/null || echo "No output files found"
