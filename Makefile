# Makefile репозитория banking_mini_cores. Единственный на весь монорепозиторий.
#
# Три группы целей: генерация кода из proto, сборка образов и запуск стека.
# Полный цикл с нуля:
#
#   make tools     # один раз на машину
#   make generate
#   make up
#
# Работает в Git Bash: рецепты используют sh-синтаксис, cmd.exe не подойдёт.
#
# Сообщения, которые цели печатают, — на английском намеренно. make под Windows
# передаёт текст рецепта дочернему shell через системную ANSI-кодировку, и
# кириллица в них превращается в кашу. Описания после ## этим не задеты: их
# читает awk прямо из файла.

# Под Windows make нередко запускают не из Git Bash - например, кнопкой в
# GoLand. Путь /bin/bash тогда не разрешается, make молча откатывается на
# cmd.exe, и все рецепты ломаются с невнятным "is not recognized as an internal
# or external command". Поэтому bash ищем явно.
#
# PROGRA~1 и PROGRA~2 - это короткие имена "Program Files" и "Program Files
# (x86)". Они нужны потому, что $(wildcard) разбивает аргумент по пробелам, и
# путь с пробелом сюда просто не записать.
#
# Если bash лежит где-то ещё, переопределите: make SHELL=/путь/к/bash.exe
ifeq ($(OS),Windows_NT)
BASH_CANDIDATES := \
	C:/PROGRA~1/Git/bin/bash.exe \
	C:/PROGRA~2/Git/bin/bash.exe \
	C:/Programs/Git/bin/bash.exe \
	C:/Git/bin/bash.exe \
	$(subst \,/,$(LOCALAPPDATA))/Programs/Git/bin/bash.exe
SHELL := $(firstword $(wildcard $(BASH_CANDIDATES)) /bin/bash)
else
SHELL := /bin/bash
endif

.SHELLFLAGS := -eu -o pipefail -c

# Сервисы, у которых есть buf.gen.yaml. kafka-service и monitor сюда не входят:
# у первого нет своего кода вовсе, у второго нет proto.
PROTO_SERVICES := \
	account-service \
	antifraud-service \
	api-gateway \
	auth-service \
	customer-service \
	document-service \
	kyc-service \
	ledger-service \
	notification-service

# Модули go.work. Нужны там, где обходим все модули: platform лежит отдельно от
# services и в PROTO_SERVICES не попадает.
GO_MODULES := platform $(addprefix services/,$(PROTO_SERVICES)) services/monitor

# Стеки docker compose. Порядок значим только для kafka-service - см. up.
COMPOSE_STACKS := \
	auth-service \
	customer-service \
	kyc-service \
	document-service \
	account-service \
	ledger-service \
	antifraud-service \
	notification-service \
	api-gateway \
	monitor

NETWORK := banking-net
KAFKA_CONTAINER := banking-kafka
KAFKA_INIT_CONTAINER := banking-kafka-init
KAFKA_TIMEOUT := 120

# Плагины кодогенерации. Версии не закреплены: buf.gen.yaml зовёт их как
# local-плагины, то есть берёт из PATH, а не тянет сам.
TOOLS := \
	github.com/bufbuild/buf/cmd/buf@latest \
	google.golang.org/protobuf/cmd/protoc-gen-go@latest \
	google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest \
	github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest \
	github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

.DEFAULT_GOAL := help

# ---------------------------------------------------------------- служебное --

.PHONY: help
help: ## Показать список целей
	@echo "Targets:"
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: tools
tools: ## Установить buf и плагины кодогенерации в $(go env GOPATH)/bin
	@for t in $(TOOLS); do echo "go install $$t"; go install "$$t"; done
	@echo "Done. Make sure \$$(go env GOPATH)/bin is on PATH."

.PHONY: check-tools
check-tools: ## Проверить, что buf и плагины доступны в PATH
	@missing=""; \
	for b in buf protoc-gen-go protoc-gen-go-grpc protoc-gen-grpc-gateway protoc-gen-openapiv2; do \
		command -v "$$b" >/dev/null 2>&1 || missing="$$missing $$b"; \
	done; \
	if [ -n "$$missing" ]; then \
		echo "Missing:$$missing"; \
		echo "Run: make tools"; \
		exit 1; \
	fi; \
	echo "All tools present."

