# dev 长期分支

此分支以官方 `router-for-me/CLIProxyAPI:main` 为基线，仅维护 compact v2、
Responses 客户端工具协议及必要兼容修复。Nexus 子 Key 功能位于独立的
`cpa-nexus-management` 项目，不增加 Nexus 专用的 CPA 主程序补丁。

## 2026-10-05 基线

- 官方 main：`a4acc9f752bd46571f737a10c04bf413656ab06b`。
- compact v2：移植本地 `b0329df8`，保留上游最终 payload 覆盖行为。
- 工具协议：合并官方 PR #6159，head 为
  `97dfdb21be204bb7755350158a603071232dd2b2`。
- 后续修复来源：`hrygo/CLIProxyAPI` 的 `a8c21d31`（递归 schema）、
  `c6e55bbd`、`7b876abb`、`ceccde5b`（完整工具参数整数规范化、
  Codex 请求作用域和 custom 输入原文保护）。
- 上游已包含 Gemini 分段 usage、MAX_TOKENS 和流读取错误处理，
  不重复移植 fork 的对应旧补丁。

## 整合约定

- 保留上游的 apply_patch 能力声明，并独立注入路由决定的搜索能力。
- 工具契约校验必须发生在最终 payload 覆盖之后、发送之前。
- WebSocket 保留 framing 与 payload finalizer，工具历史修复不替代它们。
- steering 继续由上游当前的 `CodexResponseSteering` 设置控制；
  旧 OAuth scope 标记不能覆盖新的共享设置。
- 修复上游 Claude/Kimi 的作用域视图复制 mutex 问题；视图使用新 mutex，
  共享具有自身 mutex 的工具别名缓存，保留线程继续调用的别名。
- compact v2 与普通 Responses 工具桥接分别验证。
- 核心桥接配置为 `requests.responses-tools`，默认开启；显式
  `enabled: false` 可停用。协议边界见 `docs/responses-tools_CN.md`。
- 上游 PR 及移植提交保留历史；不引入 fork 的发布、Homebrew 或上游摄入流程。

## 2026-10-12 同步

- 合并官方 main：`091ff27f`，较上一基线新增 99 个提交，不重写 `dev` 历史。
- 保留 compact v2、Responses 工具桥接、工具参数与 schema 规范化。
- 接入上游 Claude 原生工具搜索、Claude 压缩、WebSocket interrupt 和凭证版本校验。
- 模型目录兼容上游原生工具搜索解析器；开启本地桥接时，搜索能力仍由实际路由策略保守决定，
  不覆盖独立的 apply_patch 能力。停用本地桥接后沿用上游原生能力判断。
- 适配 Meta usage 参数，修复固定日期 golden、Windows 换行比较和取消协程同步问题。
- 修复 WebSocket 终态发送与本地 HTTP interrupt 标记清理之间的竞态：
  在发送终态前清除标记，避免迟到的控制帧取消已完成的请求。
- 本次仅同步 `dev`，不修改 `fork/main`，不执行生产部署。

## 同步和验证

在干净工作树中同步，保留 merge 历史，避免重写已共享的长期分支：

```sh
git switch dev
git fetch origin main
git merge origin/main
```

检查上游是否已实现本地补丁的行为后，再决定移除重复实现；不得仅凭同名文件或
提交标题删除功能。涉及工具协议时检查发现、调用、结果重放、流式响应与
WebSocket；涉及 compact 时检查 `remote_compaction_v2` 与压缩输出。

```sh
gofmt -w <changed-go-files>
go vet ./...
go test ./... -count=1
go test -race ./internal/responsestools ./sdk/cliproxy/auth ./sdk/api/handlers/openai
go build -o temp/cli-proxy-api-dev ./cmd/server
```

临时文件与编译产物放入 `temp/`。分支验证不等于真实提供商或生产部署验证，
测试不得使用实际 OAuth 凭据。

### Windows Docker 启动

此工作区通过直接进程启动 Docker Desktop 曾出现 WSL 目录初始化错误；
通过 Explorer 启动后引擎可正常就绪。启动环境差异已确认，具体权限或环境因素未确定。
优先使用桌面 Explorer 启动入口，并以 `docker info` 成功为就绪条件，不重复直接启动。

```powershell
Start-Process -FilePath explorer.exe -ArgumentList '"C:\Program Files\Docker\Docker\Docker Desktop.exe"' -WindowStyle Hidden
docker info
```

## 本次验证

2026-10-05 使用 Docker `golang:1.26.7-bookworm` 在 Linux amd64 验证：

- 整合代码已执行 `gofmt`。
- 全量 `go test ./... -count=1`：101 个测试包通过。
- `go vet ./...` 通过。
- `internal/responsestools`、`sdk/cliproxy/auth`、
  `internal/runtime/executor`、`sdk/api/handlers/openai` 的竞态检查通过。
- `cmd/server` 编译通过，产物位于 `temp/cli-proxy-api-dev`。

未进行真实提供商 OAuth 请求或生产服务切换。

2026-10-12 使用相同的 Go 1.26.7 Linux 容器验证同步后的代码：

- 全量 `go test ./... -count=1`：102 个测试包通过。
- 工具协议核心、凭证调度、执行器和 OpenAI handler 的全包竞态检查通过。
- 终态中断与 HTTP 取消相关回归测试在竞态检测下各重复 50 次通过。
- `go vet ./...` 和 `cmd/server` 编译通过。
- 编译产物为 `temp/cli-proxy-api-sync-20261012`；JSON 测试报告位于 `temp/sync-*-20261012.json`。

未使用真实 OAuth 凭据，未部署生产服务器。

## GitHub 镜像构建

推送到 `ViceEye/CLIProxyAPI:dev` 后，`dev-image` 工作流先运行格式检查、
`go vet` 和全量测试，再用原生 amd64、arm64 runner 构建并发布 GHCR 镜像。
使用仓库内置的 `GITHUB_TOKEN`，不需要 Docker Hub secret。

- 滚动标签：`ghcr.io/viceeye/cli-proxy-api:dev`。
- 固定版本：`ghcr.io/viceeye/cli-proxy-api:sha-<完整提交 SHA>`。
- 默认镜像保留 CGO 和 Debian bookworm 运行环境，支持现有动态库插件。
- 不在构建过程中刷新模型目录，镜像对应已提交的源码状态。
- 首次发布的 GHCR package 需设为 public，才能匿名拉取；私有 package
  需要先用具有 `read:packages` 权限的凭据登录 GHCR。

服务器沿用现有配置、凭证和插件挂载，只需修改现有 Compose 服务的 image，
或设置 `CLI_PROXY_IMAGE`：

```sh
export CLI_PROXY_IMAGE=ghcr.io/viceeye/cli-proxy-api:dev
docker compose pull cli-proxy-api
docker compose up -d --no-build cli-proxy-api
```

生产部署优先使用固定提交标签，保留上一版本的固定标签用于回退。
