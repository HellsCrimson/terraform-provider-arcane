SHELL := /bin/bash

COMPOSE_FILE ?= tests/docker-compose.yml
ARCANE_ENDPOINT ?= http://127.0.0.1:3552/api
ARCANE_ACC_ENVIRONMENT_ID ?= 0
ARCANE_API_KEY ?= arc_a54fe1040057252a19b34d72008395141a04de7731a28d6f7359baa4923b2f6a
ACC_TEST ?= TestAcc

.PHONY: test test-up test-down test-clean wait-arcane test-acc test-suite

test:
	go test ./...

test-up:
	docker compose -f $(COMPOSE_FILE) up -d

wait-arcane:
	@for i in {1..60}; do \
		if curl -fsS -H "X-API-Key: $(ARCANE_API_KEY)" "$(ARCANE_ENDPOINT)/environments" >/dev/null; then \
			echo "Arcane API is ready"; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Arcane API did not become ready at $(ARCANE_ENDPOINT)" >&2; \
	exit 1

test-acc: test-up wait-arcane
	TF_ACC=1 \
	ARCANE_ENDPOINT="$(ARCANE_ENDPOINT)" \
	ARCANE_API_KEY="$(ARCANE_API_KEY)" \
	ARCANE_ACC_ENVIRONMENT_ID="$(ARCANE_ACC_ENVIRONMENT_ID)" \
	go test ./internal/provider -run "$(ACC_TEST)" -count=1 -v

test-suite: test test-acc

test-down:
	docker compose -f $(COMPOSE_FILE) down

test-clean:
	docker compose -f $(COMPOSE_FILE) down -v
