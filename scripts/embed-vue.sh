#!/usr/bin/env bash
# 把 frontend 构建到 web/dist，供 web 包的 //go:embed 打进 Go 二进制。
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
frontend="${root}/frontend"
dist="${root}/web/dist"

if ! command -v npm >/dev/null 2>&1; then
  echo "需要 Node.js 22+ 和 npm，才能生成要嵌入的 Vue 页面。" >&2
  exit 1
fi

if [[ ! -d "${frontend}/node_modules" ]]; then
  npm ci --prefix "${frontend}"
fi

npm --prefix "${frontend}" run build

if [[ ! -f "${dist}/index.html" ]]; then
  echo "未生成 ${dist}/index.html" >&2
  exit 1
fi

if ! grep -q 'id="app"' "${dist}/index.html"; then
  echo "web/dist/index.html 里没有 Vue 挂载点 #app" >&2
  exit 1
fi

shopt -s nullglob
assets=("${dist}"/assets/*)
if [[ ${#assets[@]} -eq 0 ]]; then
  echo "web/dist/assets 为空，Vue 构建没有产出可嵌入的脚本。" >&2
  exit 1
fi

echo "Vue 页面已写入 web/dist。执行 go build 或 make build 会把它嵌进二进制。"
