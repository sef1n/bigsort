.PHONY: build run clean 

BUILD_DIR := ./bin
SERVICES  := sorter generator

build: $(addprefix build-,$(SERVICES))
build-%: | $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$* ./cmd/$*

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

run:
	@if [ -z "$(SVC)" ]; then \
		echo "Укажи сервис: make run SVC=main"; \
		exit 1; \
	fi
	go run ./cmd/$(SVC) $(ARGS)

clean:
	rm -rf $(BUILD_DIR)
