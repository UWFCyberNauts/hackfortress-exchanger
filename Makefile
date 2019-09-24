GO = go build
PROTOC = protoc

GO_OUT = hackfortress-exchanger
PROTO_PATH = ./hackfortress-grpc-protocol-spec
PROTO_SOURCE = exchange.proto

BUILDDIR = ./build
GO_OUT_DIR = $(BUILDDIR)/out
PROTO_OUT_DIR = $(BUILDDIR)/exchange

all: deps
	$(GO) -o $(GO_OUT_DIR)/$(GO_OUT)

deps:
	mkdir -p $(GO_OUT_DIR) $(PROTO_OUT_DIR)
	$(PROTOC) -I $(PROTO_PATH) --go_out=plugins=grpc:$(PROTO_OUT_DIR) $(PROTO_SOURCE)
	