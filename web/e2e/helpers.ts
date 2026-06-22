import { expect, type Locator, type Page } from '@playwright/test'

export const ownerId = process.env.OWNER_ID ?? 'u_owner'
export const ownerPassword = process.env.OWNER_PASSWORD ?? 'change-me-owner-123'
export const assigneeId = process.env.ASSIGNEE_ID ?? 'u_alice'
export const assigneePassword = process.env.ASSIGNEE_PASSWORD ?? 'change-me-alice-123'

export async function login(page: Page, userId: string, password: string) {
  await page.goto('/login')
  await expect(page.getByRole('heading', { name: '登录 TaskFlow' })).toBeVisible()
  await page.getByLabel('用户 ID').fill(userId)
  await page.getByLabel('密码').fill(password)
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page).toHaveURL(/\/tasks$/)
  await expect(page.getByRole('heading', { name: '任务列表' })).toBeVisible()
}

export async function logout(page: Page) {
  await page.getByRole('button', { name: '退出' }).click()
  await expect(page).toHaveURL(/\/login$/)
}

export async function expectNoHorizontalPageOverflow(page: Page) {
  const hasOverflow = await page.evaluate(() => {
    const tolerance = 1
    return document.documentElement.scrollWidth > window.innerWidth + tolerance
  })
  expect(hasOverflow).toBe(false)
}

export async function expectVisibleWithinViewport(locator: Locator, viewportWidth: number) {
  await expect(locator).toBeVisible()
  const box = await locator.boundingBox()
  expect(box).not.toBeNull()
  if (!box) {
    return
  }
  expect(box.x).toBeGreaterThanOrEqual(0)
  expect(box.x + box.width).toBeLessThanOrEqual(viewportWidth + 1)
}
