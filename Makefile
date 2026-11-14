.PHONY: build run clean test test-cover-html test-cover test-bench

BUILD_DIR := ./bin
SERVICES  := sorter generator
TEST_PKGS := ./internal/...

# Bench params
BTime ?= 1
BCount ?= 1
BOut ?=
BName ?=

build: $(addprefix build-,$(SERVICES))
build-%: | $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$* ./cmd/$*

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

run:
	@if [ -z "$(SVC)" ]; then \
		echo "Укажи сервис: make run SVC=sorter"; \
		exit 1; \
	fi
	go run ./cmd/$(SVC) $(ARGS)

clean:
	rm -rf $(BUILD_DIR)

test:
	go test $(TEST_PKGS)

test-cover:
	go test -coverprofile=c.out $(TEST_PKGS)
	go tool cover -func=c.out

test-cover-html: test-cover
	go tool cover -html=c.out

test-bench:
	@if [ -z "$(BName)" ]; then \
		echo "Укажи bench name: make test-bench BN=BenchmarkSmth"; \
		exit 1; \
	fi
	go test -bench=$(BName) -benchtime=$(BTime)x -count=$(BCount) $(TEST_PKGS) | tee $(BOut)
