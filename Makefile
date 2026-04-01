COMPOSE ?= docker compose
COMPOSE_FILE ?= compose.yaml
LOCALSTACK ?= localstack
CONTROL_PLANE ?= control-plane

.PHONY: up down restart build watch logs ps cli shell-control-plane shell-localstack s3-ls s3-mb s3-rb test test-go test-ts-sdk test-test-vectors clean

up:
	$(COMPOSE) -f $(COMPOSE_FILE) up -d --build

down:
	$(COMPOSE) -f $(COMPOSE_FILE) down --remove-orphans

cli:
	GOCACHE=$(CURDIR)/.cache/go-build go run ./cmd/safe $(ARGS)

restart:
	$(COMPOSE) -f $(COMPOSE_FILE) restart

build:
	$(COMPOSE) -f $(COMPOSE_FILE) build

watch:
	$(COMPOSE) -f $(COMPOSE_FILE) watch

logs:
	$(COMPOSE) -f $(COMPOSE_FILE) logs -f

ps:
	$(COMPOSE) -f $(COMPOSE_FILE) ps

shell-control-plane:
	$(COMPOSE) -f $(COMPOSE_FILE) exec $(CONTROL_PLANE) sh

shell-localstack:
	$(COMPOSE) -f $(COMPOSE_FILE) exec $(LOCALSTACK) sh

s3-ls:
	$(COMPOSE) -f $(COMPOSE_FILE) exec $(LOCALSTACK) awslocal s3 ls

s3-mb:
	$(COMPOSE) -f $(COMPOSE_FILE) exec $(LOCALSTACK) awslocal s3 mb s3://$${BUCKET:-safe-dev}

s3-rb:
	$(COMPOSE) -f $(COMPOSE_FILE) exec $(LOCALSTACK) awslocal s3 rb s3://$${BUCKET:-safe-dev} --force

test: test-go test-ts-sdk test-test-vectors

test-go:
	GOCACHE=$(CURDIR)/.cache/go-build go test ./...

test-ts-sdk:
	pnpm --filter @safe/ts-sdk test

test-test-vectors:
	pnpm --filter @safe/test-vectors test

clean:
	rm -rf .cache
