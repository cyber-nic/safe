COMPOSE ?= docker compose
COMPOSE_FILE ?= compose.yaml
COMPOSE_OVERRIDE_FILE ?= compose.override.yaml
COMPOSE_ENV_FILE ?= .env
LOCALSTACK ?= localstack
CONTROL_PLANE ?= control-plane

COMPOSE_ARGS := -f $(COMPOSE_FILE)

ifneq ($(wildcard $(COMPOSE_OVERRIDE_FILE)),)
COMPOSE_ARGS += -f $(COMPOSE_OVERRIDE_FILE)
endif

ifneq ($(wildcard $(COMPOSE_ENV_FILE)),)
include $(COMPOSE_ENV_FILE)
export
COMPOSE_ARGS += --env-file $(COMPOSE_ENV_FILE)
endif

.PHONY: up down restart build watch logs ps cli token shell-control-plane shell-localstack s3-ls s3-mb s3-rb test test-go test-ts-sdk test-test-vectors test-web clean

up:
	$(COMPOSE) $(COMPOSE_ARGS) up -d --build

down:
	$(COMPOSE) $(COMPOSE_ARGS) down --remove-orphans

cli:
	GOCACHE=$(CURDIR)/.cache/go-build go run ./cmd/safe $(ARGS)

token:
	@node -e 'const crypto=require("node:crypto"); const issuer=process.env.SAFE_OAUTH_ISSUER||"https://auth.safe.local"; const audience=process.env.SAFE_OAUTH_AUDIENCE||"safe-control-plane"; const accountId=process.env.SAFE_OAUTH_ACCOUNT_ID||"acct-dev-001"; const env=process.env.SAFE_ENV||"development"; const secret=process.env.SAFE_OAUTH_HS256_SECRET||""; if (secret.length < 32) { console.error("SAFE_OAUTH_HS256_SECRET must be at least 32 characters"); process.exit(1); } const now=Math.floor(Date.now()/1000); const base64url=(value)=>Buffer.from(JSON.stringify(value)).toString("base64url"); const header=base64url({alg:"HS256",typ:"JWT"}); const payload=base64url({iss:issuer,aud:audience,sub:accountId,accountId,env,iat:now,exp:now+3600}); const signature=crypto.createHmac("sha256", secret).update(header+"."+payload).digest("base64url"); process.stdout.write(header+"."+payload+"."+signature+"\\n");'

restart:
	$(COMPOSE) $(COMPOSE_ARGS) restart

build:
	$(COMPOSE) $(COMPOSE_ARGS) build

watch:
	$(COMPOSE) $(COMPOSE_ARGS) watch

logs:
	$(COMPOSE) $(COMPOSE_ARGS) logs -f

ps:
	$(COMPOSE) $(COMPOSE_ARGS) ps

shell-control-plane:
	$(COMPOSE) $(COMPOSE_ARGS) exec $(CONTROL_PLANE) sh

shell-localstack:
	$(COMPOSE) $(COMPOSE_ARGS) exec $(LOCALSTACK) sh

s3-ls:
	$(COMPOSE) $(COMPOSE_ARGS) exec $(LOCALSTACK) awslocal s3 ls

s3-mb:
	$(COMPOSE) $(COMPOSE_ARGS) exec $(LOCALSTACK) awslocal s3 mb s3://$${BUCKET:-safe-dev}

s3-rb:
	$(COMPOSE) $(COMPOSE_ARGS) exec $(LOCALSTACK) awslocal s3 rb s3://$${BUCKET:-safe-dev} --force

test: test-go test-ts-sdk test-test-vectors test-web

test-go:
	GOCACHE=$(CURDIR)/.cache/go-build go test ./...

test-ts-sdk:
	pnpm --filter @safe/ts-sdk test

test-test-vectors:
	pnpm --filter @safe/test-vectors test

test-web:
	pnpm --filter @safe/web test

clean:
	rm -rf .cache
