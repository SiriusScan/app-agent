#!/bin/bash

echo "==== Debugging Sirius Agent Communication ===="
echo ""

echo "1. Checking server connection..."
docker exec -it sirius-engine sh -c 'ps aux | grep app-agent'
echo ""

echo "2. Checking RabbitMQ queues..."
docker exec -it sirius-engine sh -c 'rabbitmqctl list_queues'
echo ""

echo "3. Checking tasks sent to the tasks queue..."
docker exec -it sirius-engine sh -c 'cd /app-agent && go run cmd/result-retriever/main.go --format=plain'
echo ""

echo "4. Sending a test task..."
docker exec -it sirius-engine sh -c 'cd /app-agent && go run cmd/agent-message-tester/main.go --type=exec --cmd="hostname" --target=localhost'
echo ""

echo "5. Checking if task was added to queue..."
docker exec -it sirius-engine sh -c 'rabbitmqctl list_queues'
echo ""

echo "6. Checking agent logs..."
tail -n 20 agent.log
echo ""

echo "7. Checking local test agent connection..."
ps aux | grep cmd/agent/main
echo ""

echo "=== Debug Complete ===" 