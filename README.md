# Backend Smart Home

Backend sistem monitoring smart home berbasis Go yang terintegrasi dengan Firebase Realtime Database, Firestore, dan Firebase Cloud Messaging untuk monitoring dan notifikasi real-time.

## Deskripsi Project

Backend Smart Home adalah aplikasi server-side yang dirancang untuk memonitor dan mengelola sistem smart home secara real-time. Aplikasi ini mengimplementasikan tiga modul monitoring utama yang berjalan secara konkuren:

1. Monitoring konsumsi listrik dengan perhitungan biaya
2. Monitoring sensor asap untuk deteksi kebakaran
3. Sistem notifikasi push menggunakan Firebase Cloud Messaging

## Fitur Utama

### 1. Monitoring Listrik
- Monitoring konsumsi daya listrik secara real-time
- Perhitungan otomatis konsumsi energi dalam Watt-hour (Wh) dan kilowatt-hour (kWh)
- Kalkulasi biaya listrik berdasarkan tarif per kWh
- Penyimpanan historis data harian dan bulanan di Firestore
- Perhitungan rata-rata konsumsi dan statistik penggunaan listrik

### 2. Monitoring Sensor Asap
- Deteksi asap real-time untuk pencegahan kebakaran
- Pembaruan status notifikasi otomatis saat asap terdeteksi
- Monitoring berkelanjutan dengan interval pengecekan 2 detik

### 3. Sistem Notifikasi
- Push notification via Firebase Cloud Messaging (FCM)
- Notifikasi otomatis untuk deteksi asap dan status tandon air
- Manajemen FCM token untuk multiple devices
- API REST untuk registrasi device dan status notifikasi

## Teknologi yang Digunakan

- **Bahasa Pemrograman**: Go 1.24
- **Database**: 
  - Firebase Realtime Database
  - Cloud Firestore
- **Cloud Services**: 
  - Firebase Cloud Messaging
  - Google Cloud Platform
- **Dependencies Utama**:
  - `firebase.google.com/go/v4`
  - `cloud.google.com/go/firestore`
  - `google.golang.org/api`

## Struktur File

```
backend_smart_home/
├── main.go                      # Entry point aplikasi
├── monitoringListrik.go         # Modul monitoring konsumsi listrik
├── monitoringSensorAsap.go      # Modul monitoring sensor asap
├── notifikasi.go                # Modul sistem notifikasi dan API
├── go.mod                       # Dependency management
├── go.sum                       # Checksum dependencies
├── serviceAccountKey.json       # Firebase service account (tidak di-commit)
└── .gitignore                   # Konfigurasi file yang diabaikan
```

## Prasyarat

- Go 1.24 atau versi lebih baru
- Firebase Project dengan layanan:
  - Realtime Database
  - Firestore
  - Cloud Messaging
- Service Account Key dari Firebase Console

## Instalasi

### 1. Clone Repository

