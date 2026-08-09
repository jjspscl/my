/**
 * E2E environment with local-development defaults. CI overrides all three.
 *
 * E2E_EMAIL must match the API's MY_USER_EMAIL: the backend silently
 * no-ops when the submitted email differs (no token, no email, non-
 * disclosing 200), which would make the magic-link helpers time out.
 */
export const TEST_EMAIL = process.env.E2E_EMAIL ?? 'jjspscl@gmail.com'

/** Mailpit HTTP API base. */
export const MAILPIT_API_URL =
  process.env.E2E_MAILPIT_URL ?? 'http://localhost:8025/api/v1'
