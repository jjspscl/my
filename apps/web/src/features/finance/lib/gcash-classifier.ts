/**
 * Deterministic first-pass classification for GCash statement rows.
 *
 * Gives the review step sensible defaults before any LLM analysis: income vs
 * expense vs probable own-wallet transfer, plus a category guess. The LLM
 * analysis layer (intelligence context) can refine these later.
 */

import type { GcashRow } from './gcash-text'

export type DraftKind = 'expense' | 'income' | 'transfer_out' | 'transfer_in'

export interface Classification {
  kind: DraftKind
  category: string
  /** Normalized counterparty name, when recognizable. */
  counterparty?: string
  /** Transfer candidate matched to a wallet name pattern. */
  transferWalletHint?: string
}

const WALLET_HINTS: Array<{ pattern: RegExp; wallet: string }> = [
  { pattern: /\bBDO\b|BDO Unibank/i, wallet: 'BDO' },
  { pattern: /\bBPI\b/i, wallet: 'BPI' },
  { pattern: /\bMaya\b|PayMaya/i, wallet: 'Maya' },
  { pattern: /\bShopeePay\b/i, wallet: 'ShopeePay' },
  { pattern: /\bUnionBank\b/i, wallet: 'UnionBank' },
  { pattern: /\bRCBC\b/i, wallet: 'RCBC' },
  { pattern: /\bMetrobank\b|Metro Bank/i, wallet: 'Metrobank' },
  { pattern: /\bPNB\b/i, wallet: 'PNB' },
  { pattern: /\bCoins\.ph\b/i, wallet: 'Coins.ph' },
  { pattern: /\bGrabPay\b|Grab Pay/i, wallet: 'GrabPay' },
  { pattern: /\bSecurity Bank\b/i, wallet: 'Security Bank' },
  { pattern: /\bEastWest\b|East West/i, wallet: 'EastWest' },
  { pattern: /\bLandBank\b|Land Bank/i, wallet: 'LandBank' },
  { pattern: /\bGCash to GCash\b|\bG2G\b/i, wallet: 'GCash' },
]

const CATEGORY_HINTS: Array<{ re: RegExp; category: string }> = [
  { re: /\b(Jollibee|McDonald|McDo|KFC|Chowking|Greenwich|Mang Inasal|Foodpanda|GrabFood|Shakey|Pizza|Wendy|Subway|Starbucks|Coffee)\b/i, category: 'Food' },
  { re: /\b(Grab|Angkas|JoyRide|Move It|Lalamove|Transport|Fare|Gas|Petron|Shell|Caltex|Unioil)\b/i, category: 'Transport' },
  { re: /\b(Lazada|Shopee|Zalora|Amazon|Shein|Temu|Checkout|Supermarket|Grocery|Puregold|Landers|Robinsons|SM Supermarket|Savemore)\b/i, category: 'Shopping' },
  { re: /\b(Meralco|Manila Water|Maynilad|PLDT|Globe|Smart|DITO|Internet|Postpaid|Prepaid|Electric|Water bill|Utilities)\b/i, category: 'Utilities' },
  { re: /\b(Load|Giga|Data|Unli|Promo)\b/i, category: 'Load & Data' },
  { re: /\b(Rent|Apartment|Condominium|HOA)\b/i, category: 'Rent' },
  { re: /\b(Insurance|SSS|PhilHealth|Pag-?IBIG)\b/i, category: 'Insurance' },
  { re: /\b(Hospital|Clinic|Pharmacy|Mercury|Watsons|Doctor|Medical|Medicine)\b/i, category: 'Health' },
  { re: /\b(Netflix|Spotify|Disney|YouTube|Streaming|Prime|Subscrib)\b/i, category: 'Subscriptions' },
  { re: /\b(Received from|Received GCash from|Cash In|Salary|Payroll|Disbursement|Remittance|Refund|Reversal|Cashback|Rewards|Money received)\b/i, category: 'Income' },
]

/**
 * Classify one statement row deterministically.
 *
 * Transfers are ONLY claimed when the description matches a wallet the user
 * owns (case-insensitive, unique match). Everything else defaults to the
 * safer income/expense kinds: an unmapped "transfer-looking" row can still
 * be reclassified in review, but it must never arrive at the API without a
 * counter wallet.
 */
export function classifyRow(row: GcashRow, ownedWalletNames: string[] = []): Classification {
  const text = `${row.description} ${row.referenceNo}`
  const isDebit = !!row.debit && !row.credit

  // Case-insensitive, unique owned-wallet match → transfer candidate.
  const owned = findOwnedWallet(text, ownedWalletNames)

  let kind: DraftKind
  if (isDebit) {
    kind = owned ? 'transfer_out' : 'expense'
  } else {
    kind = owned ? 'transfer_in' : 'income'
  }

  // Counterparty hint from wallet patterns in the description.
  const walletHint = WALLET_HINTS.find((h) => h.pattern.test(text))

  let category = 'Unclassified'
  for (const hint of CATEGORY_HINTS) {
    if (hint.re.test(text)) {
      category = hint.category
      break
    }
  }
  if (kind === 'transfer_out' || kind === 'transfer_in') {
    category = 'Transfer'
  }
  if (kind === 'income' && category === 'Unclassified') {
    category = 'Income'
  }

  return {
    kind,
    category,
    counterparty: walletHint?.wallet ?? owned,
    transferWalletHint: owned ?? walletHint?.wallet,
  }
}

// findOwnedWallet returns the wallet name when EXACTLY ONE owned wallet name
// appears in the text. Ambiguous mentions (or none) return undefined: a
// generic bank/e-wallet word is not proof of ownership.
function findOwnedWallet(text: string, names: string[]): string | undefined {
  const lower = text.toLowerCase()
  const matches = names.filter((n) => n && lower.includes(n.toLowerCase()))
  return matches.length === 1 ? matches[0] : undefined
}

/** Group rows by normalized description for the transfer-mapping review. */
export function groupCounterparties(rows: Array<{ description: string }>): Map<string, number> {
  const groups = new Map<string, number>()
  for (const row of rows) {
    const key = row.description.trim() || '(no description)'
    groups.set(key, (groups.get(key) ?? 0) + 1)
  }
  return groups
}
