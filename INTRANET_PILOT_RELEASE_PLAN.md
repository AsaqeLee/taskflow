# TaskFlow 内网试点发布收口计划

面向“从当前候选发布版推进到可试点部署版”的下一阶段执行计划。当前 `main` 已推进到 `0d7edf3`，本地自动化、Compose 验收、Playwright 主流程与最新 GitHub Actions `ci` 均已通过；同时 `docker-compose.yml` 已支持通过 `.env` / shell 注入生产参数。项目当前的主要矛盾已经不是“继续补 MVP 功能”，而是“把候选发布版收口成可试点部署版”。

## 1. 本轮目标

1. 收口发布资料与事实状态，避免文档落后于代码。
2. 准备真实生产参数与首批用户文件，完成目标环境部署前置项。
3. 在目标内网环境执行一次最小试点验收，形成 go / no-go 结论。

## 2. 默认假设

- 本轮不新增第二期功能，不引入用户管理 UI、密码重置页、分页搜索、附件、通知等新范围。
- 生产部署仍沿用当前 `docker-compose.yml`、`INTRANET_RUNBOOK.md`、`INTRANET_OPS.md` 路径。
- 真实内网域名、入口层、证书、主机信息尚未在仓库中固化，需要在执行阶段由部署方提供。

## 3. 非目标

- 不把生产密钥、真实密码、真实域名直接提交进仓库。
- 不在本轮引入 Nginx/Caddy 新服务，除非目标环境验证明确需要。
- 不把“试点上线前收口”和“第二期产品增强”混成同一轮任务。

## 4. 当前状态基线

已完成：

- 内网 MVP 功能闭环已完成，见 `INTRANET_MVP.md`。
- 本地验证链路已跑通：
  - `go test ./...`
  - `cd web && npm run lint && npm run test && npm run build`
  - `bash scripts/compose_smoke.sh`
  - `COLD_START=1 bash scripts/intranet_acceptance.sh`
  - `cd web && npm run acceptance:browser`
- Compose 已参数化，`JWT_SECRET`、`APP_VERSION`、`BOOTSTRAP_USERS_FILE`、`CORS_ALLOWED_ORIGINS` 可通过 `.env` / shell 覆盖。
- 最新 `main` push 的云端 `ci` 已成功。

未完成：

- 真实生产值尚未落地：
  - 强随机 `JWT_SECRET`
  - 明确的 `APP_VERSION`
  - 自定义 `BOOTSTRAP_USERS_FILE`
  - 真实 `CORS_ALLOWED_ORIGINS`
- bootstrap 默认密码尚未替换为生产值。
- 移动端 / 窄屏人工验收尚未完成。
- 正式发布清单与临时校验文档仍引用旧 commit / 旧备份记录。
- HTTPS / 反向代理 / 防火墙策略仍停留在文档建议层，未形成目标环境实配记录。

## 5. 里程碑

| 里程碑 | 目标 | 关键产出 | 预计耗时 |
|--------|------|----------|----------|
| M1 | 同步发布资料 | 更新后的发布清单、校验文档、当前版本事实表 | 0.5 天 |
| M2 | 准备生产参数 | 目标 `.env`、自定义 bootstrap 用户文件、参数核对记录 | 0.5-1 天 |
| M3 | 目标环境试点验收 | 目标环境部署记录、健康检查、浏览器点验、备份验证 | 0.5-1 天 |
| M4 | 上线判定 | go / no-go 结论、回滚入口、后续 backlog 分类 | 0.5 天 |

## 6. 执行计划

### 6.1 同步发布资料

目标：解决“文档事实落后于代码”的问题。

任务：

- 更新 `INTRANET_RELEASE_CHECKLIST.md` 中的 `Git commit`、备份文件路径、风险描述。
- 将本轮校验结论整理到正式、受版本控制的发布记录，例如 `reports/intranet-release-assessment-2026-06-22.md`。
- 明确写出“代码已可部署，但默认值不能直接上内网”的结论，避免口径混淆。

验收信号：

- 发布清单、临时校验文档、实际 `HEAD` 三者一致。
- 文档不再引用参数化前的默认状态。

### 6.2 准备生产参数与首批用户文件

目标：把“仓库支持生产参数”推进到“部署方手里有可用参数集”。

任务：

- 基于 `.env.intranet.example` 生成目标环境 `.env`。
- 先明确目标环境入口方式：
  - 前后端同域，或
  - 前后端分域 + 反向代理 / HTTPS
- 生成强随机 `JWT_SECRET`，并确认不进入 git。
- 选定本次试点的 `APP_VERSION`，建议使用 tag 或 `0d7edf3` 这样的短 SHA。
- 复制 `scripts/users.example.json` 为目标环境用户文件，并替换全部 `change-me-*` 密码。
- 将 `.env` 中的 `BOOTSTRAP_USERS_FILE` 指向自定义用户文件。
- 按入口方式设置 `CORS_ALLOWED_ORIGINS`；如前后端同域，则写成实际入口域名即可。
- 通过 `docker compose config` 再次核对参数展开结果。