# --------------------------------------------------------------- генерация ---

.PHONY: generate
generate: check-tools ## Сгенерировать Go-код из proto во всех сервисах
	@for s in $(PROTO_SERVICES); do \
		echo "==> buf generate: $$s"; \
		(cd services/$$s && buf generate); \
	done
	@echo "Generated code is under services/*/internal/pb/gen."

.PHONY: proto-lint
proto-lint: check-tools ## Проверить proto линтером buf
	@for s in $(PROTO_SERVICES); do \
		echo "==> buf lint: $$s"; \
		(cd services/$$s && buf lint); \
	done

.PHONY: proto-format
proto-format: check-tools ## Отформатировать proto на месте
	@for s in $(PROTO_SERVICES); do \
		echo "==> buf format: $$s"; \
		(cd services/$$s && buf format -w); \
	done

# proto-файлы продублированы по сервисам: у каждого своя копия с собственным
# go_package. Копии расходятся молча, и обнаруживается это уже несовместимостью
# на проводе, поэтому расхождение лучше ловить заранее.
.PHONY: proto-diff
proto-diff: ## Сверить копии proto сервисов с копией в api-gateway
	@status=0; \
	for s in $(PROTO_SERVICES); do \
		if [ "$$s" != "api-gateway" ]; then \
			for f in $$(cd services/$$s/proto && find . -name '*.proto' -not -path './google/*'); do \
				ref="services/api-gateway/proto/$${f#./}"; \
				if [ -f "$$ref" ]; then \
					if ! diff -q \
						<(grep -v 'go_package' "services/$$s/proto/$${f#./}") \
						<(grep -v 'go_package' "$$ref") >/dev/null; then \
						echo "DIFF: services/$$s/proto/$${f#./}"; \
						status=1; \
					fi; \
				fi; \
			done; \
		fi; \
	done; \
	if [ $$status -eq 0 ]; then echo "All proto copies match (go_package ignored)."; fi; \
	exit $$status

# -------------------------------------------------------------------- Go -----

.PHONY: build
build: ## Собрать все Go-модули
	@for m in $(GO_MODULES); do echo "==> go build: $$m"; (cd $$m && go build ./...); done

.PHONY: test
test: ## Прогнать тесты всех модулей
	@for m in $(GO_MODULES); do echo "==> go test: $$m"; (cd $$m && go test ./...); done

.PHONY: vet
vet: ## Прогнать go vet по всем модулям
	@for m in $(GO_MODULES); do echo "==> go vet: $$m"; (cd $$m && go vet ./...); done

.PHONY: fmt
fmt: ## Отформатировать Go-код (сгенерированный не трогает)
	@gofmt -w $$(find . -name '*.go' -not -path '*/internal/pb/gen/*' -not -path '*/vendor/*')

.PHONY: tidy
tidy: ## go mod tidy во всех модулях
	@for m in $(GO_MODULES); do echo "==> go mod tidy: $$m"; (cd $$m && go mod tidy); done

.PHONY: check
check: proto-lint vet test ## Линт proto + vet + тесты

# ---------------------------------------------------------------- Docker -----

.PHONY: net
net: ## Создать общую сеть banking-net (идемпотентно)
	@docker network inspect $(NETWORK) >/dev/null 2>&1 \
		|| docker network create $(NETWORK)

.PHONY: images
images: ## Собрать образы всех сервисов, не запуская их
	@for s in $(COMPOSE_STACKS); do \
		echo "==> docker compose build: $$s"; \
		(cd services/$$s && docker compose build); \
	done

.PHONY: up
up: net kafka-up ## Поднять весь стек (сеть, брокер, затем сервисы)
	@for s in $(COMPOSE_STACKS); do \
		echo "==> docker compose up: $$s"; \
		(cd services/$$s && docker compose up -d --build); \
	done
	@echo
	@$(MAKE) --no-print-directory ps

