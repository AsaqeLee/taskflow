import { expect, test, type Page } from '@playwright/test'

const ownerId = process.env.OWNER_ID ?? 'u_owner'
const ownerPassword = process.env.OWNER_PASSWORD ?? 'change-me-owner-123'
const assigneeId = process.env.ASSIGNEE_ID ?? 'u_alice'
const assigneePassword = process.env.ASSIGNEE_PASSWORD ?? 'change-me-alice-123'

test('owner and assignee complete the browser workflow', async ({ page }) => {
  const taskTitle = `Browser acceptance ${Date.now()}`

  await login(page, ownerId, ownerPassword)

  await page.getByRole('link', { name: '新建任务' }).click()
  await expect(page).toHaveURL(/\/tasks\/new$/)
  await page.getByLabel('标题').fill(taskTitle)
  await page.getByLabel('描述').fill('browser acceptance flow')
  await page.getByRole('button', { name: '创建' }).click()
  await expect(page.getByRole('heading', { name: taskTitle })).toBeVisible()
  const taskUrl = page.url()

  await page.getByLabel('选择执行人').selectOption(assigneeId)
  await page.getByRole('button', { name: '分配' }).click()
  await expect(page.getByText('已分配').first()).toBeVisible()

  await logout(page)

  await login(page, assigneeId, assigneePassword)
  await page.goto(taskUrl)
  await expect(page.getByRole('heading', { name: taskTitle })).toBeVisible()
  await page.getByRole('button', { name: '开始' }).click()
  await expect(page.getByText('进行中').first()).toBeVisible()
  await page.getByRole('button', { name: '提交' }).click()
  await page.getByPlaceholder('填写说明（必填）').fill('browser acceptance submitted')
  await page.getByRole('button', { name: '确认' }).click()
  await expect(page.getByText('已提交').first()).toBeVisible()
  await page.getByRole('button', { name: '记录' }).click()
  await expect(page.getByText('browser acceptance submitted')).toBeVisible()

  await logout(page)

  await login(page, ownerId, ownerPassword)
  await page.goto(taskUrl)
  await expect(page.getByRole('heading', { name: taskTitle })).toBeVisible()
  await page.getByRole('button', { name: '审批通过' }).click()
  await page.getByPlaceholder('填写说明（必填）').fill('browser acceptance approved')
  await page.getByPlaceholder('补充审核意见（可选）').fill('reviewed in browser')
  await page.getByRole('button', { name: '确认' }).click()
  await expect(page.getByText('已审批').first()).toBeVisible()
  await page.getByRole('button', { name: '关闭' }).click()
  await expect(page.getByText('已完成').first()).toBeVisible()

  await page.getByRole('button', { name: '记录' }).click()
  await expect(page.getByText('reviewed in browser')).toBeVisible()
  await page.getByRole('button', { name: '审计' }).click()
  await expect(page.getByText('task_approved')).toBeVisible()
})

async function login(page: Page, userId: string, password: string) {
  await page.goto('/login')
  await expect(page.getByRole('heading', { name: '登录 TaskFlow' })).toBeVisible()
  await page.getByLabel('用户 ID').fill(userId)
  await page.getByLabel('密码').fill(password)
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page).toHaveURL(/\/tasks$/)
  await expect(page.getByRole('heading', { name: '任务列表' })).toBeVisible()
}

async function logout(page: Page) {
  await page.getByRole('button', { name: '退出' }).click()
  await expect(page).toHaveURL(/\/login$/)
}
