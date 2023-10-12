BIN := smoggytexas

GOPATH := $(shell go env GOPATH)
GO_FILES := $(shell find . -name "*.go")

$(BIN): $(GO_FILES)
	gofumpt -w $^
	go build -o $(BIN) cmd/main.go

test: $(BIN)
	./$(BIN) --verbose
.PHONY: test

install: $(GOPATH)/bin/$(BIN)
.PHONY: install

$(GOPATH)/bin/$(BIN): $(BIN)
	mv $(BIN) $(GOPATH)/bin/$(BIN)

clean:
	rm -f $(BIN)
.PHONY: clean
