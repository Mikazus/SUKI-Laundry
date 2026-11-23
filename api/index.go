package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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
	CreatedDate   string  `json:"created_date"`
	EstFinish     string  `json:"est_finish"`
	StepTime      string  `json:"step_time"`
	PickupToken   string  `json:"pickup_token,omitempty"`
}

var (
	dbMu   sync.Mutex
	db     *sql.DB
	wibLoc = time.FixedZone("WIB", 7*3600)
)

var indoMonths = []string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

var indoMonthsShort = []string{
	"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun",
	"Jul", "Agu", "Sep", "Okt", "Nov", "Des",
}

func nowWIB() time.Time {
	return time.Now().In(wibLoc)
}

func fmtFullWIB(t time.Time) string {
	return fmt.Sprintf("%d %s %d · %02d.%02d WIB", t.Day(), indoMonths[t.Month()], t.Year(), t.Hour(), t.Minute())
}

func fmtShortWIB(t time.Time) string {
	return fmt.Sprintf("%d %s, %02d.%02d WIB", t.Day(), indoMonthsShort[t.Month()], t.Hour(), t.Minute())
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func getDB() (*sql.DB, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	if db != nil {
		if err := db.Ping(); err == nil {
			return db, nil
		}
		_ = db.Close()
		db = nil
	}

	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
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
		_ = conn.Close()
		return nil, err
	}
	if err := initSchemaAndSeed(conn); err != nil {
		_ = conn.Close()
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
		created_date TEXT NOT NULL DEFAULT '',
		est_finish TEXT NOT NULL,
		step_time TEXT NOT NULL,
		pickup_token TEXT NOT NULL DEFAULT ''
	);

	ALTER TABLE orders ADD COLUMN IF NOT EXISTS created_date TEXT NOT NULL DEFAULT '';
	ALTER TABLE orders ADD COLUMN IF NOT EXISTS pickup_token TEXT NOT NULL DEFAULT '';`

	if _, err := conn.Exec(schema); err != nil {
		return err
	}

	// Isi otomatis created_date bagi data lama yang belum memiliki kolom ini
	todayISO := nowWIB().Format("2006-01-02")
	_, _ = conn.Exec(`UPDATE orders SET created_date = $1 WHERE created_date = ''`, todayISO)

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

	// Seed pesanan awal dengan tanggal dinamis (WIB)
	d0 := nowWIB()
	d1 := d0.AddDate(0, 0, -1)
	d2 := d0.AddDate(0, 0, -2)
	d3 := d0.AddDate(0, 0, -3)
	d5 := d0.AddDate(0, 0, -5)

	seedOrders := []Order{
		{
			ID: "#SK-1048", CustomerName: "Richard Pratama", CustomerEmail: "richard@student.ub.ac.id",
			Phone: "0812 3456 7890", ServiceType: "Cuci, kering & setrika", Package: "Reguler",
			Weight: 4, PricePerKg: 7000, PerfumeFee: 2000, Discount: 2000, Total: 28000,
			Status: "Disetrika", PaymentStatus: "Lunas", PaymentMethod: "QRIS",
			RackLocation: "B-01", BagsCount: 1, Notes: "Pisahkan kemeja putih.",
			CreatedAt: fmtFullWIB(d0), CreatedDate: d0.Format("2006-01-02"), EstFinish: "Hari ini, 18.00 WIB", StepTime: fmtShortWIB(d0),
		},
		{
			ID: "#SK-1041", CustomerName: "Richard Pratama", CustomerEmail: "richard@student.ub.ac.id",
			Phone: "0812 3456 7890", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 3, PricePerKg: 7000, Total: 21000, Status: "Siap Diambil",
			PaymentStatus: "Lunas", PaymentMethod: "QRIS", RackLocation: "A-04", BagsCount: 1,
			CreatedAt: fmtFullWIB(d2), CreatedDate: d2.Format("2006-01-02"), EstFinish: "Siap diambil", StepTime: fmtShortWIB(d0),
		},
		{
			ID: "#SK-1027", CustomerName: "Richard Pratama", CustomerEmail: "richard@student.ub.ac.id",
			Phone: "0812 3456 7890", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 2, PricePerKg: 14000, Total: 28000, Status: "Selesai",
			PaymentStatus: "Lunas", PaymentMethod: "Tunai", RackLocation: "A-01", BagsCount: 1,
			CreatedAt: fmtFullWIB(d5), CreatedDate: d5.Format("2006-01-02"), EstFinish: "Selesai", StepTime: fmtShortWIB(d5),
		},
		{
			ID: "#SK-1611-024", CustomerName: "Nadia Putri", CustomerEmail: "nadia@mail.com",
			Phone: "0812 3450 9988", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 3, PricePerKg: 14000, Total: 42000, Status: "Antre",
			PaymentStatus: "Lunas", PaymentMethod: "QRIS", RackLocation: "C-01", BagsCount: 1,
			Notes: "Pisahkan pakaian putih. Jangan gunakan pewangi.",
			CreatedAt: fmtFullWIB(d0), CreatedDate: d0.Format("2006-01-02"), EstFinish: "Hari ini, 17.00 WIB", StepTime: fmtShortWIB(d0),
		},
		{
			ID: "#SK-1611-022", CustomerName: "Rina Wulandari", CustomerEmail: "rina@mail.com",
			Phone: "0813 9988 1122", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 2, PricePerKg: 14000, Total: 28000, Status: "Antre",
			PaymentStatus: "Lunas", PaymentMethod: "QRIS", RackLocation: "C-02", BagsCount: 1,
			CreatedAt: fmtFullWIB(d0), CreatedDate: d0.Format("2006-01-02"), EstFinish: "Hari ini, 18.00 WIB", StepTime: fmtShortWIB(d0),
		},
		{
			ID: "#SK-1611-020", CustomerName: "Dewi Lestari", CustomerEmail: "dewi@mail.com",
			Phone: "0812 7766 5544", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 4, PricePerKg: 14000, Total: 56000, Status: "Dicuci",
			PaymentStatus: "Lunas", PaymentMethod: "Tunai", RackLocation: "B-02", BagsCount: 2,
			CreatedAt: fmtFullWIB(d1), CreatedDate: d1.Format("2006-01-02"), EstFinish: "Hari ini, 15.00 WIB", StepTime: fmtShortWIB(d0),
		},
		{
			ID: "#SK-1611-023", CustomerName: "Budi Santoso", CustomerEmail: "budi@mail.com",
			Phone: "0857 1122 3344", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 5, PricePerKg: 7000, Total: 35000, Status: "Dicuci",
			PaymentStatus: "Lunas", PaymentMethod: "QRIS", RackLocation: "B-04", BagsCount: 2,
			CreatedAt: fmtFullWIB(d1), CreatedDate: d1.Format("2006-01-02"), EstFinish: fmtShortWIB(d0.AddDate(0, 0, 1)), StepTime: fmtShortWIB(d0),
		},
		{
			ID: "#SK-1611-019", CustomerName: "Andi Pratama", CustomerEmail: "andi@mail.com",
			Phone: "0821 4455 6677", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 3, PricePerKg: 14000, Total: 42000, Status: "Disetrika",
			PaymentStatus: "Lunas", PaymentMethod: "QRIS", RackLocation: "A-05", BagsCount: 1,
			CreatedAt: fmtFullWIB(d3), CreatedDate: d3.Format("2006-01-02"), EstFinish: "Hari ini, 16.00 WIB", StepTime: fmtShortWIB(d0),
		},
		{
			ID: "#SK-1411-018", CustomerName: "Siti Rahma", CustomerEmail: "siti@mail.com",
			Phone: "0812 8899 0018", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 6, PricePerKg: 7000, Total: 42000, Status: "Siap Diambil",
			PaymentStatus: "Lunas", PaymentMethod: "Tunai", RackLocation: "B-03", BagsCount: 2,
			Notes: "Disimpan dalam 2 kantong terpisah.",
			CreatedAt: fmtFullWIB(d2), CreatedDate: d2.Format("2006-01-02"), EstFinish: "Siap diambil", StepTime: fmtShortWIB(d0),
		},
		{
			ID: "#SK-1411-016", CustomerName: "Agus Wijaya", CustomerEmail: "agus@mail.com",
			Phone: "0819 3322 1100", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 3, PricePerKg: 7000, Total: 21000, Status: "Siap Diambil",
			PaymentStatus: "Belum lunas", PaymentMethod: "Tunai", RackLocation: "A-02", BagsCount: 1,
			CreatedAt: fmtFullWIB(d3), CreatedDate: d3.Format("2006-01-02"), EstFinish: "Siap diambil", StepTime: fmtShortWIB(d0),
		},
	}

	for i := len(seedOrders) - 1; i >= 0; i-- {
		o := seedOrders[i]
		_, err := conn.Exec(
			`INSERT INTO orders (
				id, customer_name, customer_email, phone, service_type, package,
				weight, price_per_kg, perfume_fee, discount, total, status,
				payment_status, payment_method, rack_location, bags_count,
				notes, created_at, created_date, est_finish, step_time, pickup_token
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
			ON CONFLICT DO NOTHING`,
			o.ID, o.CustomerName, o.CustomerEmail, o.Phone, o.ServiceType, o.Package,
			o.Weight, o.PricePerKg, o.PerfumeFee, o.Discount, o.Total, o.Status,
			o.PaymentStatus, o.PaymentMethod, o.RackLocation, o.BagsCount,
			o.Notes, o.CreatedAt, o.CreatedDate, o.EstFinish, o.StepTime, "",
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
	case "/api/orders/pickup-qr":
		handleGeneratePickupQR(w, r, conn)
	case "/api/orders/verify-pickup":
		handleVerifyPickupQR(w, r, conn)
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
		                     notes, created_at, created_date, est_finish, step_time, pickup_token
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
				&o.Notes, &o.CreatedAt, &o.CreatedDate, &o.EstFinish, &o.StepTime, &o.PickupToken,
			); err == nil {
				res = append(res, o)
			}
		}
		writeJSON(w, http.StatusOK, res)

	case http.MethodPost:
		var req struct {
			CustomerName  string  `json:"customer_name"`
			Phone         string  `json:"phone"`
			ServiceType   string  `json:"service_type"`
			Package       string  `json:"package"`
			PaymentMethod string  `json:"payment_method"`
			Weight        float64 `json:"weight"`
			Notes         string  `json:"notes"`
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

		// Validasi metode pembayaran QRIS / Tunai
		payMethod := strings.TrimSpace(req.PaymentMethod)
		if payMethod != "QRIS" && payMethod != "Tunai" {
			payMethod = "QRIS"
		}

		var nextNum int
		if err := conn.QueryRow(`SELECT nextval('order_seq')`).Scan(&nextNum); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		now := nowWIB()
		price := 7000
		estDate := now.AddDate(0, 0, 2)
		est := fmt.Sprintf("%d %s · 17.00 WIB", estDate.Day(), indoMonthsShort[estDate.Month()])
		if req.Package == "Ekspres" {
			price = 14000
			estDate = now.Add(8 * time.Hour)
			est = fmt.Sprintf("Hari ini, %02d.%02d WIB", estDate.Hour(), estDate.Minute())
		}

		// Cocokkan email pelanggan dari nomor WA jika terdaftar, default ke akun demo Richard
		custEmail := "richard@student.ub.ac.id"
		var matchedEmail string
		if err := conn.QueryRow(`SELECT email FROM users WHERE REPLACE(phone, ' ', '') = $1 LIMIT 1`, cleanPhone).Scan(&matchedEmail); err == nil {
			custEmail = matchedEmail
		}

		o := Order{
			ID:            fmt.Sprintf("#SK-%d", nextNum),
			CustomerName:  req.CustomerName,
			CustomerEmail: custEmail,
			Phone:         req.Phone,
			ServiceType:   req.ServiceType,
			Package:       req.Package,
			Weight:        req.Weight,
			PricePerKg:    price,
			Total:         int(math.Round(req.Weight * float64(price))),
			Status:        "Antre",
			PaymentStatus: "Lunas",
			PaymentMethod: payMethod,
			RackLocation:  fmt.Sprintf("B-%02d", (nextNum%9)+1),
			BagsCount:     1,
			Notes:         req.Notes,
			CreatedAt:     fmtFullWIB(now),
			CreatedDate:   now.Format("2006-01-02"),
			EstFinish:     est,
			StepTime:      fmtShortWIB(now),
			PickupToken:   "",
		}

		_, err := conn.Exec(
			`INSERT INTO orders (
				id, customer_name, customer_email, phone, service_type, package,
				weight, price_per_kg, perfume_fee, discount, total, status,
				payment_status, payment_method, rack_location, bags_count,
				notes, created_at, created_date, est_finish, step_time, pickup_token
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
			o.ID, o.CustomerName, o.CustomerEmail, o.Phone, o.ServiceType, o.Package,
			o.Weight, o.PricePerKg, o.PerfumeFee, o.Discount, o.Total, o.Status,
			o.PaymentStatus, o.PaymentMethod, o.RackLocation, o.BagsCount,
			o.Notes, o.CreatedAt, o.CreatedDate, o.EstFinish, o.StepTime, o.PickupToken,
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

	nowStr := fmtShortWIB(nowWIB())
	var o Order
	err := conn.QueryRow(
		`UPDATE orders
		 SET status = COALESCE(NULLIF($2, ''), status),
		     payment_status = COALESCE(NULLIF($3, ''), payment_status),
		     step_time = CASE WHEN $2 <> '' THEN $4 ELSE step_time END,
		     pickup_token = CASE WHEN $2 = 'Selesai' THEN '' ELSE pickup_token END
		 WHERE id = $1
		 RETURNING id, customer_name, customer_email, phone, service_type, package,
		           weight, price_per_kg, perfume_fee, discount, total, status,
		           payment_status, payment_method, rack_location, bags_count,
		           notes, created_at, created_date, est_finish, step_time, pickup_token`,
		req.ID, req.Status, req.PaymentStatus, nowStr,
	).Scan(
		&o.ID, &o.CustomerName, &o.CustomerEmail, &o.Phone, &o.ServiceType, &o.Package,
		&o.Weight, &o.PricePerKg, &o.PerfumeFee, &o.Discount, &o.Total, &o.Status,
		&o.PaymentStatus, &o.PaymentMethod, &o.RackLocation, &o.BagsCount,
		&o.Notes, &o.CreatedAt, &o.CreatedDate, &o.EstFinish, &o.StepTime, &o.PickupToken,
	)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Pesanan tidak ditemukan"})
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// POST /api/orders/pickup-qr — Owner generate QR unik untuk pengambilan cucian
func handleGeneratePickupQR(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ID pesanan wajib diisi"})
		return
	}

	token := randomHex(4) // 8 karakter acak
	res, err := conn.Exec(
		`UPDATE orders SET pickup_token = $1 WHERE id = $2 AND status <> 'Selesai'`,
		token, req.ID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Pesanan tidak ditemukan atau sudah berstatus Selesai"})
		return
	}

	payload := fmt.Sprintf("SUKI-PICKUP|%s|%s", req.ID, token)
	writeJSON(w, http.StatusOK, map[string]string{
		"order_id":   req.ID,
		"token":      token,
		"qr_payload": payload,
	})
}

// POST /api/orders/verify-pickup — Pelanggan scan QR di layar Owner -> status menjadi Selesai
func handleVerifyPickupQR(w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	var req struct {
		QRPayload string `json:"qr_payload"`
		Email     string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Payload QR tidak valid"})
		return
	}

	parts := strings.Split(strings.TrimSpace(req.QRPayload), "|")
	if len(parts) != 3 || parts[0] != "SUKI-PICKUP" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Format QR tidak dikenali. Pindai QR yang muncul di layar kasir SUKI Laundry."})
		return
	}
	orderID, token := parts[1], parts[2]

	var dbToken, dbStatus, custName string
	err := conn.QueryRow(
		`SELECT pickup_token, status, customer_name FROM orders WHERE id = $1`,
		orderID,
	).Scan(&dbToken, &dbStatus, &custName)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Pesanan " + orderID + " tidak ditemukan"})
		return
	}
	if dbStatus == "Selesai" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Pesanan " + orderID + " sudah selesai diambil sebelumnya"})
		return
	}
	if dbToken == "" || dbToken != token {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Kode QR tidak cocok atau sudah kadaluarsa. Minta kasir generate ulang QR."})
		return
	}

	stepTime := fmtShortWIB(nowWIB())
	_, err = conn.Exec(
		`UPDATE orders
		 SET status = 'Selesai', payment_status = 'Lunas', step_time = $1, pickup_token = ''
		 WHERE id = $2`,
		stepTime, orderID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message":       fmt.Sprintf("Pengambilan pesanan %s (%s) berhasil diverifikasi!", orderID, custName),
		"order_id":      orderID,
		"status":        "Selesai",
		"customer_name": custName,
	})
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
			reply = fmt.Sprintf("Cucian %s saat ini berstatus '%s' sejak %s. Estimasi siap: %s. Saat mengambil di kasir, gunakan tombol Scan QR Pengambilan ya!", active.ID, active.Status, active.StepTime, active.EstFinish)
		} else {
			reply = "Saat ini belum ada cucian aktif atas akunmu. Yuk antar cucian ke outlet Dinoyo!"
		}
	case strings.Contains(msg, "harga") || strings.Contains(msg, "tarif"):
		reply = "Daftar harga SUKI Laundry: Reguler (2 hari) Rp7.000/kg, Ekspres (8 jam) Rp14.000/kg. Mendukung pembayaran QRIS & Tunai, plus diskon mahasiswa Rp2.000!"
	case strings.Contains(msg, "jam") || strings.Contains(msg, "buka") || strings.Contains(msg, "lokasi"):
		reply = "SUKI Laundry Dinoyo (Jl. MT Haryono 82, Malang) buka setiap hari pukul 08.00–20.00 WIB."
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"reply": reply,
		"time":  nowWIB().Format("15.04"),
	})
}