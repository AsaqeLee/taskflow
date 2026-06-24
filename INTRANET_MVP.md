# TaskFlow 小团队内网 MVP 范围

面向 5–20 人内网团队的最小可用版本说明。这个文件只保留范围边界和完成定义，不再承担实施任务拆解。当前发布执行以 `INTRANET_RELEASE_CHECKLIST.md` 和 `INTRANET_RUNBOOK.md` 为准。

## MVP 目标

- 浏览器内完成登录、查看任务、创建任务、分配、启动、提交、驳回、审批、关闭
- 任务详情可查看协作记录和审计日志
- `DEV_MODE=false` 下可通过 Compose 启动并保留数据
- 有首批用户初始化流程
- 有 Mongo 备份与恢复路径

## 完成定义

以下条件满足，即视为内网 MVP 代码侧完成：

- [x] owner 能创建、分配、审批、驳回、关闭任务
- [x] assignee 能启动、提交任务
- [x] 浏览器主流程可用
- [x] 响应式核心页面可用
- [x] 同域 nginx 入口可用
- [x] 监控、备份、恢复、冷启动验收可通过
- [x] 严格发布门禁可通过

## 当前非目标

- 用户管理后台
- 自助密码重置 UI
- 公开注册 UI
- 分页、搜索、附件、通知
- 企业级 SSO / MFA / 多租户
- CD 编排和仓库外基础设施自动化

## 文档分工

- `INTRANET_RELEASE_CHECKLIST.md`：当前候选版本是否可发布
- `INTRANET_RUNBOOK.md`：如何在目标环境执行部署
- `INTRANET_OPS.md`：如何日常运维
- `ACCEPTANCE_TESTING.md`：如何跑验收脚本
