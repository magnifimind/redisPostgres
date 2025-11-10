-- Create the bitcoins table
CREATE TABLE IF NOT EXISTS bitcoins (
    symbol VARCHAR(10) PRIMARY KEY,
    price INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create an index on price for faster rank queries
CREATE INDEX IF NOT EXISTS idx_bitcoin_price ON bitcoins(price DESC);

-- Insert some sample data
INSERT INTO bitcoins (symbol, price) VALUES
    ('BTC', 65000),
    ('ETH', 3500),
    ('BNB', 450)
ON CONFLICT (symbol) DO NOTHING;

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at
CREATE TRIGGER update_bitcoins_updated_at
    BEFORE UPDATE ON bitcoins
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
