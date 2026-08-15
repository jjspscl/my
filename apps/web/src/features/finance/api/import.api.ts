import { apiClient } from '@/shared/api/client'
import { z } from 'zod'

import {
  CreateImportSchema,
  ImportBatchListSchema,
  ImportBatchSchema,
  type CreateImport,
  type ImportBatch,
} from '../schemas/import.schemas'

const ImportBatchDataSchema = z.object({ data: ImportBatchSchema })
const ImportBatchListDataSchema = ImportBatchListSchema
const RollbackResultSchema = z.object({
  data: z.object({ removedEntities: z.number().int() }),
})

/**
 * Import is a bulk, reviewed operation: it bypasses the offline mutation
 * queue on purpose. Queued replays of hundreds of rows would be noisy and the
 * server's fingerprint idempotency already makes retries safe.
 */
export async function createImport(data: CreateImport): Promise<ImportBatch> {
  const parsed = CreateImportSchema.parse(data)
  const res = await apiClient('/api/v1/finance/imports', ImportBatchDataSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
  return res.data
}

export async function listImports(): Promise<ImportBatch[]> {
  const res = await apiClient('/api/v1/finance/imports', ImportBatchListDataSchema)
  return res.data
}

export async function getImport(id: string): Promise<ImportBatch> {
  const res = await apiClient(`/api/v1/finance/imports/${id}`, ImportBatchDataSchema)
  return res.data
}

export async function rollbackImport(id: string): Promise<number> {
  const res = await apiClient(`/api/v1/finance/imports/${id}`, RollbackResultSchema, {
    method: 'DELETE',
  })
  return res.data.removedEntities
}
