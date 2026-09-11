# 构建入口。游戏数据(名称表/图标 webp/宠物消息的 Go 结构体)不在仓库里,由姊妹仓库 rocom-parse
# 从解包目录生成到本仓库的 go:embed 目录(gitignore),所以 go build 之前先 make gamedata。
#
#   make gamedata   # 调 $(ROCOM_PARSE_DIR)/scripts/gen.sh(默认 ../rocom-parse;含增量解包)
#   make web        # 前端 npm build → internal/server/web(已提交,改了前端才需要)
#   make build      # 本机 go build → ./rocom-capture
#   make release    # zig 交叉编译 amd64 + arm64 静态二进制到 dist/
#
# release 背景:livecap 依赖 gopacket/afpacket(cgo),无法用 CGO_ENABLED=0 直接交叉编译;
# zig 自带各架构 musl libc 与 Linux 头,故只需装 zig,无需 arm64 库/sysroot。
#   安装(官方仓库无 zig,单文件免 root):
#     curl -L https://ziglang.org/download/0.16.0/zig-linux-x86_64-0.16.0.tar.xz | tar -xJ
#     export PATH=$$PWD/zig-x86_64-linux-0.16.0:$$PATH

ROCOM_PARSE_DIR ?= ../rocom-parse
GEN      := $(ROCOM_PARSE_DIR)/scripts/gen.sh
PB_PKG   := github.com/whoisnian/rocom-capture/internal/pb
DIST     := dist
# -extldflags=-Wl,-s 让 zig 外部链接器真正 strip(仅 -s -w 对 zig 不完全生效)
LDFLAGS  := -s -w -extldflags=-Wl,-s

.PHONY: all gamedata web build release clean FORCE

all: build

gamedata:
	@test -x "$(GEN)" || { echo "找不到 rocom-parse:$(GEN)(克隆到 ../rocom-parse 或设 ROCOM_PARSE_DIR)" >&2; exit 1; }
	"$(GEN)" --gamedata internal/gamedata/data --pb internal/pb --pb-pkg $(PB_PKG) $(GEN_FLAGS)

web:
	cd web && npm run build

build: gamedata
	go build -o rocom-capture ./cmd/rocom-capture

release: gamedata $(DIST)/rocom-capture-linux-amd64 $(DIST)/rocom-capture-linux-arm64
	@echo "==> 完成:" && ls -lh $(DIST)

$(DIST)/rocom-capture-linux-amd64: FORCE
	@mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		CC="zig cc -target x86_64-linux-musl" \
		go build -trimpath -ldflags "$(LDFLAGS)" -o $@ ./cmd/rocom-capture

$(DIST)/rocom-capture-linux-arm64: FORCE
	@mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC="zig cc -target aarch64-linux-musl" \
		go build -trimpath -ldflags "$(LDFLAGS)" -o $@ ./cmd/rocom-capture

clean:
	rm -rf $(DIST) rocom-capture

FORCE:
