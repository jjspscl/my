-- 014_transaction_edit_delete
-- Edit/delete support for transactions:
--   * revision + updated_at enable optimistic concurrency (If-Match) so edits
--     and deletes never silently clobber a newer state.
--   * import provenance lifecycle: when an imported transaction is edited or
--     deleted the original statement entry is kept (it is immutable audit
--     data) but marked, and its batch is flagged as no longer intact.
--   * bill_payments.transaction_link_source distinguishes inferred (auto)
--     links from explicit user action (manual) so editing/deleting a
--     transaction can reconcile payments without un-paying a bill the user
--     deliberately marked paid.

ALTER TABLE transactions ADD COLUMN revision INTEGER NOT NULL DEFAULT 1;
ALTER TABLE transactions ADD COLUMN updated_at TEXT;

ALTER TABLE finance_import_entries ADD COLUMN entity_status TEXT NOT NULL DEFAULT 'active'; -- active | modified | deleted
ALTER TABLE finance_import_entries ADD COLUMN entity_modified_at TEXT;
ALTER TABLE finance_import_entries ADD COLUMN entity_deleted_at TEXT;

ALTER TABLE finance_imports ADD COLUMN integrity_status TEXT NOT NULL DEFAULT 'intact'; -- intact | modified

ALTER TABLE bill_payments ADD COLUMN transaction_link_source TEXT NOT NULL DEFAULT 'legacy'; -- auto | manual | legacy

-- Fast lookup of the import entry backing a transaction/transfer entity.
CREATE INDEX idx_finance_import_entries_entity
  ON finance_import_entries(entity_type, entity_id);
