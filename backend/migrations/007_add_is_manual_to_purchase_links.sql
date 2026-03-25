-- Migration: 007_add_is_manual_to_purchase_links
-- Adds is_manual column to track manually added purchase links vs scraped ones

ALTER TABLE purchase_links ADD COLUMN IF NOT EXISTS is_manual BOOLEAN NOT NULL DEFAULT FALSE;

-- Index for filtering manual links
CREATE INDEX IF NOT EXISTS idx_purchase_links_is_manual ON purchase_links(is_manual);

-- Comments
COMMENT ON COLUMN purchase_links.is_manual IS 'True if link was manually added, false if scraped automatically';
