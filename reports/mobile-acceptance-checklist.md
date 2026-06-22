# 前端窄屏 / 移动端验收清单

在宽度 **375px**（iPhone SE）与 **768px**（平板）下各测一遍。使用 `web npm run dev` + 浏览器开发者工具设备模拟。

自动化补充：

- `npm --prefix web run acceptance:responsive`
- 该脚本覆盖登录、任务列表、新建任务、任务详情、分配区在窄屏下的基础可用性
- 仍建议保留一次人工 spot-check，确认真实浏览器下的滚动与点击体验

## 登录 `/login`

- [x] 表单不横向溢出（`acceptance:responsive` 2026-06-22）
- [~] 输入框可聚焦、键盘不遮挡提交按钮（自动化未覆盖，建议人工 spot-check）
- [~] 错误提示完整可见（自动化未覆盖）

## 任务列表 `/tasks`

- [x] 「新建任务」按钮可点击（`acceptance:responsive`）
- [~] 表格在小屏可横向滚动或换行可读（标题列优先）

## 新建任务 `/tasks/new`

- [x] 标题/描述输入可用（`acceptance:responsive`）
- [x] 创建成功后跳转详情（`acceptance:responsive`）

## 任务详情 `/tasks/:id`

- [~] 状态 Stepper 换行不重叠
- [x] **分配**：下拉与按钮在竖屏下纵向排列（`acceptance:responsive` 375px 断言）
- [x] 动作按钮不挤出屏幕（viewport 边界检查）
- [~] 删除/提交等 **弹窗** 居中、可滚动、可关闭
- [x] 记录/审计 Tab 可切换（`acceptance:responsive`）

## 权限提示

- [ ] 无可用操作时显示「当前状态下你没有可执行的操作」

## MVP 范围外（确认未实现）

- [ ] 无用户管理页
- [ ] 无密码重置页
- [ ] 无注册页（`ALLOW_PUBLIC_REGISTER=false`）

## 签署

| 角色 | 账号 | 日期 | 通过 |
|------|------|------|------|
| owner | u_owner | | [ ] |
| assignee | u_alice | | [ ] |
