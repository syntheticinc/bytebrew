---
title: "Example: IoT Analyzer"
description: Telemetry monitoring system with multi-agent anomaly detection, InfluxDB queries, Slack alerting, and cron triggers.
---

A telemetry monitoring system with a supervisor agent that coordinates anomaly detection across IoT sensors. This example demonstrates multi-agent orchestration, time-series data queries, Slack alerting with confirmation, and periodic cron triggers.

## What this demonstrates

- **Multi-agent spawning** -- the supervisor delegates anomaly detection to a specialized sub-agent.
- **Time-series queries** -- custom HTTP tool queries InfluxDB for sensor data.
- **Slack alerting with confirmation** -- the agent asks before sending alerts to avoid noise.
- **Cron trigger** -- automatic analysis every 10 minutes.

## Prerequisites

- InfluxDB or similar time-series database with IoT sensor data.
- Slack incoming webhook URL for alert delivery.
- IoT sensors writing telemetry data to the time-series database.

## Full configuration

```yaml
# IoT Analyzer — Telemetry monitoring with anomaly detection
agents:
  iot-supervisor:
    model: glm-5
    system: |
      You monitor IoT device telemetry streams. Detect anomalies,
      correlate events across sensors, and suggest automation rules.
      Alert operators when thresholds are breached.
    can_spawn:
      - anomaly-detector
    tools:
      - manage_tasks
      - send_alert

  anomaly-detector:
    model: qwen-3-32b
    lifecycle: spawn
    system: |
      Analyze the provided sensor data window. Identify statistical
      anomalies, trend deviations, and potential equipment failures.
      Return a structured report with severity levels.
    tools:
      - query_timeseries

tools:
  query_timeseries:
    type: http
    method: POST
    url: "${INFLUX_API}/query"
    body:
      query: "{{flux_query}}"
    auth:
      type: bearer
      token: ${INFLUX_TOKEN}

  send_alert:
    type: http
    method: POST
    url: "${SLACK_WEBHOOK_URL}"
    body:
      text: "{{message}}"
    confirmation_required: true

triggers:
  telemetry-check:
    cron: "*/10 * * * *"
    agent: iot-supervisor
    message: "Analyze the last 10 minutes of telemetry data for anomalies."

models:
  glm-5:
    provider: openai
    api_key: "${OPENAI_API_KEY}"
  qwen-3-32b:
    provider: openai
    api_key: "${OPENAI_API_KEY}"
```

## How to test

```bash
# Manually trigger an analysis
curl -N http://localhost:8080/api/v1/agents/iot-supervisor/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Analyze temperature sensors in Building A for the last hour"}'

# The supervisor will:
# 1. Spawn the anomaly-detector with the specific query
# 2. The detector queries InfluxDB for the data window
# 3. Analyzes the data for statistical anomalies
# 4. Returns a structured report to the supervisor
# 5. Supervisor decides whether to alert operators (with confirmation)

# Check recent task history (cron creates these automatically):
curl http://localhost:8080/api/v1/tasks?agent=iot-supervisor \
  -H "Authorization: Bearer bb_your_token"
```

## Customization tips

- Adjust the cron frequency based on how critical real-time monitoring is (1 min vs 10 min).
- Add more sensor types and corresponding Flux queries for the anomaly detector.
- Create a `knowledge` folder with equipment manuals so the agent can suggest specific remediation steps.
- Replace the Slack webhook with PagerDuty or email for different alerting channels.
- Add a `rule_engine` tool that lets the agent create automated threshold rules based on patterns it discovers.

---

## What's next

- [Multi-Agent Orchestration](/concepts/multi-agent/)
- [DevOps Monitor Example](/examples/devops-monitor/)
- [Configuration Reference](/getting-started/configuration/)
