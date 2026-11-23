package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"-"`
	Role     string `json:"role"`
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
	Package       string  `json:"package"`
	Weight        float64 `json:"weight"`
	PricePerKg    int     `json:"price_per_kg"`
	PerfumeFee    int     `json:"perfume_fee"`
	Discount      int     `json:"discount"`
	Total         int     `json:"total"`
	Status        string  `json:"status"`
	PaymentStatus string  `json:"payment_status"`
	PaymentMethod string  `json:"payment_method"`
	RackLocation  string  `json:"rack_location"`
	BagsCount     int     `json:"bags_count"`
	Notes         string  `json:"notes"`
	CreatedAt     string  `json:"created_at"`
	EstFinish     string  `json:"est_finish"`
	StepTime      string  `json:"step_time"`
}

var (
	dbMu sync.Mutex
	db   *sql.DB
)

func getDB() (*sql.DB, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	if db != nil {
		return db, nil
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("environment variable DATABASE_URL belum diatur")
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(5 * time.Minute)

	if err := conn.Ping(); err != nil {
		return nil, err
	}
	if err := initSchemaAndSeed(conn); err != nil {
		return nil, err
	}

	db = conn
	return db, nil
}

func initSchemaAndSeed(conn *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT NOT NULL,
		phone TEXT NOT NULL DEFAULT '',
		subtitle TEXT NOT NULL DEFAULT '',
		notif_end BOOLEAN NOT NULL DEFAULT TRUE,
		notif_chg BOOLEAN NOT NULL DEFAULT TRUE
	);

	CREATE SEQUENCE IF NOT EXISTS order_seq START 1049;

	CREATE TABLE IF NOT EXISTS orders (
		seq_id SERIAL,
		id TEXT PRIMARY KEY,
		customer_name TEXT NOT NULL,
		customer_email TEXT NOT NULL DEFAULT '',
		phone TEXT NOT NULL,
		service_type TEXT NOT NULL,
		package TEXT NOT NULL,
		weight DOUBLE PRECISION NOT NULL,
		price_per_kg INT NOT NULL,
		perfume_fee INT NOT NULL DEFAULT 0,
		discount INT NOT NULL DEFAULT 0,
		total INT NOT NULL,
		status TEXT NOT NULL DEFAULT 'Antre',
		payment_status TEXT NOT NULL DEFAULT 'Lunas',
		payment_method TEXT NOT NULL DEFAULT 'QRIS',
		rack_location TEXT NOT NULL DEFAULT 'A-01',
		bags_count INT NOT NULL DEFAULT 1,
		notes TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		est_finish TEXT NOT NULL,
		step_time TEXT NOT NULL
	);`

	if _, err := conn.Exec(schema); err != nil {
		return err
	}

	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// Seed akun demo (Pelanggan & Owner)
	seedUsers := []User{
		{
			ID: "USR-01", Name: "Richard Pratama", Email: "richard@student.ub.ac.id",
			Password: "123", Role: "pelanggan", Phone: "0812 3456 7890",
			Subtitle: "Mahasiswa · Universitas Brawijaya", NotifEnd: true, NotifChg: true,
		},
		{
			ID: "OWN-01", Name: "Ayu Pratiwi", Email: "ayu@sukilaundry.id",
			Password: "admin", Role: "owner", Phone: "0811 2233 4455",
			Subtitle: "Pemilik & kasir", NotifEnd: true, NotifChg: true,
		},
	}
	for _, u := range seedUsers {
		_, err := conn.Exec(
			`INSERT INTO users (id, name, email, password, role, phone, subtitle, notif_end, notif_chg)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
			u.ID, u.Name, u.Email, u.Password, u.Role, u.Phone, u.Subtitle, u.NotifEnd, u.NotifChg,
		)
		if err != nil {
			return err
		}
	}

	// Seed pesanan awal (di-insert terbalik agar #SK-1048 punya seq_id paling atas)
	seedOrders := []Order{
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

	for i := len(seedOrders) - 1; i >= 0; i-- {
		o := seedOrders[i]
		_, err := conn.Exec(
			`INSERT INTO orders (
				id, customer_name, customer_email, phone, service_type, package,
				weight, price_per_kg, perfume_fee, discount, total, status,
				payment_status, payment_method, rack_location, bags_count,
				notes, created_at, est_finish, step_time
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
			ON CONFLICT DO NOTHING`,
			o.ID, o.CustomerName, o.CustomerEmail, o.Phone, o.ServiceType, o.Package,
			o.Weight, o.PricePerKg, o.PerfumeFee, o.Discount, o.Total, o.Status,
			o.PaymentStatus, o.PaymentMethod, o.RackLocation, o.BagsCount,
			o.Notes, o.CreatedAt, o.EstFinish, o.StepTime,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := getDB()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Koneksi database gagal: " + err.Error()})
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	switch path {
	case "/api/login":
		handleLogin(w, r, conn)
	case "/api/orders":
		handleOrders(w, r, conn)
	case "/api/orders/status":
		handleOrderStatus(w, r, conn)
	case "/api/profile":
		handleProfile(w, r, conn)
	case "/api/chat":
		handleChat(w, r, conn)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan"})
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
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

	id := strings.TrimSpace(strings.ToLower(req.Identifier))
	cleanPhone := strings.ReplaceAll(id, " ", "")

	var u User
	err := conn.QueryRow(
		`SELECT id, name, email, password, role, phone, subtitle, notif_end, notif_chg
		 FROM users
		 WHERE LOWER(email) = $1 OR REPLACE(phone, ' ', '') = $2
		 LIMIT 1`,
		id, cleanPhone,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.Phone, &u.Subtitle, &u.NotifEnd, &u.NotifChg)

	if err != nil || u.Password != req.Password {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Email/No. WA atau kata sandi salah"})
		return
	}

	redirect := "/pelanggan/beranda"
	if u.Role == "owner" {
		redirect = "/owner/dasbor"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":     u,
		"role":     u.Role,
		"redirect": redirect,
	})
}

func handleOrders(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	switch r.Method {
	case http.MethodGet:
		emailFilter := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email")))
		var (
			rows *sql.Rows
			err  error
		)
		baseQuery := `SELECT id, customer_name, customer_email, phone, service_type, package,
		                     weight, price_per_kg, perfume_fee, discount, total, status,
		                     payment_status, payment_method, rack_location, bags_count,
		                     notes, created_at, est_finish, step_time
		              FROM orders`
		if emailFilter != "" {
			rows, err = conn.Query(baseQuery+` WHERE LOWER(customer_email) = $1 ORDER BY seq_id DESC`, emailFilter)
		} else {
			rows, err = conn.Query(baseQuery + ` ORDER BY seq_id DESC`)
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		res := []Order{}
		for rows.Next() {
			var o Order
			if err := rows.Scan(
				&o.ID, &o.CustomerName, &o.CustomerEmail, &o.Phone, &o.ServiceType, &o.Package,
				&o.Weight, &o.PricePerKg, &o.PerfumeFee, &o.Discount, &o.Total, &o.Status,
				&o.PaymentStatus, &o.PaymentMethod, &o.RackLocation, &o.BagsCount,
				&o.Notes, &o.CreatedAt, &o.EstFinish, &o.StepTime,
			); err == nil {
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

		var nextNum int
		if err := conn.QueryRow(`SELECT nextval('order_seq')`).Scan(&nextNum); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		price := 7000
		est := "2 hari ke depan · 17.00"
		if req.Package == "Ekspres" {
			price = 14000
			est = fmt.Sprintf("Hari ini, %s", time.Now().Add(8*time.Hour).Format("15.04"))
		}

		o := Order{
			ID:            fmt.Sprintf("#SK-%d", nextNum),
			CustomerName:  req.CustomerName,
			CustomerEmail: "richard@student.ub.ac.id",
			Phone:         req.Phone,
			ServiceType:   req.ServiceType,
			Package:       req.Package,
			Weight:        req.Weight,
			PricePerKg:    price,
			Total:         int(math.Round(req.Weight * float64(price))),
			Status:        "Antre",
			PaymentStatus: "Lunas",
			PaymentMethod: "QRIS",
			RackLocation:  fmt.Sprintf("B-%02d", (nextNum%9)+1),
			BagsCount:     1,
			Notes:         req.Notes,
			CreatedAt:     time.Now().Format("02 January 2006 · 15.04 WIB"),
			EstFinish:     est,
			StepTime:      time.Now().Format("02 Jan, 15.04 WIB"),
		}

		_, err := conn.Exec(
			`INSERT INTO orders (
				id, customer_name, customer_email, phone, service_type, package,
				weight, price_per_kg, perfume_fee, discount, total, status,
				payment_status, payment_method, rack_location, bags_count,
				notes, created_at, est_finish, step_time
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
			o.ID, o.CustomerName, o.CustomerEmail, o.Phone, o.ServiceType, o.Package,
			o.Weight, o.PricePerKg, o.PerfumeFee, o.Discount, o.Total, o.Status,
			o.PaymentStatus, o.PaymentMethod, o.RackLocation, o.BagsCount,
			o.Notes, o.CreatedAt, o.EstFinish, o.StepTime,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, o)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func handleOrderStatus(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
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

	nowStr := time.Now().Format("02 Jan, 15.04 WIB")
	var o Order
	err := conn.QueryRow(
		`UPDATE orders
		 SET status = COALESCE(NULLIF($2, ''), status),
		     payment_status = COALESCE(NULLIF($3, ''), payment_status),
		     step_time = CASE WHEN $2 <> '' THEN $4 ELSE step_time END
		 WHERE id = $1
		 RETURNING id, customer_name, customer_email, phone, service_type, package,
		           weight, price_per_kg, perfume_fee, discount, total, status,
		           payment_status, payment_method, rack_location, bags_count,
		           notes, created_at, est_finish, step_time`,
		req.ID, req.Status, req.PaymentStatus, nowStr,
	).Scan(
		&o.ID, &o.CustomerName, &o.CustomerEmail, &o.Phone, &o.ServiceType, &o.Package,
		&o.Weight, &o.PricePerKg, &o.PerfumeFee, &o.Discount, &o.Total, &o.Status,
		&o.PaymentStatus, &o.PaymentMethod, &o.RackLocation, &o.BagsCount,
		&o.Notes, &o.CreatedAt, &o.EstFinish, &o.StepTime,
	)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Pesanan tidak ditemukan"})
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func handleProfile(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
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

	var u User
	err := conn.QueryRow(
		`UPDATE users
		 SET phone = COALESCE(NULLIF($2, ''), phone),
		     notif_end = $3,
		     notif_chg = $4
		 WHERE LOWER(email) = LOWER($1)
		 RETURNING id, name, email, role, phone, subtitle, notif_end, notif_chg`,
		req.Email, req.Phone, req.NotifEnd, req.NotifChg,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Phone, &u.Subtitle, &u.NotifEnd, &u.NotifChg)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "User tidak ditemukan"})
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func handleChat(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
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

	var active Order
	err := conn.QueryRow(
		`SELECT id, status, step_time, est_finish
		 FROM orders
		 WHERE LOWER(customer_email) = LOWER($1) AND status <> 'Selesai'
		 ORDER BY seq_id DESC
		 LIMIT 1`,
		req.Email,
	).Scan(&active.ID, &active.Status, &active.StepTime, &active.EstFinish)
	hasActive := err == nil

	reply := "Halo! Aku Asisten SUKI Laundry. Kamu bisa tanya status cucian, daftar harga, jam buka, atau estimasi selesai."
	switch {
	case strings.Contains(msg, "status") || strings.Contains(msg, "sampai mana") || strings.Contains(msg, "estimasi"):
		if hasActive {
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