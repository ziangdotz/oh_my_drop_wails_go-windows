FIX_PATH = $(subst /,\,$(1))
RM = if exist $(call FIX_PATH, $(1)) rd /s /q $(call FIX_PATH, $(1))
RF = if exist $(call FIX_PATH, $(1)) del /f /q $(call FIX_PATH, $(1))
MD = if not exist $(call FIX_PATH, $(1)) mkdir $(call FIX_PATH, $(1))
UPX_PATH := $(shell where upx 2>nul)

ifdef UPX_PATH
define compress
	@echo ">> 正在压缩 $(1) 并生成带 _upx 后缀的文件..."
	@if exist $(call FIX_PATH, $(1)) copy $(call FIX_PATH, $(1)) $(call FIX_PATH, $(patsubst %.exe,%_upx.exe,$(1))) >nul 2>&1 || cp $(1) $(1)_upx
	@upx --best --lzma $(call FIX_PATH, $(patsubst %.exe,%_upx.exe,$(1))) 2>nul || upx --best --lzma $(1)_upx 2>/dev/null || true
endef
else
define compress
	@echo ">> 跳过压缩: 系统未安装 UPX"
endef
endif


FLAGS := -ldflags "-s -w"

BUILDDIR := build/bin

FRONTENDDIR = frontend

APPNAME := ohMyDrop

all: help

.PHONY: prepare clean help

prepare:
	cd $(FRONTENDDIR) && npm install && cd ../ && go mod download
	@$(call MD, $(BUILDDIR))

run_dev: prepare
	wails dev

build_windows_amd64: prepare clean
	@echo "Build started: Windows amd64 version..."
	wails build -platform windows/amd64 $(FLAGS)
	$(call compress, $(BUILDDIR)/$(APPNAME).exe)
	@echo "Build completed."

build_all: build_windows_amd64

clean:
	$(call RM, $(BUILDDIR))

help:
	@echo ------------------------
	@echo "make run_dev:             run"
	@echo "make build_windows_amd64: build -- Windows amd64 version"
	@echo "make build_all:           build all works"
	@echo "make clean:               clean middle files"
	@echo "make help:                see custom commands"
	@echo "make:                     like 'make help'"
	@echo ------------------------