```bash
git clone https://github.com/nuhgroh2004/backend_smart_home.git
cd backend_smart_home
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Konfigurasi Firebase

1. Buat project di [Firebase Console](https://console.firebase.google.com/)
2. Aktifkan Realtime Database, Firestore, dan Cloud Messaging
3. Download Service Account Key:
   - Buka Project Settings > Service Accounts
   - Generate New Private Key
   - Simpan sebagai `serviceAccountKey.json` di root directory

### 4. Konfigurasi Database URL

Sesuaikan URL database di file-file berikut:
- `monitoringListrik.go`: Line 55
- `monitoringSensorAsap.go`: Line 30
- `notifikasi.go`: Line 51

```go
const realtimeDatabaseURL = "https://your-project-id-default-rtdb.firebaseio.com"
```

### 5. Build Aplikasi

```bash
go build -o backend_smart_home
```

## Cara Menjalankan

### Mode Normal (Monitoring)

```bash
./backend_smart_home
```

atau

```bash
go run .
```

### Mode API Server

```bash
./backend_smart_home api
```

atau

```bash
go run . api
```

## Struktur Data Firebase

### Realtime Database

```json
{
  "IoTSystem": {
    "Listrik": {
      "dayaSaatIni_W": 150.5,
      "timestamp": "2026-01-26T10:30:00Z"
    },
    "SensorAsap": {
      "nilai": 250,
      "status": "AMAN"
    },
    "Notifikasi": {
      "asapTerdeteksi": false,
      "pesan": "Semua sistem normal",
      "tandonPenuh": false
    },
    "FCMTokens": {
      "device_id_1": {
        "token": "fcm_token_here",
        "deviceName": "Android Device",
        "active": true,
        "lastUpdated": "2026-01-26T10:00:00Z"
      }
    }
  }
}
```

### Firestore Collections

#### electricity_history

```json
{
  "2026-01-26": {
    "totalDaya_Wh": 3600.5,
    "totalDaya_kWh": 3.6005,
    "jumlahPembacaan": 120,
    "rataRata_W": 150.02,
    "dayaTerakhir_W": 155.5,
    "biayaHarian_Rp": 4867.88,
    "date": "2026-01-26",
    "year": 2026,
    "month": 1,
    "day": 26,
    "firstRecordedAt": "2026-01-26T00:00:00Z",
    "lastUpdatedAt": "2026-01-26T23:59:00Z",
    "lastPembacaanAt": "2026-01-26T23:59:00Z"
  }
}
```

#### monthly_electricity_history

```json
{
  "2026-01": {
    "totalDaya_Wh": 93600,
    "totalKonsumsi_kWh": 93.6,
    "totalBiaya_Rp": 126547.2,
    "rataRataHarian_kWh": 3.6,
    "jumlahHari": 26,
    "month": 1,
    "year": 2026,
    "lastUpdatedAt": "2026-01-26T23:59:00Z"
  }
}
```

## API Endpoints

Server API berjalan di `http://localhost:8080` saat menjalankan mode API.

### POST /api/fcm/register

Registrasi FCM token untuk device.

**Request Body:**
```json
{
  "device_id": "unique_device_id",
  "fcm_token": "firebase_cloud_messaging_token",
  "device_name": "My Android Phone"
}
```

**Response:**
```json
{
  "success": true,
  "message": "FCM token registered successfully",
  "data": {
    "device_id": "unique_device_id",
    "device_name": "My Android Phone",
    "registered_at": "2026-01-26T10:30:00Z"
  }
}
```

### GET /api/fcm/notification-status

Mendapatkan status notifikasi saat ini.

**Response:**
```json
{
  "success": true,
  "data": {
    "asapTerdeteksi": false,
    "pesan": "Semua sistem normal",
    "tandonPenuh": false
  }
}
```

### GET /api/fcm/tokens

Mendapatkan semua FCM tokens yang terdaftar.

**Response:**
```json
{
  "success": true,
  "data": {
    "device_id_1": {
      "token": "fcm_token_here",
      "deviceName": "Android Device",
      "active": true,
      "lastUpdated": "2026-01-26T10:00:00Z"
    }
  }
}
```

## Konfigurasi

### Interval Monitoring

Sesuaikan interval pengecekan di file masing-masing:

**monitoringListrik.go:**
```go
const checkIntervalSeconds = 3 * time.Second
```

**monitoringSensorAsap.go:**
```go
const sensorAsapCheckInterval = 2 * time.Second
```

**notifikasi.go:**
```go
const checkInterval = 2 * time.Second
```

### Tarif Listrik

Sesuaikan tarif listrik per kWh di `monitoringListrik.go`:

```go
const tarifPerKwh_Rp = 1352.0
```

## Algoritma Perhitungan Konsumsi Listrik

Sistem menggunakan metode perhitungan berbasis interval waktu:

1. **Konsumsi per Interval:**
   ```
   konsumsiWh = ((dayaSebelum + dayaSekarang) / 2) * intervalJam
   ```

2. **Total Konsumsi:**
   ```
   totalDaya_kWh = totalDaya_Wh / 1000
   ```

3. **Biaya Listrik:**
   ```
   biayaHarian_Rp = totalDaya_kWh * tarifPerKwh_Rp
   ```

4. **Rata-rata Konsumsi:**
   ```
   rataRata_W = totalDaya_Wh / totalDurasiJam
   ```

## Fitur Concurrency

Aplikasi menggunakan goroutine untuk menjalankan tiga modul secara bersamaan:

```go
// Notifikasi monitoring
go RunNotifikasi(ctx)

// Monitoring listrik
go RunMonitoringListrik(ctx)

// Monitoring sensor asap
go RunMonitoringSensorAsap(ctx)
```

