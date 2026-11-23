# SUKI Laundry — Web Application (Pelanggan & Owner)

Aplikasi manajemen dan pelacakan laundry modern dengan antarmuka **Mobile-First untuk Pelanggan** dan **Desktop Dashboard untuk Owner/Kasir**. Dibangun menggunakan tech stack ringan tanpa build-step kompleks: **HTML5, Tailwind CSS, Vanilla JavaScript (SPA Router)**, dan **Golang (`net/http`)**.

---
Kelompok :
- Mikhael Lucien Then /255150201111036
- Michael Juanio Patrick /255150201111037
- Almer Firdaus Widjokongko /255150201111039
---

## 1. Product Requirements Document (PRD)

### 1.1 Latar Belakang & Tujuan
Outlet laundry di area kampus/perkotaan membutuhkan alur pencatatan pesanan yang cepat, transparan, dan minim kesalahan saat pengambilan. SUKI Laundry menyelesaikan masalah tersebut melalui dua antarmuka terintegrasi:
- **Pelanggan (Mobile View):** Memantau progres cucian secara real-time, menyimpan nota digital & QR code pengambilan, dan mengakses bantuan cepat 24 jam.
- **Owner / Kasir (Desktop View):** Mencatat pesanan baru dengan kalkulasi tarif otomatis, mengatur prioritas antrean kerja (Ekspres vs Reguler), memverifikasi pengambilan barang, dan merekap pendapatan harian/mingguan.

### 1.2 Target Pengguna & Peran (Roles)
| Role | Perangkat Utama | Akses URL | Akun Demo |
| :--- | :--- | :--- | :--- |
| **Pelanggan** | Smartphone (Mobile View) | `/pelanggan/*` | `richard@student.ub.ac.id` / `123` |
| **Owner / Kasir** | Laptop / PC (Desktop View) | `/owner/*` | `ayu@sukilaundry.id` / `admin` |

### 1.3 Ruang Lingkup Fitur (Functional Requirements)

#### A. Autentikasi & Role Detection (`/login.html`)
- Login menggunakan Email atau Nomor WhatsApp beserta kata sandi.
- Deteksi peran otomatis: akun pelanggan diarahkan ke `/pelanggan/beranda`, sedangkan akun owner diarahkan ke `/owner/dasbor`.
- Proteksi halaman (*route guard*) pada masing-masing antarmuka.

#### B. Modul Pelanggan — Mobile View (`/pelanggan/`)
1. **Beranda (`/pelanggan/beranda`)**
   - Status outlet (lokasi & jam operasional `08.00–20.00`).
   - Kartu pesanan aktif dengan indikator *progress bar* dan estimasi selesai.
   - Daftar pesanan lainnya (*Siap diambil*, *Selesai*) dan pintasan ke Chat Bantuan.
2. **Lacak Cucian (`/pelanggan/lacak`)**
   - *Vertical timeline* 4 tahap: **Antre → Dicuci → Disetrika → Siap Diambil**.
   - Informasi berat, jenis layanan, estimasi waktu selesai, dan tombol langsung ke nota.
3. **Nota Digital & Riwayat (`/pelanggan/nota`)**
   - *Segmented tab* untuk beralih antara Nota Aktif dan Riwayat transaksi.
   - Rincian biaya transparan (tarif per kg, pewangi tambahan, diskon mahasiswa, total).
   - Kode QR digital untuk dipindai kasir saat pengambilan cucian.
4. **Bantuan AI (`/pelanggan/bantuan`)**
   - Asisten virtual interaktif dengan tombol *quick reply* (`Cek status cucian`, `Daftar harga`, `Jam buka`, `Estimasi selesai`).
   - Terhubung langsung ke data pesanan aktif milik pelanggan.
5. **Profil & Notifikasi (`/pelanggan/profil`)**
   - Informasi akun pelanggan dan pembaruan Nomor WhatsApp.
   - Toggle preferensi notifikasi (saat cucian selesai & setiap perubahan status).

