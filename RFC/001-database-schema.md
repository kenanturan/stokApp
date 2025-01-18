# RFC 001: Veritabanı Şeması

## Özet
Ürün yönetimi için veritabanı şemasının oluşturulması.

## Detaylar
Product tablosu aşağıdaki alanları içerecektir:
- id (PRIMARY KEY)
- company_name (VARCHAR)
- category (VARCHAR)
- product_name (VARCHAR)
- unit (VARCHAR)
- invoice_no (VARCHAR)
- invoice_date (DATE)
- initial_stock (DECIMAL)
- current_stock (DECIMAL)
- unit_price (DECIMAL)
- vat (DECIMAL)
- total_cost (DECIMAL) 