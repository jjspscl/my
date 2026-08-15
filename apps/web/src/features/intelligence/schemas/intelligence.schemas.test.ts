import { describe, expect, it } from 'vitest'

import {
  ConnectorAuthSchema,
  ConnectorKindSchema,
  ConnectorSchema,
  CreateConnectorSchema,
} from './intelligence.schemas'

describe('connector schemas', () => {
  it('parses a native provider connector response', () => {
    const parsed = ConnectorSchema.parse({
      id: 'c1',
      name: 'Tavily',
      kind: 'tavily',
      endpoint: '',
      authType: 'bearer',
      enabled: true,
      allowlist: ['tavily-search'],
      timeoutMs: 15000,
      hasToken: true,
      createdAt: '2026-08-15T00:00:00Z',
      updatedAt: '2026-08-15T00:00:00Z',
    })
    expect(parsed.kind).toBe('tavily')
    expect(parsed.authType).toBe('bearer')
  })

  it('rejects unknown kinds and auth types', () => {
    expect(ConnectorKindSchema.safeParse('exa').success).toBe(true)
    expect(ConnectorKindSchema.safeParse('serper').success).toBe(false)
    expect(ConnectorAuthSchema.safeParse('x-api-key').success).toBe(true)
    expect(ConnectorAuthSchema.safeParse('query').success).toBe(false)
  })

  it('accepts native create payloads without an endpoint', () => {
    const parsed = CreateConnectorSchema.parse({
      name: 'Tavily',
      kind: 'tavily',
      allowlist: ['tavily-search'],
      token: 'tvly-x',
    })
    expect(parsed.endpoint).toBeUndefined()
  })

  it('accepts custom MCP create payloads with endpoint and auth', () => {
    const parsed = CreateConnectorSchema.parse({
      name: 'Self-hosted',
      kind: 'custom_mcp',
      endpoint: 'https://mcp.example.com/mcp',
      authType: 'x-api-key',
      allowlist: ['brave_web_search'],
      token: 'secret',
    })
    expect(parsed.kind).toBe('custom_mcp')
    expect(parsed.authType).toBe('x-api-key')
  })

  it('allows keyless Exa (no token)', () => {
    const parsed = CreateConnectorSchema.parse({
      name: 'Exa',
      kind: 'exa',
      allowlist: ['web_search_exa'],
    })
    expect(parsed.token).toBeUndefined()
  })
})
