#!/usr/bin/env bash
set -e

# xraycli (v2ray-agent Go) 一键编译与构建脚本
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${PROJECT_ROOT}/bin"
OUTPUT="${BIN_DIR}/xraycli"
CMD_PATH="${PROJECT_ROOT}/cmd/xraycli/main.go"

# 颜色输出
GREEN="\033[32m"
YELLOW="\033[33m"
CYAN="\033[36m"
RESET="\033[0m"

echo -e "${CYAN}==> xraycli (v2ray-agent Go) 构建工具${RESET}"

ACTION="${1:-build}"

case "${ACTION}" in
    "build")
        mkdir -p "${BIN_DIR}"
        echo -e "${YELLOW}正在编译本地平台二进制 [${OUTPUT}]...${RESET}"
        go build -ldflags="-s -w" -o "${OUTPUT}" "${CMD_PATH}"
        echo -e "${GREEN}✔ 编译成功: ${OUTPUT}${RESET}"
        ;;

    "install")
        mkdir -p "${BIN_DIR}"
        echo -e "${YELLOW}正在编译并安装至系统目录 [/usr/local/bin/xraycli]...${RESET}"
        go build -ldflags="-s -w" -o "${OUTPUT}" "${CMD_PATH}"
        install -m 755 "${OUTPUT}" /usr/local/bin/xraycli
        ln -sf /usr/local/bin/xraycli /usr/local/bin/xcli
        ln -sf /usr/local/bin/xraycli /usr/local/bin/v2cli
        echo -e "${GREEN}✔ 安装完成！您可以在终端直接运行 [xraycli]、[xcli] 或 [v2cli]${RESET}"
        ;;

    "release")
        mkdir -p "${BIN_DIR}"
        echo -e "${YELLOW}正在跨平台交叉编译 (Linux AMD64 / ARM64)...${RESET}"
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${BIN_DIR}/xraycli-linux-amd64" "${CMD_PATH}"
        CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "${BIN_DIR}/xraycli-linux-arm64" "${CMD_PATH}"
        echo -e "${GREEN}✔ 跨架构编译成功: ${BIN_DIR}/xraycli-linux-amd64, ${BIN_DIR}/xraycli-linux-arm64${RESET}"
        ;;

    "clean")
        rm -rf "${BIN_DIR}"
        echo -e "${GREEN}✔ 已清理构建产物${RESET}"
        ;;

    *)
        echo "用法: $0 [build|install|release|clean]"
        exit 1
        ;;
esac
