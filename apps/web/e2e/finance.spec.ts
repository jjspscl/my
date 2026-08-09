import { test, expect } from '@playwright/test'
import { login } from './helpers/auth'

const runId = Date.now()
let uid = 0
function unique(name: string): string {
  uid++
  return `${name} ${runId}-${uid}`
}

test.describe('Finance', () => {
  test.beforeEach(async ({ page }) => {
    await login(page)
  })

  test('create a wallet', async ({ page }) => {
    const walletName = unique('Test Wallet')
    await page.goto('/finance')

    // Go to Wallets tab
    await page.getByRole('tab', { name: 'Wallets' }).click()

    // Click Add Wallet
    await page.getByRole('button', { name: 'Add Wallet' }).click()

    // Fill wallet form
    await page.getByLabel('Name').fill(walletName)
    await page.getByLabel('Type').click()
    await page.getByRole('option', { name: 'E-Wallet' }).click()
    await page.getByLabel('Opening Balance').fill('1000')

    // Submit
    await page.getByRole('button', { name: 'Create Wallet' }).click()

    // Wait for wallet to appear in list
    await expect(page.getByText(walletName)).toBeVisible()
  })

  test('create a transaction', async ({ page }) => {
    const description = unique('Lunch E2E test')
    await page.goto('/finance')

    // Click Add Transaction button
    await page.getByRole('button', { name: 'Add Transaction' }).click()

    // Fill transaction form
    await page.getByLabel('Amount').fill('250.50')
    await page.getByLabel('Category').click()
    await page.getByPlaceholder('Search or add...').fill('Food')
    await page.getByRole('option').first().click()

    // Select wallet if available (scope to dialog to avoid tabpanel conflict)
    const dialog = page.getByRole('dialog')
    const walletSelect = dialog.getByRole('combobox', { name: 'Wallet' })
    if (await walletSelect.isVisible()) {
      await walletSelect.click()
      await page.getByRole('option').first().click()
    }

    // Fill description
    await page.getByLabel('Description').fill(description)

    // Submit
    await page.getByRole('button', { name: 'Add Expense' }).click()

    // Transaction should appear in table
    await expect(page.getByText(description)).toBeVisible()
    await expect(page.getByRole('cell', { name: /250\.50/ }).first()).toBeVisible()
  })

  test('create a savings goal', async ({ page }) => {
    const walletName = unique('Goal Wallet')
    const goalName = unique('Test Goal')
    await page.goto('/finance')

    // Create a wallet first so we have one to assign the goal to
    await page.getByRole('tab', { name: 'Wallets' }).click()
    await page.getByRole('button', { name: 'Add Wallet' }).click()
    await page.getByLabel('Name').fill(walletName)
    await page.getByLabel('Type').click()
    await page.getByRole('option', { name: 'Bank' }).click()
    await page.getByLabel('Opening Balance').fill('50000')
    await page.getByRole('button', { name: 'Create Wallet' }).click()
    await expect(page.getByText(walletName)).toBeVisible()
    // Wait for the dialog to fully detach before interacting with the tabs.
    await expect(page.getByRole('dialog')).toHaveCount(0)

    // Switch to Goals tab
    await page.getByRole('tab', { name: 'Goals' }).click()

    // Click Add Goal
    await page.getByRole('button', { name: 'Add Goal' }).click()

    // Fill goal form
    await page.getByLabel('Name').fill(goalName)
    await page.getByLabel('Target Amount').fill('50000')

    // Select the wallet from Select
    await page.getByRole('combobox', { name: 'Target Wallet' }).click()
    await page.getByRole('option', { name: walletName }).click()

    // Submit
    await page.getByRole('button', { name: 'Create Goal' }).click()

    // Verify it appears in the UI
    await expect(page.getByText(goalName)).toBeVisible()
  })
})
