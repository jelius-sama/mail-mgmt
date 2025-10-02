BIN_DIR := bin
TITLE := mail-mgmt

VERSION := $(shell jq -r '.Version' ./config.json)
PASSWD_FILE := $(shell jq -r '.PasswdFile' ./config.json)
VMAILBASEDIR := $(shell jq -r '.VmailBaseDir' ./config.json)
DOVECOT_SERVICE := $(shell jq -r '.DovecotService' ./config.json)
VMAIL_USER := $(shell jq -r '.VmailUser' ./config.json)
VMAIL_GROUP := $(shell jq -r '.VmailGroup' ./config.json)
HASH_SCHEME := $(shell jq -r '.HashScheme' ./config.json)
DOVEADM_CMD := $(shell jq -r '.DoveadmCmd' ./config.json)

LD_FLAGS := -ldflags "\
	-s -w \
	-X main.Version=$(VERSION) \
	-X main.PasswdFile=$(PASSWD_FILE) \
	-X main.VmailBaseDir=$(VMAILBASEDIR) \
	-X main.DovecotService=$(DOVECOT_SERVICE) \
	-X main.VmailUser=$(VMAIL_USER) \
	-X main.VmailGroup=$(VMAIL_GROUP) \
	-X main.HashScheme=$(HASH_SCHEME) \
	-X main.DoveadmCmd=$(DOVEADM_CMD) \
"

OS_ARCH := \
	linux_amd64 linux_arm linux_arm64 linux_ppc64 linux_ppc64le \
	linux_mips linux_mipsle linux_mips64 linux_mips64le linux_s390x \
	darwin_amd64 darwin_arm64 \
	freebsd_amd64 freebsd_386 \
	openbsd_amd64 openbsd_386 openbsd_arm64 \
	netbsd_amd64 netbsd_386 netbsd_arm \
	dragonfly_amd64 \
	solaris_amd64 \
	plan9_386 plan9_amd64

RED := \033[0;31m
GREEN := \033[0;32m
NC := \033[0m

.PHONY: all clean host_default cross

all: host_default cross

host_default:
	@mkdir -p $(BIN_DIR)
	@echo "Building host binary..."
	@CGO_ENABLED=0 go build $(LD_FLAGS) -trimpath -buildvcs=false -o $(BIN_DIR)/$(TITLE) ./ && \
		printf '$(GREEN)Build succeeded: host_default$(NC)\n' || \
		(printf '$(RED)Build failed: host_default$(NC)\n' && exit 1)

cross: $(OS_ARCH)

$(OS_ARCH):
	@mkdir -p $(BIN_DIR)
	@OS=$$(echo $@ | cut -d_ -f1); \
	ARCH=$$(echo $@ | cut -d_ -f2); \
	OUT=$(BIN_DIR)/$(TITLE)-$$OS-$$ARCH; \
	echo "Building $@..."; \
	CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH go build $(LD_FLAGS) -trimpath -buildvcs=false -o $$OUT ./ && \
	printf '$(GREEN)Build succeeded: $@$(NC)\n' || \
	(printf '$(RED)Build failed: $@$(NC)\n' && exit 1)

clean:
	@rm -rf $(BIN_DIR)
	@echo "Cleaned binaries."
