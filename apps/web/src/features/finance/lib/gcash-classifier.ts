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

const INCOME_RE =
  /\b(Received from|Cash In|Salary|Payroll|Refund|Disbursement|Remittance|Send Money received|Load Wallet|Transfer received|Top-?up|Rewards|Cashback|Interest)\b/i

const TRANSFER_RE =
  /\b(InstaPay|PESONet|Bank Transfer|Transfer to|Transferred|Move Money|Withdrawal to)\b/i

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
  { re: /\b(Received from|Cash In|Salary|Payroll|Disbursement|Remittance)\b/i, category: 'Income' },
]

/** Classify one statement row deterministically. */
export function classifyRow(row: GcashRow, ownedWalletNames: string[] = []): Classification {
  const text = `${row.description} ${row.referenceNo}`
  const isDebit = !!row.debit && !row.credit

  // Wallet-name match against known user wallets: strong transfer hint.
  const exactWallet = ownedWalletNames.find((name) => name && text.includes(name))

  let kind: DraftKind
  if (isDebit) {
    kind = TRANSFER_RE.test(text) || exactWallet ? 'transfer_out' : 'expense'
  } else {
    kind = INCOME_RE.test(text) ? 'income' : 'transfer_in'
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
    counterparty: walletHint?.wallet ?? exactWallet,
    transferWalletHint: walletHint?.wallet ?? exactWallet,
  }
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
