import { expect, test } from '@playwright/test'
import {
  assigneeId,
  expectNoHorizontalPageOverflow,
  expectVisibleWithinViewport,
  login,
  logout,
  ownerId,
  ownerPassword,
} from './helpers'

const viewports = [
  { name: 'iphone-se', width: 375, height: 667 },
  { name: 'tablet', width: 768, height: 1024 },
]

for (const viewport of viewports) {
  test(`${viewport.name} core task pages remain usable`, async ({ page }) => {
    await page.setViewportSize({ width: viewport.width, height: viewport.height })

    const taskTitle = `Responsive acceptance ${viewport.name} ${Date.now()}`

    await login(page, ownerId, ownerPassword)
    await expectNoHorizontalPageOverflow(page)
    await expectVisibleWithinViewport(page.getByRole('link', { name: '新建任务' }), viewport.width)

    await page.getByRole('link', { name: '新建任务' }).click()
    await expect(page).toHaveURL(/\/tasks\/new$/)
    await expectNoHorizontalPageOverflow(page)
    await page.getByLabel('标题').fill(taskTitle)
    await page.getByLabel('描述').fill(`responsive flow ${viewport.name}`)
    await expectVisibleWithinViewport(page.getByRole('button', { name: '创建' }), viewport.width)
    await page.getByRole('button', { name: '创建' }).click()

    await expect(page.getByRole('heading', { name: taskTitle })).toBeVisible()
    await expectNoHorizontalPageOverflow(page)

    const assigneeSelect = page.getByLabel('选择执行人')
    const assignButton = page.getByRole('button', { name: '分配' })
    await expectVisibleWithinViewport(assigneeSelect, viewport.width)
    await expectVisibleWithinViewport(assignButton, viewport.width)

    if (viewport.width === 375) {
      const selectBox = await assigneeSelect.boundingBox()
      const buttonBox = await assignButton.boundingBox()
      expect(selectBox).not.toBeNull()
      expect(buttonBox).not.toBeNull()
      if (selectBox && buttonBox) {
        expect(buttonBox.y).toBeGreaterThanOrEqual(selectBox.y + selectBox.height - 1)
      }
    }

    await assigneeSelect.selectOption(assigneeId)
    await assignButton.click()
    await expect(page.getByText('已分配').first()).toBeVisible()
    await expectNoHorizontalPageOverflow(page)

    await page.getByRole('button', { name: '记录' }).click()
    await expectNoHorizontalPageOverflow(page)
    await page.getByRole('button', { name: '审计' }).click()
    await expectNoHorizontalPageOverflow(page)

    await logout(page)
  })
}
