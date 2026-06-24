# TaskFlow 内网试点发布收口计划

面向“从当前候选发布版推进到目标内网可试点部署版”的短版执行跟踪。当前候选版本为 `dee3a20`，仓库内严格发布门禁已经通过；后续工作重点不在继续补功能，而在目标环境落地。

## 当前状态

- M1 已完成：发布清单、上线 Runbook、校验记录已同步到当前基线
- M2 待完成：目标环境真实 `.env`、真实 bootstrap 用户文件、真实 JWT 密钥、真实入口域名与 HTTPS
- M3 待完成：目标内网主机试点部署、健康检查、主流程点验、备份恢复演练
- M4 待完成：go / no-go 判定、业务签收、回滚入口确认

## 本轮目标

1. 把“仓库内可交付”推进到“目标环境可试点”
2. 明确哪些事项必须在仓库外完成
3. 避免继续在 repo 内堆重复文档和重复发布步骤

## 剩余执行项

### M2 准备目标环境输入

- 基于 `.env.intranet.example` 生成目标环境 `.env`
- 生成强随机 `JWT_SECRET`
- 选定 `APP_VERSION`
- 生成真实 bootstrap 用户文件，替换全部默认密码
- 明确前端入口方式：
  - 同域 nginx
  - 分域 + HTTPS / 反向代理
- 明确 `CORS_ALLOWED_ORIGINS`

完成标准：

- `bash scripts/validate_production_env.sh <target-env>` 通过
- 目标环境参数不再包含示例密钥、示例密码、示例用户文件

### M3 目标环境试点验收

- 按 `INTRANET_RUNBOOK.md` 部署目标环境
- 验证 `/readyz`
- 验证 owner 登录和用户列表
- 验证 owner + assignee 主流程
- 验证同域或 HTTPS 正式入口
- 完成一次备份创建和一次恢复演练
- 验证 Prometheus / Alertmanager 与真实告警路由

完成标准：

- 目标环境最小试点链路通过
- 有正式部署记录和恢复记录

### M4 上线判定

- 输出 go / no-go 结论
- 记录回滚镜像 / 回滚命令 / 回滚负责人
- 记录已知限制和后续 backlog

## 非目标

以下内容不属于本轮收口：

- 用户管理后台
- 自助密码重置 UI
- 公开注册 UI
- 分页、搜索、附件、通知
- 企业级 SSO / MFA / 多租户

## 当前文档边界

- `INTRANET_RELEASE_CHECKLIST.md`：当前事实和上线判定
- `INTRANET_RUNBOOK.md`：目标环境执行步骤
- `INTRANET_OPS.md`：上线后日常运维
- `reports/intranet-release-assessment-2026-06-23.md`：当前候选版本正式校验记录
