import { useState } from 'react'
import { Bot, Loader2, Plus, Trash2, Zap } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import {
  useConnectors,
  useCreateConnector,
  useCreateProvider,
  useDeleteConnector,
  useDeleteProvider,
  useProviders,
  useSaveConnectorCredential,
  useSaveProviderCredential,
  useTestConnector,
  useTestProvider,
} from '../../hooks/use-intelligence'
import type { Connector, ProviderProfile } from '../../schemas/intelligence.schemas'

const PROVIDER_TYPES = [
  { value: 'openai', label: 'OpenAI (Responses / Codex models)' },
  { value: 'openai_compatible', label: 'OpenAI-compatible (Yunwu, gateways)' },
  { value: 'ollama', label: 'Ollama / local' },
  { value: 'codex_cli', label: 'Codex CLI (sandboxed, local only)' },
]

export function AiSettings() {
  return (
    <div className="space-y-6 rounded-lg border p-4">
      <div className="flex items-center gap-2">
        <Bot className="h-4 w-4" />
        <h3 className="text-sm font-medium">AI analysis settings</h3>
      </div>

      <Alert variant="default" className="border-muted">
        <AlertTitle className="text-xs">Privacy notes</AlertTitle>
        <AlertDescription className="text-xs space-y-1">
          <p>• Suggestions are preselect-only: the agent never writes to your finances; you confirm the import.</p>
          <p>• Credentials are encrypted (AES-256-GCM) and never returned by the API.</p>
          <p>• Web search connectors (Brave/Exa MCP) are used with redacted queries (no references, phones, amounts).</p>
          <p>• Codex CLI runs in a read-only sandbox with MCP, hooks, memory, and web search disabled.</p>
        </AlertDescription>
      </Alert>

      <ProvidersSection />
      <ConnectorsSection />
    </div>
  )
}

// ---- providers ----

