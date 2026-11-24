#################################################################
#																#
#	This file is intended for the development environment only	#
#		The actual build is performed in GitHub Actions			#
#			该文件仅用于开发环境，实际编译在 Github Action 中		#
#																#
#################################################################

# Monorepo：前端源码在 frontend/，不再依赖同级独立仓库。
FRONTEND_DIR := frontend
EMBED_DIR := embedFS/dist
LITE_ASSET := lite/dist-lite
OUT_DIR := .
RM := rm -rf
CGO := CGO_ENABLED=0
PNPM ?= pnpm
NODE ?= node


ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    SHELL := powershell.exe
	RM = rm -r
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        DETECTED_OS := Linux
    else ifeq ($(UNAME_S),Darwin)
        DETECTED_OS := MacOS
        OUT_DIR = $(HOME)/dmp
    else
        $(error Unsupported OS: $(UNAME_S))
    endif
endif

.PHONY: all frontend-install frontend2backend backend-only frontend-only lite lite-frontend-only lite-copy-frontend help codes codes-verify check

# 安装前端依赖并构建，再构建后端
all: frontend-only copy-frontend backend-only

# 构建前端并将产物复制到后端
frontend2backend: frontend-only copy-frontend

lite-frontend2backend: lite-frontend-only lite-copy-frontend

# 安装前端依赖（严格使用 pnpm-lock.yaml）
frontend-install:
	@echo "=== 正在安装前端依赖, 当前系统为: $(DETECTED_OS) ==="
	$(PNPM) --dir $(FRONTEND_DIR) install --frozen-lockfile
	$(PNPM) --dir $(FRONTEND_DIR) rebuild

# 构建标准版前端
frontend-only: frontend-install
	@echo "=== 正在编译前端, 当前系统为: $(DETECTED_OS) ==="
	$(PNPM) --dir $(FRONTEND_DIR) run build

# 构建后端（会先校验错误码表与文档是否同步）
backend-only: codes-verify
	@echo "=== 正在编译后端, 当前系统为: $(DETECTED_OS) ==="
	$(NODE) -e "require('node:fs').mkdirSync('$(OUT_DIR)', { recursive: true })"
	$(CGO) go build -ldflags '-s -w' -v -o $(OUT_DIR)/dmp

# 复制前端产物到 embedFS/dist（不重新构建前端）
copy-frontend:
	@echo "=== 正在复制前端, 当前系统为: $(DETECTED_OS) ==="
	$(NODE) scripts/copy-dir.mjs $(FRONTEND_DIR)/dist $(EMBED_DIR)

lite: codes-verify
	@echo "=== 正在编译后端(lite), 当前系统为: $(DETECTED_OS) ==="
	$(CGO) go build -ldflags '-s -w' -v -tags=lite -o $(OUT_DIR)/dmp-lite

lite-frontend-only: frontend-install
	@echo "=== 正在编译前端(lite), 当前系统为: $(DETECTED_OS) ==="
	$(PNPM) --dir $(FRONTEND_DIR) run build:lite

lite-copy-frontend:
	@echo "=== 正在复制前端(lite), 当前系统为: $(DETECTED_OS) ==="
	$(NODE) scripts/copy-dir.mjs $(FRONTEND_DIR)/dist-lite $(LITE_ASSET)

# 重新生成业务错误码文档
codes:
	@echo "=== 正在生成业务错误码文档 ==="
	go run ./cmd/codedoc

# 校验错误码表：码唯一、号段正确、词条存在、无死码、文档已同步
codes-verify:
	@echo "=== 正在校验业务错误码 ==="
	go run ./cmd/codedoc -verify

# 提交前的完整检查
check: codes-verify
	@echo "=== 正在检查格式与静态问题 ==="
	gofmt -l $$(git ls-files '*.go')
	go vet ./...
	go vet -tags=lite ./...

help:
	@echo "=== 正在打印帮助信息, 当前系统为: $(DETECTED_OS) ==="
	@echo "make frontend-install    安装 frontend/ 依赖（frozen lockfile）"
	@echo "make frontend2backend    构建前端并将产物复制到后端"
	@echo "make backend-only        构建后端"
	@echo "make lite                构建后端(lite)"
	@echo "make all                 构建前端后构建后端"
	@echo "make codes               重新生成业务错误码文档(docs/error-codes.md)"
	@echo "make codes-verify        校验错误码表与文档是否同步"
	@echo "make check               gofmt + go vet(含 lite) + 错误码校验"
