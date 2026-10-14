package web

import "embed"

// Dist 是生产构建的 Vue 页面。改完前端后执行 go generate ./web、
// bash scripts/embed-vue.sh，或在 Windows 上执行 scripts\embed-vue.bat，
// 再编译即可嵌进二进制。
//
//go:generate bash ../scripts/embed-vue.sh
//go:embed all:dist
var Dist embed.FS
