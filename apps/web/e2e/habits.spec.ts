import { test, expect } from '@playwright/test'
import { login } from './helpers/auth'

test.describe('Habits', () => {
  test.beforeEach(async ({ page }) => {
    await login(page)
  })

  test('create a daily habit', async ({ page }) => {
    await page.goto('/habits')

    // Click Add Habit
    await page.getByRole('button', { name: 'Add Habit' }).click()

    // Fill name
    await page.getByLabel('Name').fill('Read 20 pages')

    // Color picker — click the third color swatch
    const colorButtons = page.locator('button[class*="rounded-full"]')
    await colorButtons.nth(2).click()

    // Frequency defaults to Daily — leave it
    // Submit
    await page.getByRole('button', { name: 'Create Habit' }).click()

    // Habit should appear in list
    await expect(page.getByText('Read 20 pages').first()).toBeVisible()
  })

  test('create a weekly habit with target', async ({ page }) => {
    await page.goto('/habits')

    await page.getByRole('button', { name: 'Add Habit' }).click()
    await page.getByLabel('Name').fill('Gym workout')

    // Switch to weekly frequency
    await page.getByLabel('Frequency').click()
    await page.getByRole('option', { name: 'Weekly' }).click()

    // Set target per week
    await page.getByLabel('Times/week').fill('3')

    await page.getByRole('button', { name: 'Create Habit' }).click()

    await expect(page.getByText('Gym workout').first()).toBeVisible()
  })

  test('toggle habit completion', async ({ page }) => {
    await page.goto('/habits')

    // First create a habit
    await page.getByRole('button', { name: 'Add Habit' }).click()
    await page.getByLabel('Name').fill('Meditate')
    await page.getByRole('button', { name: 'Create Habit' }).click()
    await expect(page.getByText('Meditate').first()).toBeVisible()

    // Toggle it — find the check button in the habit card
    // Note: getByText finds the <p> inside the name div. Go up 2 levels to reach the card container.
    const habitCard = page.getByText('Meditate').first().locator('..').locator('..')
    const toggleButton = habitCard.locator('button[class*="h-8 w-8"]')
    await toggleButton.click()

    // Button should show filled (completed) state after toggle
    await expect(toggleButton).toHaveClass(/bg-foreground/)
  })
})
