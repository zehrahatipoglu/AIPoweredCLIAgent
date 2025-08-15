# 🤖 AI-Powered CLI Agent & FizzBuzz Demo

Bu proje, **Go** dilinde geliştirilmiş bir **OpenAI destekli komut satırı ajanı** ve basit bir **JavaScript FizzBuzz** örneğini içerir.  
Ajan, OpenAI'nin `ChatCompletion` API’si üzerinden doğal dil ile etkileşime geçerek dosya okuma, listeleme ve düzenleme işlemleri yapabilir.  
Proje aynı zamanda klasik **FizzBuzz algoritması** ile örnek bir JavaScript kodu içerir.

---

## 📌 Özellikler

### Go CLI Agent
- **OpenAI API Entegrasyonu** (GPT-3.5 Turbo)
- **Fonksiyon Çağrısı (Function Calling)** desteği
- **Araçlar (Tools)**
  - 📂 `read_file` — Dosya içeriğini okuma
  - 📜 `list_files` — Dizin içeriğini listeleme
  - ✏️ `edit_file` — Dosya düzenleme veya oluşturma
- **JSON Schema ile Parametre Doğrulama**
- **Gerçek zamanlı CLI etkileşimi**

### JavaScript FizzBuzz
- 1’den 15’e kadar sayıları yazar.
- 3’e bölünebilen sayılar için **Fizz**, 5’e bölünebilenler için **Buzz**, her ikisine de bölünebilenler için **FizzBuzz** yazdırır.

---

Çalışma Akışı:

1. OpenAI API anahtarını ortam değişkeninden alır (OPENAI_API_KEY).

2. Kullanıcıdan terminal üzerinden metin komutu alır.

3. Bu komutu, önceki konuşma geçmişiyle birlikte OpenAI API’sine gönderir.

4. Model, eğer ihtiyaç duyarsa bir tool call (fonksiyon çağrısı) yapar.
   
  Tanımlı araçlar:

  * read_file → Dosya içeriğini okur.

  * list_files → Dizin içeriğini listeler.

  * edit_file → Dosya düzenler veya oluşturur.

5. Tool çağrısı gelirse ilgili fonksiyon çalışır ve sonucu tekrar modele gönderilir.

6. Modelin ürettiği nihai cevap terminalde kullanıcıya gösterilir.

```
go run main.go
```

<img width="797" height="253" alt="Ekran görüntüsü 2025-08-02 024714" src="https://github.com/user-attachments/assets/e9e9ac5e-32cc-4da4-9742-6f3f43140c7e" />


<img width="449" height="209" alt="Ekran görüntüsü 2025-08-03 011329" src="https://github.com/user-attachments/assets/a61b7b9a-b63a-435b-901c-068bf4f35c54" />

<img width="774" height="389" alt="Ekran görüntüsü 2025-08-03 012858" src="https://github.com/user-attachments/assets/550aa445-472a-48b4-b7e8-c94607370f56" />



