-- Yeni migration dosyası
ALTER TABLE sales ADD COLUMN unit_cost DECIMAL(10,2) NOT NULL DEFAULT 0;

-- Geri alma
-- ALTER TABLE sales DROP COLUMN unit_cost; 