验收信号：

- 生产参数四项 `JWT_SECRET` / `APP_VERSION` / `BOOTSTRAP_USERS_FILE` / `CORS_ALLOWED_ORIGINS` 已全部有真实值。
- `docker compose config` 展开结果与预期一致。
- 默认 bootstrap 密码不再作为上线输入。

### 6.3 目标内网环境试点部署与验收

目标：把“本地已通过”升级成“目标环境可运行”。

任务：

- 按 `INTRANET_RUNBOOK.md` 在目标主机执行 `docker compose up -d --build`。
- 执行后端健康校验：
  - `curl -sf http://127.0.0.1:8080/readyz`
  - `curl -sf http://127.0.0.1:8080/health`
- 用 bootstrap owner 账号执行一次登录与 `GET /users?active=true`。
- 执行一次浏览器主路径点验：
  - owner 登录、创建、分配
  - assignee 登录、开始、提交
  - owner 审批、关闭、查看记录与审计
- 执行一次目标环境备份命令，确认生成新的 `.gz` 文件。

验收信号：

- 目标环境上服务健康、用户可登录、主流程可跑通。
- 目标环境的备份命令可执行。
- 验证记录可回填到发布清单。

### 6.4 移动端 / 窄屏与入口层收尾

目标：解决当前明确剩余的人工点验空白。

任务：

- 按 `reports/mobile-acceptance-checklist.md` 在 `375px` 与 `768px` 下至少各测一遍。
- 明确目标环境是否采用同域部署：
  - 若同域，则确认不需要额外 CORS 扩展。
  - 若分域，则确认前端域名已写入 `CORS_ALLOWED_ORIGINS`。
- 明确入口层策略：
  - 内网直连 `8080`，或
  - HTTPS / 反向代理前置。
- 记录防火墙放行范围与访问边界。

验收信号：

- 移动端清单关键项完成勾选。
- 入口层与访问边界有明确部署记录，而不只是建议。

### 6.5 go / no-go 判定与回滚入口

目标：让试点部署有明确的放行标准，而不是“感觉差不多可以上”。

任务：

- 在 `INTRANET_RELEASE_CHECKLIST.md` 回填本次试点的 commit、版本号、备份路径、执行人。
- 记录回滚路径：
  - 使用上一版镜像或上一版 commit 重新 `docker compose up -d --build`
  - 如有需要，使用最近一份 Mongo 备份恢复
- 把未完成事项归类为：
  - 阻断试点上线
  - 不阻断试点，但需进入第二期 backlog

验收信号：

- 有清晰的放行结论与回滚入口。
- 团队可以基于文档做试点决策，而不是依赖口头说明。

## 7. 优先级建议

1. P0：发布资料同步
   原因：这是最小成本的清障动作，先消除“文档和事实不一致”。
2. P1：生产参数落地
   原因：这是当前唯一真正阻断内网试点的核心项。
3. P2：目标环境试点验收
   原因：本地通过不等于目标环境通过，必须做一次真实落地验证。
4. P3：移动端与入口层收尾
   原因：这两项现在不是功能阻断，但会影响试点体验与交付完整度。
5. P4：第二期 backlog 梳理
   原因：应放在试点可行性确认之后，而不是提前扩 scope。

## 8. 输出物

提交到仓库：

- 当前计划文档
- 更新后的发布清单
- 如有需要，新增正式、受版本控制的校验记录

不提交到仓库：

- 真实 `.env`
- 自定义 bootstrap 用户文件
- 真实密码 / 密钥 / 证书信息

可选本地审计材料：

- 目标环境部署命令记录
- `docker compose config` 核对输出
- 移动端点验截图

## 9. go / no-go 判定标准

满足以下条件，才建议标记为“可试点部署”：

- `JWT_SECRET` 已替换为强随机值。
- `APP_VERSION` 已设为真实版本标识。
- `BOOTSTRAP_USERS_FILE` 已指向自定义文件，且默认密码全部替换。
- `CORS_ALLOWED_ORIGINS` 已与真实入口对齐。
- 目标环境健康检查、owner 登录、用户列表、主流程浏览器点验通过。
- 至少完成一次目标环境备份验证。

若以下任一项缺失，则建议保持“候选发布版”而非“可试点部署版”：

- 仍使用默认 bootstrap 密码
- 仍使用 compose 示例密钥
- 仍未确认入口域名 / 访问边界
- 仍未完成目标环境实际部署

## 10. 第二期 backlog 入口

只有在本计划完成后，才建议进入以下增强项：

- 用户管理后台
- 密码重置页面
- 任务列表分页 / 搜索
- 附件 / 通知
- Agent 子任务建议与结构化反馈模型

## 11. 参考

- `INTRANET_MVP.md`
- `INTRANET_RELEASE_CHECKLIST.md`
- `INTRANET_RUNBOOK.md`
- `INTRANET_OPS.md`
- `reports/intranet-release-assessment-2026-06-22.md`
- `reports/mobile-acceptance-checklist.md`
- `docker-compose.yml`
