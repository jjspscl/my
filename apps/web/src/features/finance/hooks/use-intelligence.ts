import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import {
  createAnalysis,
  createConnector,
  createProvider,
  deleteConnector,
  deleteProvider,
  listConnectors,
  listProviders,
  saveConnectorCredential,
  saveProviderCredential,
  testConnector,
  testProvider,
  updateConnector,
  updateProvider,
} from '../api/intelligence.api'
import type {
  AnalysisResult,
  CreateAnalysis,
  CreateConnector,
  CreateProvider,
  UpdateConnector,
  UpdateProvider,
} from '../schemas/intelligence.schemas'

const intelligenceKeys = {
  all: ['intelligence'] as const,
  providers: () => [...intelligenceKeys.all, 'providers'] as const,
  connectors: () => [...intelligenceKeys.all, 'connectors'] as const,
}

// ---- providers ----

export function useProviders() {
  return useQuery({
    queryKey: intelligenceKeys.providers(),
    queryFn: listProviders,
    staleTime: 1000 * 30,
  })
}

export function useCreateProvider() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateProvider) => createProvider(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: intelligenceKeys.providers() }),
  })
}

export function useUpdateProvider() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateProvider }) => updateProvider(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: intelligenceKeys.providers() }),
  })
}

export function useDeleteProvider() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteProvider(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: intelligenceKeys.providers() }),
  })
}

export function useSaveProviderCredential() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, value }: { id: string; value: string }) => saveProviderCredential(id, value),
    onSuccess: () => qc.invalidateQueries({ queryKey: intelligenceKeys.providers() }),
  })
}

export function useTestProvider() {
  return useMutation({
    mutationFn: (id: string) => testProvider(id),
    onSuccess: () => toast.success('Provider connection works.'),
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Provider test failed.'),
  })
}

// ---- connectors ----

export function useConnectors() {
  return useQuery({
    queryKey: intelligenceKeys.connectors(),
    queryFn: listConnectors,
    staleTime: 1000 * 30,
  })
}

export function useCreateConnector() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateConnector) => createConnector(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: intelligenceKeys.connectors() }),
  })
}

export function useUpdateConnector() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateConnector }) => updateConnector(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: intelligenceKeys.connectors() }),
  })
}

export function useDeleteConnector() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteConnector(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: intelligenceKeys.connectors() }),
  })
}

export function useSaveConnectorCredential() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, value }: { id: string; value: string }) => saveConnectorCredential(id, value),
    onSuccess: () => qc.invalidateQueries({ queryKey: intelligenceKeys.connectors() }),
  })
}

export function useTestConnector() {
  return useMutation({
    mutationFn: (id: string) => testConnector(id),
    onSuccess: () => toast.success('Connector reachable.'),
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Connector test failed.'),
  })
}

// ---- analysis ----

export function useRunAnalysis() {
  return useMutation({
    mutationFn: (data: CreateAnalysis) => createAnalysis(data),
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'AI analysis failed.')
    },
  })
}

export type { AnalysisResult }