# Брокер поднимается отдельно и раньше всех по двум причинам: сервисы проверяют
# связь с Kafka на старте и падают, если её нет, а консьюмеру вдобавок нужно,
# чтобы топики уже существовали в момент входа в группу — иначе он останется без
# партиций. Топики заводит задача kafka-init, её завершения тоже надо дождаться.
.PHONY: kafka-up
kafka-up: net ## Поднять Kafka, дождаться healthy и создания топиков
	@(cd services/kafka-service && docker compose up -d)
	@echo -n "waiting for $(KAFKA_CONTAINER) to become healthy"
	@healthy=0; \
	for i in $$(seq 1 $(KAFKA_TIMEOUT)); do \
		state=$$(docker inspect --format '{{.State.Health.Status}}' $(KAFKA_CONTAINER) 2>/dev/null || echo missing); \
		if [ "$$state" = "healthy" ]; then healthy=1; break; fi; \
		echo -n "."; sleep 1; \
	done; \
	if [ $$healthy -eq 0 ]; then \
		echo; echo "broker not healthy after $(KAFKA_TIMEOUT)s, see: make kafka-logs"; exit 1; \
	fi; \
	echo " ok"
	@echo -n "waiting for topics"
	@for i in $$(seq 1 $(KAFKA_TIMEOUT)); do \
		st=$$(docker inspect --format '{{.State.ExitCode}}:{{.State.Running}}' $(KAFKA_INIT_CONTAINER) 2>/dev/null || echo "x:x"); \
		case "$$st" in \
			"0:false") echo " ok"; exit 0;; \
			*":false") echo; echo "topic creation failed: docker logs $(KAFKA_INIT_CONTAINER)"; exit 1;; \
		esac; \
		echo -n "."; sleep 1; \
	done; \
	echo; echo "topics not created after $(KAFKA_TIMEOUT)s"; exit 1

.PHONY: down
down: ## Остановить все сервисы и брокер (тома сохраняются)
	@for s in $(COMPOSE_STACKS); do \
		echo "==> docker compose down: $$s"; \
		(cd services/$$s && docker compose down); \
	done
	@(cd services/kafka-service && docker compose down)

# Отдельная цель, а не флаг: -v стирает тома с базами и данными Kafka, и такое
# лучше набирать осознанно.
.PHONY: clean
clean: ## Остановить всё И УДАЛИТЬ тома (базы, данные Kafka)
	@read -p "Delete ALL volumes (databases and Kafka data)? [y/N] " ok; \
	if [ "$$ok" != "y" ]; then echo "Cancelled."; exit 1; fi
	@for s in $(COMPOSE_STACKS); do (cd services/$$s && docker compose down -v); done
	@(cd services/kafka-service && docker compose down -v)

.PHONY: restart
restart: down up ## Перезапустить стек, сохранив тома

.PHONY: ps
ps: ## Показать состояние контейнеров стека
	@docker ps -a --filter "network=$(NETWORK)" --format "{{.Names}}\t{{.Status}}" \
		| sort | column -t -s $$'\t'

.PHONY: logs
logs: ## Логи сервиса: make logs S=auth-service
	@if [ -z "$(S)" ]; then echo "Usage: make logs S=auth-service"; exit 1; fi
	@(cd services/$(S) && docker compose logs -f --tail 100)

.PHONY: kafka-logs
kafka-logs: ## Логи брокера
	@(cd services/kafka-service && docker compose logs -f --tail 100)

# MSYS_NO_PATHCONV=1 нужен Git Bash: без него он принимает /opt/kafka/... за
# путь Windows и переписывает его в C:/Programs/Git/opt/kafka/..., после чего
# docker exec не находит файл внутри контейнера. На Linux и macOS переменная
# просто игнорируется.
.PHONY: topics
topics: ## Список топиков в брокере
	@MSYS_NO_PATHCONV=1 docker exec $(KAFKA_CONTAINER) \
		/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list

.PHONY: health
health: ## Health всех сервисов глазами monitor
	@docker logs --tail 20 monitor 2>&1 | grep 'msg=health' | tail -1 \
		|| echo "monitor is not running, try: make up"

# Если рецепты ведут себя странно - например, запуск из IDE падает с
# "is not recognized as an internal or external command" - начните с этой цели:
# скорее всего make не нашёл bash и откатился на cmd.exe.
.PHONY: which-shell
which-shell: ## Показать, какой shell использует make (диагностика)
	@echo "SHELL = $(SHELL)"
	@echo -n "bash version: "; $(SHELL) --version | head -1