function ProvidersSection() {
  const { data: providers } = useProviders()
  const createProvider = useCreateProvider()
  const deleteProvider = useDeleteProvider()
  const saveCredential = useSaveProviderCredential()
  const testProvider = useTestProvider()

  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [type, setType] = useState('openai_compatible')
  const [baseUrl, setBaseUrl] = useState('')
  const [model, setModel] = useState('')
  const [allowLocal, setAllowLocal] = useState(false)
  const [apiKey, setApiKey] = useState('')
  const [saving, setSaving] = useState(false)

  const handleCreate = async () => {
    setSaving(true)
    try {
      await createProvider.mutateAsync({
        name,
        providerType: type as ProviderProfile['providerType'],
        baseUrl: baseUrl || undefined,
        model,
        allowLocal,
        apiKey: apiKey || undefined,
      })
      toast.success('Provider added.')
      setShowForm(false)
      setName('')
      setBaseUrl('')
      setModel('')
      setApiKey('')
    } catch {
      // toast from hook
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Providers</h4>
        <Button size="sm" variant="outline" className="h-7 text-xs" onClick={() => setShowForm((v) => !v)}>
          <Plus className="mr-1 h-3 w-3" /> Add provider
        </Button>
      </div>

      {showForm && (
        <div className="space-y-3 rounded-md border p-3">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div className="space-y-1">
              <Label className="text-xs">Name</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} className="h-8 text-xs" placeholder="Yunwu" />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Type</Label>
              <Select value={type} onValueChange={setType}>
                <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {PROVIDER_TYPES.map((t) => (
                    <SelectItem key={t.value} value={t.value} className="text-xs">{t.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {type !== 'codex_cli' && (
              <div className="space-y-1">
                <Label className="text-xs">Base URL (https required unless local mode)</Label>
                <Input value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} className="h-8 text-xs" placeholder="https://…" />
              </div>
            )}
            <div className="space-y-1">
              <Label className="text-xs">Model</Label>
              <Input value={model} onChange={(e) => setModel(e.target.value)} className="h-8 text-xs" placeholder="gpt-5.2-codex / qwen3 / llama3" />
            </div>
          </div>
          {type !== 'codex_cli' && (
            <div className="space-y-1">
              <Label className="text-xs">API key (optional for local)</Label>
              <Input type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} className="h-8 text-xs" placeholder="sk-…" />
            </div>
          )}
          {type !== 'codex_cli' && (
            <div className="flex items-center gap-2">
              <Checkbox id="allow-local" checked={allowLocal} onCheckedChange={(v) => setAllowLocal(v === true)} />
              <Label htmlFor="allow-local" className="text-xs">Allow loopback endpoint (local provider)</Label>
            </div>
          )}
          <div className="flex gap-2">
            <Button size="sm" className="h-7 text-xs" onClick={handleCreate} disabled={saving || !name || !model}>
              {saving && <Loader2 className="mr-1 h-3 w-3 animate-spin" />} Save
            </Button>
            <Button size="sm" variant="outline" className="h-7 text-xs" onClick={() => setShowForm(false)}>Cancel</Button>
          </div>
        </div>
      )}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="text-xs">Name</TableHead>
              <TableHead className="text-xs">Type / model</TableHead>
              <TableHead className="text-xs">Credential</TableHead>
              <TableHead className="text-xs">Status</TableHead>
              <TableHead className="w-28 text-right"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(providers ?? []).map((p) => (
              <ProviderRow key={p.id} provider={p} onDelete={(id) => deleteProvider.mutate(id)} onTest={(id) => testProvider.mutate(id)} onCredential={(id, v) => saveCredential.mutate({ id, value: v })} />
            ))}
            {(providers ?? []).length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-xs text-muted-foreground">
                  No providers yet — add one to enable AI import analysis.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

function ProviderRow({
  provider,
  onDelete,
  onTest,
  onCredential,
}: {
  provider: ProviderProfile
  onDelete: (id: string) => void
  onTest: (id: string) => void
  onCredential: (id: string, value: string) => void
}) {
  const [keyValue, setKeyValue] = useState('')
  const testing = useTestProvider().isPending

  return (
    <TableRow>
      <TableCell className="text-xs">{provider.name}</TableCell>
      <TableCell className="text-xs text-muted-foreground">
        {provider.providerType} · {provider.model}
      </TableCell>
      <TableCell className="text-xs">
        {provider.hasCredential ? (
          <Badge variant="outline" className="text-[10px]">stored</Badge>
        ) : (
          <div className="flex items-center gap-1">
            <Input type="password" value={keyValue} onChange={(e) => setKeyValue(e.target.value)} className="h-6 w-36 text-xs" placeholder="API key" />
            <Button size="sm" variant="outline" className="h-6 text-xs" disabled={!keyValue} onClick={() => { onCredential(provider.id, keyValue); setKeyValue('') }}>
              Save
            </Button>
          </div>
        )}
      </TableCell>
      <TableCell>
        <Badge variant={provider.enabled ? 'default' : 'outline'} className="text-[10px]">
          {provider.enabled ? 'enabled' : 'disabled'}
        </Badge>
      </TableCell>
      <TableCell className="text-right">
        <Button size="sm" variant="ghost" className="h-6 text-xs" onClick={() => onTest(provider.id)} disabled={testing}>
          <Zap className="mr-1 h-3 w-3" /> Test
        </Button>
        <Button size="sm" variant="ghost" className="h-6 text-xs text-destructive" onClick={() => onDelete(provider.id)}>
          <Trash2 className="h-3 w-3" />
        </Button>
      </TableCell>
    </TableRow>
  )
}

// ---- connectors ----

function ConnectorsSection() {
  const { data: connectors } = useConnectors()
  const createConnector = useCreateConnector()
  const deleteConnector = useDeleteConnector()
  const saveCredential = useSaveConnectorCredential()
  const testConnector = useTestConnector()

  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [endpoint, setEndpoint] = useState('')
  const [allowlist, setAllowlist] = useState('brave_web_search')
  const [token, setToken] = useState('')
  const [saving, setSaving] = useState(false)

  const handleCreate = async () => {
    setSaving(true)
    try {
      await createConnector.mutateAsync({
        name,
        endpoint,
        allowlist: allowlist.split(',').map((s) => s.trim()).filter(Boolean),
        token: token || undefined,
      })
      toast.success('Connector added.')
      setShowForm(false)
      setName('')
      setEndpoint('')
      setToken('')
    } catch {
      // toast from hook
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          Web search connectors (MCP)
        </h4>
        <Button size="sm" variant="outline" className="h-7 text-xs" onClick={() => setShowForm((v) => !v)}>
          <Plus className="mr-1 h-3 w-3" /> Add connector
        </Button>
      </div>

      {showForm && (
        <div className="space-y-3 rounded-md border p-3">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div className="space-y-1">
              <Label className="text-xs">Name</Label>
              <Input value={name} onChange={(e) => setName(e.target.value)} className="h-8 text-xs" placeholder="Brave Search" />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Endpoint (https)</Label>
              <Input value={endpoint} onChange={(e) => setEndpoint(e.target.value)} className="h-8 text-xs" placeholder="https://…/mcp" />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Allowlisted tools (comma separated)</Label>
              <Input value={allowlist} onChange={(e) => setAllowlist(e.target.value)} className="h-8 text-xs" />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Bearer token</Label>
              <Input type="password" value={token} onChange={(e) => setToken(e.target.value)} className="h-8 text-xs" placeholder="mcp token" />
            </div>
          </div>
          <div className="flex gap-2">
            <Button size="sm" className="h-7 text-xs" onClick={handleCreate} disabled={saving || !name || !endpoint}>
              {saving && <Loader2 className="mr-1 h-3 w-3 animate-spin" />} Save
            </Button>
            <Button size="sm" variant="outline" className="h-7 text-xs" onClick={() => setShowForm(false)}>Cancel</Button>
          </div>
        </div>
      )}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="text-xs">Name</TableHead>
              <TableHead className="text-xs">Endpoint</TableHead>
              <TableHead className="text-xs">Tools</TableHead>
              <TableHead className="text-xs">Status</TableHead>
              <TableHead className="w-28 text-right"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(connectors ?? []).map((c) => (
              <ConnectorRow key={c.id} connector={c} onDelete={(id) => deleteConnector.mutate(id)} onTest={(id) => testConnector.mutate(id)} onCredential={(id, v) => saveCredential.mutate({ id, value: v })} />
            ))}
            {(connectors ?? []).length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-xs text-muted-foreground">
                  No connectors — merchant lookup stays model-only.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

function ConnectorRow({
  connector,
  onDelete,
  onTest,
  onCredential,
}: {
  connector: Connector
  onDelete: (id: string) => void
  onTest: (id: string) => void
  onCredential: (id: string, value: string) => void
}) {
  const [tokenValue, setTokenValue] = useState('')

  return (
    <TableRow>
      <TableCell className="text-xs">{connector.name}</TableCell>
      <TableCell className="max-w-[220px] truncate text-xs text-muted-foreground">{connector.endpoint}</TableCell>
      <TableCell className="text-xs text-muted-foreground">{connector.allowlist.join(', ')}</TableCell>
      <TableCell>
        <Badge variant={connector.enabled ? 'default' : 'outline'} className="text-[10px]">
          {connector.enabled ? 'enabled' : 'disabled'}
        </Badge>
      </TableCell>
      <TableCell className="text-right">
        {!connector.hasToken && (
          <>
            <Input type="password" value={tokenValue} onChange={(e) => setTokenValue(e.target.value)} className="mb-1 h-6 w-32 text-xs" placeholder="token" />
            <Button size="sm" variant="outline" className="h-6 text-xs" disabled={!tokenValue} onClick={() => { onCredential(connector.id, tokenValue); setTokenValue('') }}>
              Save
            </Button>
          </>
        )}
        <Button size="sm" variant="ghost" className="h-6 text-xs" onClick={() => onTest(connector.id)}>
          <Zap className="mr-1 h-3 w-3" /> Test
        </Button>
        <Button size="sm" variant="ghost" className="h-6 text-xs text-destructive" onClick={() => onDelete(connector.id)}>
          <Trash2 className="h-3 w-3" />
        </Button>
      </TableCell>
    </TableRow>
  )
}
