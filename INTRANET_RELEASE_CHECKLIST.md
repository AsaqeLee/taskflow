# TaskFlow 内网发布检查清单

面向“这版是否可以作为内网 MVP 发布”的收口清单。具体部署步骤看 `INTRANET_RUNBOOK.md`，脚本细节看 `ACCEPTANCE_TESTING.md`，日常运维看 `INTRANET_OPS.md`。

## 1. 发布门槛

以下项目全部满足，才视为本次版本可发布：

- [ ] 已确认发布范围仍为 MVP：不含用户管理 UI、密码重置页、公开注册页、任务分页搜索
- [ ] `.env` 或部署环境已设置 `DEV_MODE=false`
- [ ] `JWT_SECRET` 已替换为强随机值
- [ ] `APP_VERSION` 已设置为明确版本号或 commit tag
- [ ] `scripts/users.example.json` 或自定义 bootstrap 用户文件已确认
- [ ] 本地后端验证通过：`go test ./...`
- [ ] 本地前端验证通过：`cd web && npm run lint && npm run test && npm run build`
- [ ] Compose smoke 通过：`bash scripts/compose_smoke.sh`
- [ ] 完整冷启动验收通过：`COLD_START=1 bash scripts/intranet_acceptance.sh`
- [ ] 至少完成一次浏览器人工点验：owner + assignee 两个账号走登录、创建、分配、提交、审批、关闭
- [ ] 已保留本次验收产物：日志、备份文件路径、版本号、执行人

## 2. 发布前检查

```bash
go test ./...
cd web && npm run lint && npm run test && npm run build
bash scripts/compose_smoke.sh
COLD_START=1 bash scripts/intranet_acceptance.sh
```

需要同时确认：

- `GET /readyz` 返回 `200`
- owner 可 `GET /users?active=true`
- 匿名 `POST /users` 返回 `401`
- 备份恢复后任务仍可读

## 3. 部署顺序

1. 准备 `.env`，确认 `JWT_SECRET`、`APP_VERSION`、`CORS_ALLOWED_ORIGINS`
2. 执行 `docker compose up -d --build`
3. 检查 `docker compose ps` 与 `curl -sf http://127.0.0.1:8080/readyz`
4. 用 bootstrap owner 账号执行一次登录验证
5. 按需启动前端演示：`cd web && npm run dev`，或托管 `web/dist`

## 4. 发布后人工点验

- [ ] owner 登录成功
- [ ] assignee 登录成功
- [ ] owner 可创建任务
- [ ] owner 可分配任务
- [ ] assignee 可 `start`
- [ ] assignee 可 `submit`
- [ ] owner 可 `approve`
- [ ] owner 可 `close`
- [ ] 详情页可查看 records 与 audit logs
- [ ] 服务重启后任务数据仍在

## 5. 发布记录

| 字段 | 填写 |
|------|------|
| 日期 | |
| 执行人 | |
| Git commit | |
| APP_VERSION | |
| 镜像 tag | |
| 验收结果 | |
| 备份文件路径 | |
| 已知风险 | |

## 6. 当前已知非目标

- 用户管理后台
- 自助密码重置 UI
- 公开注册 UI
- 分页、搜索、附件、通知
- Agent 子任务建议、结构化反馈 schema、多人协作模型
