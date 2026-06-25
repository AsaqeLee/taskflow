# TaskFlow 内网首次上线 Runbook

面向第一次在内网主机部署 TaskFlow 的运维 / 开发同事。预计耗时 30-60 分钟。

适用前提：

- 候选版本已经按 `INTRANET_RELEASE_CHECKLIST.md` 完成仓库内门禁
- 当前工作目标是把候选版本落到目标环境，而不是继续补仓库内功能

## 前置条件

- Docker 和 Docker Compose 可用
- 目标主机可访问 Mongo、反向代理或同机 compose 组件
- 已克隆本仓库
- 已明确是否走同域入口 `--profile full`

## 步骤 1：准备环境变量

推荐方式：

```bash
bash scripts/init_pilot_env.sh
```

该脚本会生成：

- `.env`
- `scripts/users.intranet.json`
- `evidence/pilot-credentials.txt`

手工方式：

```bash
cp .env.intranet.example .env
cp scripts/users.example.json scripts/users.intranet.json
```

至少修改这些字段：

```env
JWT_SECRET=<openssl rand -hex 32 的输出>
APP_VERSION=<git tag 或 short SHA>
BOOTSTRAP_USERS_FILE=./scripts/users.intranet.json
STRICT_PRODUCTION_CONFIG=true
```

如果前后端分离部署，再设置：

```env
CORS_ALLOWED_ORIGINS=https://taskflow.internal
```

注意：

- `scripts/users.intranet.json` 中每个默认密码都必须替换
- bootstrap 密码只能视为首次登录的临时密码

## 步骤 2：校验生产型参数

```bash
bash scripts/validate_production_env.sh .env
```

若失败，先修正 `.env`，不要带着占位值继续部署。

## 步骤 3：启动服务

后端和 Mongo 基线：

```bash
docker compose up -d --build
```

如果需要同域 Web 入口：

```bash
docker compose --profile full up -d --build web
```

## 步骤 4：检查健康状态

```bash
docker compose ps
curl -sf http://127.0.0.1:8080/readyz
curl -sf http://127.0.0.1:8080/metrics | head
```

如启用同域入口，再检查：

```bash
curl -sf http://127.0.0.1:8081/
```

## 步骤 5：执行最小验收

至少执行：

```bash
bash scripts/compose_smoke.sh
```

如使用同域入口，补跑：

```bash
bash scripts/nginx_smoke.sh
```

如需要浏览器侧确认，补跑：

```bash
bash scripts/web_acceptance_smoke.sh
```

## 步骤 6：验证主流程

在目标环境确认这些动作可用：

- owner 登录
- 创建任务
- 分配任务
- assignee 启动并提交
- owner 审批 / 驳回 / 关闭
- 任务详情可查看协作记录和审计日志

## 步骤 7：做一次备份恢复演练

推荐直接按仓库既有脚本和运维说明执行，并记录：

- 备份时间
- 备份命令
- 恢复命令
- 恢复后校验结果

更完整的日常运维条目见 `INTRANET_OPS.md`，验收脚本说明见 `ACCEPTANCE_TESTING.md`。

## 回滚入口

回滚前先停止向新版本送流量，再执行：

```bash
TASKFLOW_PREVIOUS_IMAGE=taskflow:previous \
TASKFLOW_ENV_FILE=.env.production \
TASKFLOW_CONTAINER_NAME=taskflow \
TASKFLOW_PORT=8080 \
./scripts/rollback_image.sh
```

回滚后再次检查：

- `/readyz`
- 登录与核心任务流
- 审计日志和请求 ID / trace ID

## 故障排查入口

| 现象 | 第一步 |
| --- | --- |
| `/readyz` 非 200 | `docker compose logs taskflow mongo` |
| 登录 401 | 检查 bootstrap 是否成功，`docker compose logs bootstrap` |
| CORS 错误 | 检查 `CORS_ALLOWED_ORIGINS` 与浏览器 origin |
| 5xx | 用响应中的 `request_id` 对照 taskflow 日志 |
| Prometheus 无数据 | `docker compose --profile monitoring logs prometheus alertmanager` |

## MVP 范围提醒

本阶段仍不包含：

- 用户管理 UI
- 自助密码重置页
- 公开注册页
- 任务分页搜索
