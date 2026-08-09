import { expect } from '@playwright/test'
import { MAILPIT_API_URL } from './env'

interface MailpitMessage {
  ID: string
  To: { Address: string; Name: string }[]
  Subject: string
}

interface MailpitMessageDetail {
  ID: string
  Text?: string
  HTML?: string
}

/**
 * Poll Mailpit for a magic link email sent to the given address.
 * Returns the full verify URL from the email body.
 */
export async function getMagicLinkUrl(email: string): Promise<string> {
  const maxAttempts = 20
  const pollIntervalMs = 1000

  for (let i = 0; i < maxAttempts; i++) {
    const resp = await fetch(`${MAILPIT_API_URL}/messages`)
    const data = (await resp.json()) as { messages: MailpitMessage[] }

    const msg = data.messages.find((m) =>
      m.To.some((t) => t.Address === email),
    )

    if (msg) {
      const detailResp = await fetch(`${MAILPIT_API_URL}/message/${msg.ID}`)
      const detail = (await detailResp.json()) as MailpitMessageDetail
      const body = detail.Text || detail.HTML || ''

      // Magic link URL format: http://localhost:5173/auth/verify?token=<token>
      const match = body.match(/(https?:\/\/[^\s"']+\/auth\/verify\?token=[^\s"']+)/)
      if (match) {
        return match[1]
      }
    }

    await new Promise((r) => setTimeout(r, pollIntervalMs))
  }

  throw new Error(
    `Magic link email not found for ${email} after ${maxAttempts * pollIntervalMs}ms`,
  )
}

/**
 * Delete all messages in Mailpit to get a clean state.
 */
export async function clearMailpitMessages(): Promise<void> {
  const resp = await fetch(`${MAILPIT_API_URL}/messages`, { method: 'DELETE' })
  expect(resp.ok).toBe(true)
}
