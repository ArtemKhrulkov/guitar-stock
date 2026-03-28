-- Add image-related fields to guitars table
ALTER TABLE guitars ADD COLUMN IF NOT EXISTS image_source VARCHAR(50);
ALTER TABLE guitars ADD COLUMN IF NOT EXISTS image_scraped_at TIMESTAMP;

-- Create guitar_images table for storing multiple image options
CREATE TABLE IF NOT EXISTS guitar_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guitar_id UUID REFERENCES guitars(id) ON DELETE CASCADE,
    image_url VARCHAR(500) NOT NULL,
    source VARCHAR(50) NOT NULL,
    width INT,
    height INT,
    is_primary BOOLEAN DEFAULT false,
    scraped_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(guitar_id, image_url)
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_guitar_images_guitar_id ON guitar_images(guitar_id);
CREATE INDEX IF NOT EXISTS idx_guitar_images_source ON guitar_images(source);
