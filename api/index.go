package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Role     string `json:"role"` // "pelanggan" atau "owner"
	Phone    string `json:"phone"`
	Subtitle string `json:"subtitle"`
	NotifEnd bool   `json:"notif_end"`
	NotifChg bool   `json:"notif_chg"`
}

type Order struct {
	ID            string  `json:"id"`
	CustomerName  string  `json:"customer_name"`
	CustomerEmail string  `json:"customer_email"`
	Phone         string  `json:"phone"`
	ServiceType   string  `json:"service_type"`
	Package       string  `json:"package"` // "Reguler" atau "Ekspres"
	Weight        float64 `json:"weight"`
	PricePerKg    int     `json:"price_per_kg"`
	PerfumeFee    int     `json:"perfume_fee"`
	Discount      int     `json:"discount"`
	Total         int     `json:"total"`
	Status        string  `json:"status"`         // "Antre", "Dicuci", "Disetrika", "Siap Diambil", "Selesai"
	PaymentStatus string  `json:"payment_status"` // "Lunas" atau "Belum lunas"
	PaymentMethod string  `json:"payment_method"` // "Tunai", "QRIS", "Transfer"
	RackLocation  string  `json:"rack_location"`
	BagsCount     int     `json:"bags_count"`
	Notes         string  `json:"notes"`
	CreatedAt     string  `json:"created_at"`
	EstFinish     string  `json:"est_finish"`
	StepTime      string  `json:"step_time"`
}

var (
	initOnce sync.Once
	mu       sync.RWMutex
	users    = map[string]*User{}
	orders   = []*Order{}
	seqNum   = 1049
)