Setiap modul berjalan independen dengan context cancellation untuk graceful shutdown.

## Graceful Shutdown

Aplikasi menangani signal interrupt (CTRL+C) dengan baik:

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan
cancel() // Cancel context untuk stop semua goroutine
```

## Logging

Aplikasi menggunakan format logging yang konsisten:

- Informasi: `✓` atau `✅`
- Warning: `⚠️`
- Error: `❌`
- Monitoring: `🔍`
- API: `🌐`
- Dokumen: `📝`

## Keamanan

### File yang Tidak Boleh Di-commit

File berikut sudah dikonfigurasi di `.gitignore`:

- `serviceAccountKey.json` - Firebase credentials
- `*-firebase-adminsdk-*.json` - Service account keys
- `.env` files
- Binary files

### Best Practices

1. Jangan commit service account key ke repository
2. Gunakan environment variables untuk konfigurasi sensitif
3. Implementasi rate limiting untuk API endpoints
4. Validasi input dari API requests
5. Implement HTTPS untuk production

## Troubleshooting

### Error: "error initializing Firebase"

- Pastikan `serviceAccountKey.json` ada di root directory
- Verifikasi format file JSON valid
- Cek permission file (readable)

### Error: "error getting data"

- Pastikan struktur data di Realtime Database sesuai
- Verifikasi database URL benar
- Cek Firebase rules untuk read/write access

### Error: "no active FCM tokens found"

- Registrasi device terlebih dahulu via API endpoint
- Pastikan token FCM valid dan aktif

### Notifikasi Tidak Terkirim

- Verifikasi FCM token valid
- Cek Firebase Cloud Messaging diaktifkan
- Pastikan aplikasi client memiliki permission notifikasi

## Monitoring dan Maintenance

### Log Monitoring

Aplikasi menampilkan log real-time untuk setiap aktivitas:

```
[10:30:05] ✅ Update #120 | Daya: 155.50 W | Total: 3600.50 Wh | Rata-rata: 150.02 W
[10:30:07] 🔍 Sensor Asap | Nilai: 250 | Status: AMAN
```

### Database Maintenance

- Data harian disimpan per tanggal (format: YYYY-MM-DD)
- Data bulanan diaggregasi otomatis
- Implementasi data retention policy sesuai kebutuhan

## Kontribusi

Kontribusi untuk project ini sangat diterima. Silakan:

1. Fork repository
2. Buat branch baru untuk fitur
3. Commit perubahan
4. Push ke branch
5. Buat Pull Request

## Lisensi

Informasi lisensi belum tersedia.

## Kontak

Repository: [https://github.com/nuhgroh2004/backend_smart_home](https://github.com/nuhgroh2004/backend_smart_home)

## Changelog

### Version 1.0.0 (Current)

- Implementasi monitoring listrik real-time
- Implementasi monitoring sensor asap
- Sistem notifikasi FCM
- API REST untuk manajemen device
- Perhitungan konsumsi dan biaya listrik
- Penyimpanan historis data harian dan bulanan

## Rencana Pengembangan

- Implementasi dashboard web untuk monitoring
- Tambahan sensor (suhu, kelembaban, gerakan)
- Integrasi dengan sistem kontrol otomatis
- Machine learning untuk prediksi konsumsi
- Export data ke format CSV/Excel
- Notifikasi via email dan SMS
- Multi-user authentication
- Role-based access control

## Performa

- Interval monitoring: 2-3 detik
- Concurrent processing untuk multiple modul
- Efficient database writes dengan batch operations
- Low memory footprint
- Graceful shutdown untuk data consistency

## Dependencies

Untuk melihat semua dependencies yang digunakan, lihat file `go.mod`.

Dependencies utama:
- firebase.google.com/go/v4 v4.18.0
- cloud.google.com/go/firestore v1.20.0
- google.golang.org/api v0.247.0

## Catatan Penting

1. Aplikasi memerlukan koneksi internet stabil untuk komunikasi dengan Firebase
2. Pastikan Firebase Realtime Database rules dikonfigurasi dengan benar
3. Firestore indexes mungkin perlu dibuat untuk query kompleks
4. Monitoring interval yang terlalu cepat dapat meningkatkan biaya Firebase
5. Implementasi rate limiting untuk menghindari Firebase quota limits
