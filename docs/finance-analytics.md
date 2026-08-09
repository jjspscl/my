# Finance derived analytics

Derived analytics compose the analytics core (spending summary, cash flow,
savings rate) with wallets and bills into higher-level views. Every response
carries named `assumptions` and human-readable `explanation` strings; nothing
returns an opaque score or a bare yes/no.

All endpoints live under `GET /finance/analytics/*` and require the session
cookie + CSRF header like the rest of the finance API.

## Endpoints

| Endpoint | Purpose |
| --- | --- |
| `GET /finance/analytics/anomalies?currency=PHP&months=6` | Spending anomalies per category (Hampel filter) |
| `GET /finance/analytics/recurring-charges?currency=PHP&months=6` | Recurring-charge summary classified against explicit bills |
| `GET /finance/analytics/bill-reconciliation?month=YYYY-MM` | Expected-vs-actual bill payments for a month |
| `GET /finance/analytics/emergency-fund?currency=PHP&targetMonths=0` | Liquid balance vs a target range of months of essential spending |
| `GET /finance/analytics/affordability?currency=PHP&amountCents=200000` | Runway model before/after a prospective purchase |
| `GET /finance/analytics/digest?month=YYYY-MM` | Composed monthly digest |

`month` defaults to the current month; `currency` is required everywhere it
appears.

## Spending anomalies (Hampel filter)

Monthly category spending is a time series, so anomalies are detected with a
local-window Hampel filter rather than a global modified z-score (the
Iglewicz–Hoaglin source explicitly warns the global score is not appropriate
for time series where local-window robust methods are preferred).

Parameters (published defaults):

- `MADScaleFactor = 1.4826` — scale factor mapping MAD to a normal sigma.
- `MeanADScaleFactor = 1.253314` — scale factor for mean absolute deviation
  (utility; not used by the anomaly path).
- `HampelNSigma = 3.0` — flag threshold.
- `HampelWindow = 5` — local window size (2 on each side).
- `MinAnomalyMonths = 6` — below this the report is returned with
  `sufficient: false` and must not be presented as definitive.

A zero-MAD window is a constant baseline and is never flagged (a ₱501 month
after months of ₱500 is not news). The report is per currency; months with no
spending are zero-filled.

## Recurring charges

A category is recurring when it has at least `RecurringMinOccurrences = 3`
charges across `RecurringMinDistinctMonths = 3` distinct months. Cadence beats
amount: amount similarity is not a membership requirement, only the
`tracked` / `amount_changed` classification.

Classification against explicit bills:

- `tracked` — a bill exists within tolerance.
- `amount_changed` — a bill exists but the median drifts beyond tolerance.
- `untracked` — no matching bill.

Tolerance is ±10% (`RecurringTolerancePct = 0.10`) or ±₱50
(`RecurringToleranceFloorCents = 50`), whichever is larger. 10% is calibrated
to skip normal utility fluctuation; the floor keeps a small subscription from
flipping status over a few cents.

A charge is never claimed to be unused — no usage data exists.

## Emergency fund

`GET /finance/analytics/emergency-fund` reports liquid balance (sum of wallet
balances per currency) against a target range of months of essential spending:

- Default target is 3–6 months (`EmergencyFundMinMonths`/`EmergencyFundMaxMonths`),
  the consensus range across CFPB, FINRA, Fidelity, and the St. Louis Fed.
  Reporting a range rather than a single number avoids implying advice.
- `targetMonths` overrides both ends of the range when non-zero (1–12).
- Monthly essential spend is the median over the last
  `EmergencyFundWindowMonths = 12` months in essential categories
  (`finance_categories.essential = 1`), zero-filled so months without
  essential spending count as low spend.
- `monthsOfRunway` = liquid / monthly essential spend.
- `shortfallToMinCents` / `shortfallToMaxCents` are the amounts needed to reach
  each end of the range (0 when already covered).

The endpoint refuses with `ErrInsufficientClassification` when more than
`MaxUnclassifiedShare = 20%` of spending in the currency is unclassified.

## Affordability

`GET /affordability` is a financial model, never a recommendation:

- Monthly obligation = median essential spend + bills due in the next 30 days
  (unpaid occurrences only).
- `runwayMonthsBefore` / `runwayMonthsAfter` = liquid balance / obligation,
  before and after the purchase.
- No yes/no is returned; the caller decides from the model and its named
  assumptions.

## Monthly digest

`GET /digest` composes the month's sections: cash flow, spending breakdown,
savings rate, recurring charges, anomalies, and emergency fund. Sections that
cannot be computed are omitted and named in `omitted` with the reason (e.g.
insufficient classification); the digest never fails wholesale over one
unavailable section. Windowed sections (recurring, anomalies, emergency) run
on the primary currency — the first currency in the month's cash flow.

## Classification guard

Analytics that depend on essential/discretionary classification
(emergency fund, affordability, spending breakdown) refuse with
`ErrInsufficientClassification` when more than 20% of spending is
unclassified, listing the top unclassified categories so the caller can tell
the user what to classify.