func seedData() {
	users["richard@student.ub.ac.id"] = &User{
		ID: "USR-01", Name: "Richard Pratama", Email: "richard@student.ub.ac.id",
		Password: "123", Role: "pelanggan", Phone: "0812 3456 7890",
		Subtitle: "Mahasiswa · Universitas Brawijaya", NotifEnd: true, NotifChg: true,
	}
	users["ayu@sukilaundry.id"] = &User{
		ID: "OWN-01", Name: "Ayu Pratiwi", Email: "ayu@sukilaundry.id",
		Password: "admin", Role: "owner", Phone: "0811 2233 4455",
		Subtitle: "Pemilik & kasir", NotifEnd: true, NotifChg: true,
	}

	orders = []*Order{
		{
			ID: "#SK-1048", CustomerName: "Richard Pratama", CustomerEmail: "richard@student.ub.ac.id",
			Phone: "0812 3456 7890", ServiceType: "Cuci, kering & setrika", Package: "Reguler",
			Weight: 4, PricePerKg: 7000, PerfumeFee: 2000, Discount: 2000, Total: 28000,
			Status: "Disetrika", PaymentStatus: "Lunas", PaymentMethod: "QRIS",
			RackLocation: "B-01", BagsCount: 1, Notes: "Pisahkan kemeja putih.",
			CreatedAt: "16 November 2026 · 08.15 WIB", EstFinish: "Hari ini, 18.00 WIB", StepTime: "16 Nov, 14.10 WIB",
		},
		{
			ID: "#SK-1041", CustomerName: "Richard Pratama", CustomerEmail: "richard@student.ub.ac.id",
			Phone: "0812 3456 7890", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 3, PricePerKg: 7000, Total: 21000, Status: "Siap Diambil",
			PaymentStatus: "Lunas", PaymentMethod: "QRIS", RackLocation: "A-04", BagsCount: 1,
			CreatedAt: "14 November 2026 · 09.30 WIB", EstFinish: "16 Nov · 10.00 WIB", StepTime: "16 Nov, 09.45 WIB",
		},
		{
			ID: "#SK-1027", CustomerName: "Richard Pratama", CustomerEmail: "richard@student.ub.ac.id",
			Phone: "0812 3456 7890", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 2, PricePerKg: 12000, Total: 24000, Status: "Selesai",
			PaymentStatus: "Lunas", PaymentMethod: "Tunai", RackLocation: "A-01", BagsCount: 1,
			CreatedAt: "8 November 2026 · 10.00 WIB", EstFinish: "8 Nov · 18.00 WIB", StepTime: "8 Nov, 17.30 WIB",
		},
		{
			ID: "#SK-1611-024", CustomerName: "Nadia Putri", CustomerEmail: "nadia@mail.com",
			Phone: "0812 3450 9988", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 3, PricePerKg: 14000, Total: 42000, Status: "Antre",
			PaymentStatus: "Lunas", PaymentMethod: "QRIS", RackLocation: "C-01", BagsCount: 1,
			Notes: "Pisahkan pakaian putih. Jangan gunakan pewangi.",
			CreatedAt: "16 November 2026 · 09.00 WIB", EstFinish: "Hari ini, 17.00", StepTime: "16 Nov, 09.00 WIB",
		},
		{
			ID: "#SK-1611-022", CustomerName: "Rina Wulandari", CustomerEmail: "rina@mail.com",
			Phone: "0813 9988 1122", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 2, PricePerKg: 14000, Total: 28000, Status: "Antre",
			PaymentStatus: "Lunas", PaymentMethod: "Transfer", RackLocation: "C-02", BagsCount: 1,
			CreatedAt: "16 November 2026 · 10.00 WIB", EstFinish: "Hari ini, 18.00", StepTime: "16 Nov, 10.00 WIB",
		},
		{
			ID: "#SK-1611-020", CustomerName: "Dewi Lestari", CustomerEmail: "dewi@mail.com",
			Phone: "0812 7766 5544", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 4, PricePerKg: 14000, Total: 56000, Status: "Dicuci",
			PaymentStatus: "Lunas", PaymentMethod: "Tunai", RackLocation: "B-02", BagsCount: 2,
			CreatedAt: "16 November 2026 · 07.30 WIB", EstFinish: "Hari ini, 15.00", StepTime: "16 Nov, 09.15 WIB",
		},
		{
			ID: "#SK-1611-023", CustomerName: "Budi Santoso", CustomerEmail: "budi@mail.com",
			Phone: "0857 1122 3344", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 5, PricePerKg: 7000, Total: 35000, Status: "Dicuci",
			PaymentStatus: "Lunas", PaymentMethod: "QRIS", RackLocation: "B-04", BagsCount: 2,
			CreatedAt: "16 November 2026 · 08.45 WIB", EstFinish: "18 Nov, 09.00", StepTime: "16 Nov, 10.20 WIB",
		},
		{
			ID: "#SK-1611-019", CustomerName: "Andi Pratama", CustomerEmail: "andi@mail.com",
			Phone: "0821 4455 6677", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 3, PricePerKg: 14000, Total: 42000, Status: "Disetrika",
			PaymentStatus: "Lunas", PaymentMethod: "Transfer", RackLocation: "A-05", BagsCount: 1,
			CreatedAt: "16 November 2026 · 08.00 WIB", EstFinish: "Hari ini, 13.00", StepTime: "16 Nov, 11.40 WIB",
		},
		{
			ID: "#SK-1411-018", CustomerName: "Siti Rahma", CustomerEmail: "siti@mail.com",
			Phone: "0812 8899 0018", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 6, PricePerKg: 7000, Total: 42000, Status: "Siap Diambil",
			PaymentStatus: "Lunas", PaymentMethod: "Tunai", RackLocation: "B-03", BagsCount: 2,
			Notes: "Disimpan dalam 2 kantong terpisah.",
			CreatedAt: "14 November 2026 · 10.00 WIB", EstFinish: "16 Nov, 10.00", StepTime: "16 Nov, 09.30 WIB",
		},
		{
			ID: "#SK-1411-016", CustomerName: "Agus Wijaya", CustomerEmail: "agus@mail.com",
			Phone: "0819 3322 1100", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 3, PricePerKg: 7000, Total: 21000, Status: "Siap Diambil",
			PaymentStatus: "Belum lunas", PaymentMethod: "Tunai", RackLocation: "A-02", BagsCount: 1,
			CreatedAt: "14 November 2026 · 11.00 WIB", EstFinish: "16 Nov, 11.00", StepTime: "16 Nov, 10.15 WIB",
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Handler adalah entry point utama API (digunakan oleh Vercel & main.go lokal).
func Handler(w http.ResponseWriter, r *http.Request) {
	initOnce.Do(seedData)

	path := strings.TrimSuffix(r.URL.Path, "/")
	switch path {
	case "/api/login":
		handleLogin(w, r)
	case "/api/orders":
		handleOrders(w, r)
	case "/api/orders/status":
		handleOrderStatus(w, r)
	case "/api/profile":
		handleProfile(w, r)
	case "/api/chat":
		handleChat(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan"})
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	var req struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Format tidak valid"})
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	id := strings.TrimSpace(strings.ToLower(req.Identifier))
	cleanInput := strings.ReplaceAll(id, " ", "")
	var found *User
	for email, u := range users {
		cleanPhone := strings.ReplaceAll(u.Phone, " ", "")
		if email == id || cleanPhone == cleanInput {
			found = u
			break
		}
	}

	if found == nil || found.Password != req.Password {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Email/No. WA atau kata sandi salah"})
		return
	}

	redirect := "/pelanggan/beranda"
	if found.Role == "owner" {
		redirect = "/owner/dasbor"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":     found,
		"role":     found.Role,
		"redirect": redirect,
	})
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		emailFilter := strings.ToLower(r.URL.Query().Get("email"))
		mu.RLock()
		defer mu.RUnlock()
		res := []*Order{}
		for _, o := range orders {
			if emailFilter == "" || strings.ToLower(o.CustomerEmail) == emailFilter {
				res = append(res, o)
			}
		}
		writeJSON(w, http.StatusOK, res)

	case http.MethodPost:
		var req struct {
			CustomerName string  `json:"customer_name"`
			Phone        string  `json:"phone"`
			ServiceType  string  `json:"service_type"`
			Package      string  `json:"package"`
			Weight       float64 `json:"weight"`
			Notes        string  `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Data pesanan tidak valid"})
			return
		}
		cleanPhone := strings.ReplaceAll(req.Phone, " ", "")
		if req.CustomerName == "" || len(cleanPhone) < 10 || req.Weight <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Lengkapi nama, WhatsApp minimal 10 digit, dan berat cucian"})
			return
		}

		price := 7000
		est := "2 hari ke depan · 17.00"
		if req.Package == "Ekspres" {
			price = 14000
			est = fmt.Sprintf("Hari ini, %s", time.Now().Add(8*time.Hour).Format("15.04"))
		}

		mu.Lock()
		seqNum++
		newOrder := &Order{
			ID:            fmt.Sprintf("#SK-%d", seqNum),
			CustomerName:  req.CustomerName,
			CustomerEmail: "richard@student.ub.ac.id",
			Phone:         req.Phone,
			ServiceType:   req.ServiceType,
			Package:       req.Package,
			Weight:        req.Weight,
			PricePerKg:    price,
			Total:         int(req.Weight * float64(price)),
			Status:        "Antre",
			PaymentStatus: "Lunas",
			PaymentMethod: "QRIS",
			RackLocation:  fmt.Sprintf("B-%02d", (seqNum%9)+1),
			BagsCount:     1,
			Notes:         req.Notes,
			CreatedAt:     time.Now().Format("02 January 2006 · 15.04 WIB"),
			EstFinish:     est,
			StepTime:      time.Now().Format("02 Jan, 15.04 WIB"),
		}
		orders = append([]*Order{newOrder}, orders...)
		mu.Unlock()
		writeJSON(w, http.StatusCreated, newOrder)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func handleOrderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	var req struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		PaymentStatus string `json:"payment_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Request tidak valid"})
		return
	}

	mu.Lock()
	defer mu.Unlock()
	for _, o := range orders {
		if o.ID == req.ID {
			if req.Status != "" {
				o.Status = req.Status
				o.StepTime = time.Now().Format("02 Jan, 15.04 WIB")
			}
			if req.PaymentStatus != "" {
				o.PaymentStatus = req.PaymentStatus
			}
			writeJSON(w, http.StatusOK, o)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Pesanan tidak ditemukan"})
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	var req struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		NotifEnd bool   `json:"notif_end"`
		NotifChg bool   `json:"notif_chg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Request tidak valid"})
		return
	}

	mu.Lock()
	defer mu.Unlock()
	if u, ok := users[strings.ToLower(req.Email)]; ok {
		if req.Phone != "" {
			u.Phone = req.Phone
		}
		u.NotifEnd = req.NotifEnd
		u.NotifChg = req.NotifChg
		writeJSON(w, http.StatusOK, u)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "User tidak ditemukan"})
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	var req struct {
		Message string `json:"message"`
		Email   string `json:"email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	msg := strings.ToLower(req.Message)

	mu.RLock()
	var active *Order
	for _, o := range orders {
		if strings.ToLower(o.CustomerEmail) == strings.ToLower(req.Email) && o.Status != "Selesai" {
			active = o
			break
		}
	}
	mu.RUnlock()

	reply := "Halo! Aku Asisten SUKI Laundry. Kamu bisa tanya status cucian, daftar harga, jam buka, atau estimasi selesai."
	switch {
	case strings.Contains(msg, "status") || strings.Contains(msg, "sampai mana") || strings.Contains(msg, "estimasi"):
		if active != nil {
			reply = fmt.Sprintf("Cucian %s saat ini berstatus '%s' sejak %s. Estimasi siap: %s. Nanti kami kabari via WhatsApp ya!", active.ID, active.Status, active.StepTime, active.EstFinish)
		} else {
			reply = "Saat ini belum ada cucian aktif atas akunmu. Yuk antar cucian ke outlet Dinoyo!"
		}
	case strings.Contains(msg, "harga") || strings.Contains(msg, "tarif"):
		reply = "Daftar harga SUKI Laundry: Reguler (2 hari) Rp7.000/kg, Ekspres (8 jam) Rp14.000/kg. Tambahan pewangi premium Rp2.000, dan ada diskon mahasiswa Rp2.000!"
	case strings.Contains(msg, "jam") || strings.Contains(msg, "buka") || strings.Contains(msg, "lokasi"):
		reply = "SUKI Laundry Dinoyo (Jl. MT Haryono 82, Malang) buka setiap hari pukul 08.00–20.00 WIB."
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"reply": reply,
		"time":  time.Now().Format("15.04"),
	})
}