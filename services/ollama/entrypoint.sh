#!/bin/bash

# Start Ollama server in background
ollama serve &

# Wait for server to be ready
echo "Waiting for Ollama server to start..."
for i in $(seq 1 30); do
  if ollama list >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

# Pull model if not already present
MODEL="${OLLAMA_MODEL:-gemma3:4b}"
if ! ollama list | grep -q "$MODEL"; then
  echo "Pulling model $MODEL (this may take a few minutes on first start)..."
  ollama pull "$MODEL"
  echo "Model $MODEL ready."
else
  echo "Model $MODEL already loaded."
fi

# Keep server running in foreground
wait
