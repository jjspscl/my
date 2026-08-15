import { apiClient } from '@/shared/api/client'
import { z } from 'zod'

import {
  AnalysisResultSchema,
  ConnectorSchema,
  CreateAnalysisSchema,
  CreateConnectorSchema,
  CreateProviderSchema,
  ProviderProfileSchema,
  UpdateConnectorSchema,
  UpdateProviderSchema,
  type AnalysisResult,
  type Connector,
  type CreateAnalysis,
  type CreateConnector,
  type CreateProvider,
  type ProviderProfile,
  type UpdateConnector,
  type UpdateProvider,
} from '../schemas/intelligence.schemas'

const okResp = z.object({ ok: z.boolean().optional() })

// ---- providers ----

const ProviderListSchema = z.object({ data: z.array(ProviderProfileSchema) })
const ProviderIdSchema = z.object({ data: z.string() })
const ConnectorListSchema = z.object({ data: z.array(ConnectorSchema) })

export async function listProviders(): Promise<ProviderProfile[]> {
  const res = await apiClient('/api/v1/intelligence/providers', ProviderListSchema)
  return res.data
}

export async function createProvider(data: CreateProvider): Promise<string> {
  const parsed = CreateProviderSchema.parse(data)
  const res = await apiClient('/api/v1/intelligence/providers', ProviderIdSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
  return res.data
}

export async function updateProvider(id: string, data: UpdateProvider): Promise<string> {
  const parsed = UpdateProviderSchema.parse(data)
  const res = await apiClient(`/api/v1/intelligence/providers/${id}`, ProviderIdSchema, {
    method: 'PUT',
    body: JSON.stringify(parsed),
  })
  return res.data
}

export async function deleteProvider(id: string): Promise<void> {
  await apiClient(`/api/v1/intelligence/providers/${id}`, okResp, { method: 'DELETE' })
}

export async function saveProviderCredential(id: string, value: string): Promise<void> {
  await apiClient(`/api/v1/intelligence/providers/${id}/credential`, okResp, {
    method: 'PUT',
    body: JSON.stringify({ value }),
  })
}

export async function testProvider(id: string): Promise<void> {
  await apiClient(`/api/v1/intelligence/providers/${id}/test`, okResp, { method: 'POST' })
}

// ---- connectors ----

export async function listConnectors(): Promise<Connector[]> {
  const res = await apiClient('/api/v1/intelligence/connectors', ConnectorListSchema)
  return res.data
}

export async function createConnector(data: CreateConnector): Promise<string> {
  const parsed = CreateConnectorSchema.parse(data)
  const res = await apiClient('/api/v1/intelligence/connectors', ProviderIdSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
  return res.data
}

export async function updateConnector(id: string, data: UpdateConnector): Promise<string> {
  const parsed = UpdateConnectorSchema.parse(data)
  const res = await apiClient(`/api/v1/intelligence/connectors/${id}`, ProviderIdSchema, {
    method: 'PUT',
    body: JSON.stringify(parsed),
  })
  return res.data
}

export async function deleteConnector(id: string): Promise<void> {
  await apiClient(`/api/v1/intelligence/connectors/${id}`, okResp, { method: 'DELETE' })
}

export async function saveConnectorCredential(id: string, value: string): Promise<void> {
  await apiClient(`/api/v1/intelligence/connectors/${id}/credential`, okResp, {
    method: 'PUT',
    body: JSON.stringify({ value }),
  })
}

export async function testConnector(id: string): Promise<void> {
  await apiClient(`/api/v1/intelligence/connectors/${id}/test`, okResp, { method: 'POST' })
}

// ---- analysis ----

const AnalysisResultDataSchema = z.object({ data: AnalysisResultSchema })

export async function createAnalysis(data: CreateAnalysis): Promise<AnalysisResult> {
  const parsed = CreateAnalysisSchema.parse(data)
  const res = await apiClient('/api/v1/finance/imports/analyses', AnalysisResultDataSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
  return res.data
}

export async function getAnalysis(id: string): Promise<AnalysisResult> {
  const res = await apiClient(`/api/v1/finance/imports/analyses/${id}`, AnalysisResultDataSchema)
  return res.data
}

export async function cancelAnalysis(id: string): Promise<void> {
  await apiClient(`/api/v1/finance/imports/analyses/${id}`, okResp, { method: 'DELETE' })
}