#### C. Modul Owner — Desktop View (`/owner/`)
1. **Dasbor Utama (`/owner/dasbor`)**
   - 4 Kartu KPI operasional (Pesanan Aktif, Sedang Diproses, Siap Diambil, Total Transaksi).
   - Grafik batang pendapatan mingguan dan kartu prioritas pesanan Ekspres.
   - Tabel ringkasan pesanan terbaru.
2. **Pesanan Baru (`/owner/pesanan-baru`)**
   - Formulir input pelanggan dengan validasi Nomor WhatsApp minimal 10 digit.
   - Pilihan jenis layanan (`Cuci + setrika`, `Cuci kering`, `Setrika saja`) dan paket (`Reguler` 2 hari Rp7.000/kg vs `Ekspres` 8 jam Rp14.000/kg).
   - Panel ringkasan harga dan estimasi selesai yang dihitung otomatis.
3. **Antrean Kerja (`/owner/antrean`)**
   - Papan Kanban 4 kolom: **Antre**, **Dicuci**, **Disetrika**, **Siap Diambil**.
   - Pengurutan otomatis: pesanan **Ekspres** selalu tampil di urutan teratas tiap kolom.
   - Filter berdasarkan paket layanan dan tombol satu klik `Pindah ke tahap berikutnya`.
4. **Verifikasi Pengambilan (`/owner/verifikasi`)**
   - Pencarian berdasarkan kode pesanan / nama pelanggan atau simulasi pindai QR.
   - Menampilkan lokasi rak penyimpanan, status pembayaran, dan *checklist* kelengkapan kantong sebelum konfirmasi selesai.
5. **Rekap Pendapatan (`/owner/rekap`)**
   - Ringkasan total pemasukan beserta pemisahan pembayaran **Tunai** dan **Nontunai**.
   - Grafik pendapatan harian, daftar transaksi terbaru, dan fitur **Ekspor CSV**.

### 1.4 Design System & Palet Warna
| Token | Kode Hex | Fungsi |
| :--- | :--- | :--- |
| `canvas` | `#D7E3E3` | Latar belakang utama halaman |
| `surface` | `#FFFFFF` | Kartu, sidebar, bottom navigation, dan input |
| `tealbrand` | `#0F7D8F` | Warna primer (CTA, tab aktif, progress bar, grafik hari ini) |
| `aqua` | `#9DD0D9` | Sorotan (kartu status aktif, prioritas ekspres, lokasi rak) |
| `mutedbox` | `#D3DFDF` | Header tabel, badge cabang, kotak rincian sekunder |
| `ink` | `#163338` | Teks utama dan angka KPI |
| `inksoft` | `#4E686D` | Teks sekunder dan label timestamp |
| `borderline` | `#9FC2C8` | Garis tepi input dan tombol sekunder |

---

## 2. Arsitektur & Struktur Folder

```text
suki-laundry/
├── api/
│   └── index.go                # Go Serverless Handler (untuk Vercel)
├── shared/
│   ├── tailwind-config.js      # Konfigurasi tema & palet warna terpusat
│   └── api.js                  # Helper fetch API, auth guard, format Rupiah
├── pelanggan/
│   ├── index.html              # Mobile App Shell + Bottom Navigation
│   ├── router.js               # Client-side Router & Controller Pelanggan
│   └── pages/
│       ├── beranda.html
│       ├── lacak.html
│       ├── nota.html
│       ├── bantuan.html
│       └── profil.html
├── owner/
│   ├── index.html              # Desktop App Shell + Sidebar + Topbar
│   ├── router.js               # Client-side Router & Controller Owner
│   └── pages/
│       ├── dasbor.html
│       ├── pesanan-baru.html
│       ├── antrean.html
│       ├── verifikasi.html
│       └── rekap.html
├── login.html                  # Halaman login dengan auto-deteksi role
├── main.go                     # Server lokal Golang (untuk development)
├── go.mod                      # Modul Go
├── vercel.json                 # Routing & Rewrite rules Vercel
└── README.md                   # Dokumentasi & PRD