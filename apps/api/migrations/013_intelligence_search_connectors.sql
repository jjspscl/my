-- 013_intelligence_search_connectors
-- Search connectors gain a provider kind so native Tavily/Brave/Exa adapters
-- can replace hand-configured MCP endpoints, and an auth type for the custom
-- MCP escape hatch (none | bearer | x-api-key).
--
-- Existing rows keep their endpoint + allowlist and become custom MCP
-- connectors with bearer auth — no data migration needed.

ALTER TABLE intelligence_mcp_connectors ADD COLUMN connector_kind TEXT NOT NULL DEFAULT 'custom_mcp';
ALTER TABLE intelligence_mcp_connectors ADD COLUMN auth_type TEXT NOT NULL DEFAULT 'bearer';
