# Ürün Gereksinimleri Dökümanı (PRD)

## Proje Özeti
Stok takip sistemi için ürün ekleme ve silme işlemlerini yönetecek bir REST API.

## Temel Gereksinimler

### Teknik Gereksinimler
- API, Go programlama dili ve Gin framework kullanılarak geliştirilecek
- Veritabanı olarak SQLite ve GORM ORM kullanılacak
- API 8080 portunda çalışacak
- OpenAPI/Swagger dökümanı sağlanacak
- Unit testler yazılacak
- Stok yönetimi FIFO (First In First Out) mantığına göre yapılacak
- İlişkili modeller için validation kuralları esnek olmalı

### Ürün Özellikleri
1. Ürün Ekleme
   - Firma adı
   - Kategori
   - Ürün adı
   - Birim
   - Fatura no
   - Fatura tarihi
   - Giriş stoğu
   - Kalan stok
   - Birim fiyatı
   - KDV
   - Toplam maliyet

2. Stok Yönetimi
   - Ürünlerin stok miktarları takip edilir
   - Her ürün için güncel stok miktarı tutulur
   - Aynı ürün adına sahip farklı partiler için ayrı stok takibi yapılır
   - Stok kullanımında FIFO (First In First Out) mantığı uygulanır
   - Eski tarihli stoklar öncelikli olarak kullanılır

   ### Stok Hareketleri
   - Ürün girişleri stok hareketleri olarak kaydedilir
   - Her stok hareketi için:
     * Başlangıç miktarı
     * Kalan miktar
     * Birim maliyet
     * Hareket tarihi bilgileri tutulur
   - Satış işlemlerinde hangi stok hareketinden ne kadar kullanıldığı kaydedilir

3. Ürün Silme
   - ID ile ürün silme

4. Ürün Satışı
   - Ürün ID
   - Satış miktarı
   - Satış tarihi
   - Satış fiyatı
   - Müşteri adı
   - Müşteri telefon
   - Satış notu

### API Endpoints
- GET /api/v1/products - Tüm ürünleri listele
- POST /api/v1/products - Yeni ürün ekleme
- DELETE /api/v1/products/:id - Ürün silme
- POST /api/v1/sales - Yeni satış ekle
- GET /api/v1/sales - Tüm satışları listele
- GET /api/v1/stock-movements - Stok hareketlerini listele (FIFO sırasına göre)

## Kabul Kriterleri
1. Tüm alanlar doğru formatta girilmelidir
2. Fatura tarihi geçerli bir tarih olmalıdır
3. Birim fiyatı ve KDV pozitif sayı olmalıdır
4. Stok miktarları 0 veya pozitif sayı olmalıdır
5. API istekleri uygun HTTP durum kodlarıyla yanıt vermelidir
6. Satış işlemlerinde FIFO mantığına uygun olarak stok çıkışı yapılmalıdır
7. Yetersiz stok durumunda satış işlemi yapılmamalıdır
8. Her satış işleminde kullanılan stok hareketleri kaydedilmelidir
9. Stok hareketleri FIFO sırasına göre listelenmelidir
10. Her stok hareketi için kalan miktar güncel olmalıdır 