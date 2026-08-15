import { z } from 'zod'

// ---- Intelligence settings ----

export const ProviderProfileSchema = z.object({
  id: z.string(),
  name: z.string(),
  providerType: z.enum(['openai', 'openai_compatible', 'ollama', 'codex_cli']),
  baseUrl: z.string().optional(),
  model: z.string(),
  enabled: z.boolean(),
  priority: z.number().int(),
  maxTokens: z.number().int().optional(),
  timeoutMs: z.number().int(),
  allowLocal: z.boolean(),
  hasCredential: z.boolean(),
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type ProviderProfile = z.infer<typeof ProviderProfileSchema>

export const CreateProviderSchema = z.object({
  name: z.string().min(1),
  providerType: z.enum(['openai', 'openai_compatible', 'ollama', 'codex_cli']),
  baseUrl: z.string().optional(),
  model: z.string().min(1),
  maxTokens: z.number().int().optional(),
  timeoutMs: z.number().int().optional(),
  allowLocal: z.boolean().optional(),
  apiKey: z.string().optional(),
})
export type CreateProvider = z.infer<typeof CreateProviderSchema>

export const UpdateProviderSchema = z.object({
  name: z.string().min(1),
  providerType: z.enum(['openai', 'openai_compatible', 'ollama', 'codex_cli']),
  baseUrl: z.string().optional(),
  model: z.string().min(1),
  maxTokens: z.number().int().optional(),
  timeoutMs: z.number().int().optional(),
  allowLocal: z.boolean().optional(),
  enabled: z.boolean(),
})
export type UpdateProvider = z.infer<typeof UpdateProviderSchema>

export const ConnectorKindSchema = z.enum(['tavily', 'brave', 'exa', 'custom_mcp'])
export type ConnectorKind = z.infer<typeof ConnectorKindSchema>

export const ConnectorAuthSchema = z.enum(['none', 'bearer', 'x-api-key'])
export type ConnectorAuth = z.infer<typeof ConnectorAuthSchema>

export const ConnectorSchema = z.object({
  id: z.string(),
  name: z.string(),
  kind: ConnectorKindSchema,
  endpoint: z.string(),
  authType: ConnectorAuthSchema,
  enabled: z.boolean(),
  allowlist: z.array(z.string()),
  timeoutMs: z.number().int(),
  hasToken: z.boolean(),
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type Connector = z.infer<typeof ConnectorSchema>

export const CreateConnectorSchema = z.object({
  name: z.string().min(1),
  kind: ConnectorKindSchema,
  endpoint: z.string().optional(),
  authType: ConnectorAuthSchema.optional(),
  allowlist: z.array(z.string()).min(1),
  timeoutMs: z.number().int().optional(),
  token: z.string().optional(),
})
export type CreateConnector = z.infer<typeof CreateConnectorSchema>

export const UpdateConnectorSchema = z.object({
  name: z.string().min(1),
  kind: ConnectorKindSchema,
  endpoint: z.string().optional(),
  authType: ConnectorAuthSchema.optional(),
  allowlist: z.array(z.string()).min(1),
  timeoutMs: z.number().int().optional(),
  enabled: z.boolean(),
})
export type UpdateConnector = z.infer<typeof UpdateConnectorSchema>

// ---- Analysis ----

export const SuggestionFieldSchema = z.enum(['category', 'merchant', 'transfer', 'relationship'])
export type SuggestionField = z.infer<typeof SuggestionFieldSchema>

export const SuggestionSchema = z.object({
  id: z.string(),
  runId: z.string(),
  targetKey: z.string(),
  field: SuggestionFieldSchema,
  value: z.string(),
  confidence: z.number().min(0).max(1),
  status: z.string(),
  rationale: z.string().optional(),
  evidence: z.array(
    z.object({
      source: z.string(),
      detail: z.string(),
    }),
  ),
})
export type Suggestion = z.infer<typeof SuggestionSchema>

export const AnalysisRunSchema = z.object({
  id: z.string(),
  scope: z.string(),
  scopeId: z.string(),
  status: z.string(),
  model: z.string().optional(),
  error: z.string().optional(),
  createdAt: z.string(),
  completedAt: z.string().optional(),
})
export type AnalysisRun = z.infer<typeof AnalysisRunSchema>

export const AnalysisResultSchema = z.object({
  run: AnalysisRunSchema,
  suggestions: z.array(SuggestionSchema),
})
export type AnalysisResult = z.infer<typeof AnalysisResultSchema>

export const CreateAnalysisSchema = z.object({
  scopeId: z.string().min(1),
  rows: z
    .array(
      z.object({
        sourceReference: z.string(),
        description: z.string(),
        amountCents: z.number().int().positive(),
        kind: z.enum(['expense', 'income', 'transfer_out', 'transfer_in']),
      }),
    )
    .min(1)
    .max(50),
})
export type CreateAnalysis = z.infer<typeof CreateAnalysisSchema>

// Shared UI buckets for confidence.
export const CONFIDENCE_PRESELECT = 0.9
export const CONFIDENCE_REVIEW = 0.6

export function confidenceBucket(score: number): 'preselect' | 'review' | 'unresolved' {
  if (score >= CONFIDENCE_PRESELECT) return 'preselect'
  if (score >= CONFIDENCE_REVIEW) return 'review'
  return 'unresolved'
}
