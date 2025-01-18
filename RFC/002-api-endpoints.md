# RFC 002: API Endpoint'leri

## Özet
Ürün ekleme ve silme için REST API endpoint'lerinin implementasyonu.

## Detaylar
1. POST /api/v1/products
   - Request body validation
   - Veritabanına kayıt
   - Hata yönetimi

2. DELETE /api/v1/products/:id
   - ID validation
   - Kayıt silme
   - Hata yönetimi 