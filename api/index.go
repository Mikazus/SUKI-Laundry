package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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
	PickupToken   string  `json:"pickup_token,omitempty"`
}

var (
	dbMu   sync.Mutex
	db     *sql.DB
	locWIB = time.FixedZone("WIB", 7*3600)
)

var bulanFull = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
var bulanShort = []string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

func nowWIB() time.Time {
	return time.Now().In(locWIB)
}

func fmtFullWIB(t time.Time) string {
	return fmt.Sprintf("%02d %s %d · %02d.%02d WIB", t.Day(), bulanFull[t.Month()], t.Year(), t.Hour(), t.Minute())
}

func fmtShortWIB(t time.Time) string {
	return fmt.Sprintf("%02d %s, %02d.%02d WIB", t.Day(), bulanShort[t.Month()], t.Hour(), t.Minute())
}

func randomToken(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func getDB() (*sql.DB, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	if db != nil {
		return db, nil
	}

	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		return nil, fmt.Errorf("environment variable DATABASE_URL belum diatur")
	}

	// Wajib untuk kompatibilitas PgBouncer (-pooler) di Neon
	if !strings.Contains(dsn, "default_query_exec_mode") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "default_query_exec_mode=simple_protocol"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(5 * time.Minute)

	log.Println("[SUKI] Menghubungkan ke Neon PostgreSQL...")
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("gagal ping Neon (cek jaringan/DATABASE_URL): %w", err)
	}

	if err := initSchemaAndSeed(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("gagal migrasi tabel: %w", err)
	}

	log.Println("[SUKI] Database Neon siap!")
	db = conn
	return db, nil
}

func initSchemaAndSeed(ctx context.Context, conn *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			role TEXT NOT NULL,
			phone TEXT NOT NULL DEFAULT '',
			subtitle TEXT NOT NULL DEFAULT '',
			notif_end BOOLEAN NOT NULL DEFAULT TRUE,
			notif_chg BOOLEAN NOT NULL DEFAULT TRUE
		)`,
		`CREATE SEQUENCE IF NOT EXISTS order_seq START 1049`,
		`CREATE TABLE IF NOT EXISTS orders (
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
			step_time TEXT NOT NULL,
			pickup_token TEXT NOT NULL DEFAULT ''
		)`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS pickup_token TEXT NOT NULL DEFAULT ''`,
	}

	for _, q := range stmts {
		if _, err := conn.ExecContext(ctx, q); err != nil {
			return err
		}
	}

	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

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
		_, err := conn.ExecContext(
			ctx,
			`INSERT INTO users (id, name, email, password, role, phone, subtitle, notif_end, notif_chg)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
			u.ID, u.Name, u.Email, u.Password, u.Role, u.Phone, u.Subtitle, u.NotifEnd, u.NotifChg,
		)
		if err != nil {
			return err
		}
	}

	t := nowWIB()
	seedOrders := []Order{
		{
			ID: "#SK-1048", CustomerName: "Richard Pratama", CustomerEmail: "richard@student.ub.ac.id",
			Phone: "0812 3456 7890", ServiceType: "Cuci, kering & setrika", Package: "Reguler",
			Weight: 4, PricePerKg: 7000, PerfumeFee: 2000, Discount: 2000, Total: 28000,
			Status: "Disetrika", PaymentStatus: "Lunas", PaymentMethod: "QRIS",
			RackLocation: "B-01", BagsCount: 1, Notes: "Pisahkan kemeja putih.",
			CreatedAt: fmtFullWIB(t.Add(-4 * time.Hour)), EstFinish: "Hari ini, 18.00 WIB", StepTime: fmtShortWIB(t.Add(-1 * time.Hour)),
		},
		{
			ID: "#SK-1041", CustomerName: "Richard Pratama", CustomerEmail: "richard@student.ub.ac.id",
			Phone: "0812 3456 7890", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 3, PricePerKg: 7000, Total: 21000, Status: "Siap Diambil",
			PaymentStatus: "Lunas", PaymentMethod: "QRIS", RackLocation: "A-04", BagsCount: 1,
			CreatedAt: fmtFullWIB(t.Add(-48 * time.Hour)), EstFinish: "Siap diambil hari ini", StepTime: fmtShortWIB(t.Add(-2 * time.Hour)),
		},
		{
			ID: "#SK-1027", CustomerName: "Richard Pratama", CustomerEmail: "richard@student.ub.ac.id",
			Phone: "0812 3456 7890", ServiceType: "Cuci + setrika", Package: "Ekspres",
			Weight: 2, PricePerKg: 14000, Total: 28000, Status: "Selesai",
			PaymentStatus: "Lunas", PaymentMethod: "Tunai", RackLocation: "A-01", BagsCount: 1,
			CreatedAt: fmtFullWIB(t.Add(-72 * time.Hour)), EstFinish: "Selesai", StepTime: fmtShortWIB(t.Add(-64 * time.Hour)),
		},
		{
			ID: "#SK-1411-018", CustomerName: "Siti Rahma", CustomerEmail: "siti@mail.com",
			Phone: "0812 8899 0018", ServiceType: "Cuci + setrika", Package: "Reguler",
			Weight: 6, PricePerKg: 7000, Total: 42000, Status: "Siap Diambil",
			PaymentStatus: "Lunas", PaymentMethod: "Tunai", RackLocation: "B-03", BagsCount: 2,
			Notes: "Disimpan dalam 2 kantong terpisah.",
			CreatedAt: fmtFullWIB(t.Add(-24 * time.Hour)), EstFinish: "Siap diambil", StepTime: fmtShortWIB(t.Add(-1 * time.Hour)),
		},
	}

	for i := len(seedOrders) - 1; i >= 0; i-- {
		o := seedOrders[i]
		_, err := conn.ExecContext(
			ctx,
			`INSERT INTO orders (
				id, customer_name, customer_email, phone, service_type, package,
				weight, price_per_kg, perfume_fee, discount, total, status,
				payment_status, payment_method, rack_location, bags_count,
				notes, created_at, est_finish, step_time, pickup_token
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,'')
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
	path := strings.TrimSuffix(r.URL.Path, "/")
	if qPath := strings.TrimPrefix(r.URL.Query().Get("path"), "/"); qPath != "" {
		path = "/api/" + strings.TrimSuffix(qPath, "/")
	}
	log.Printf("[SUKI] %s %s", r.Method, path)

	conn, err := getDB()
	if err != nil {
		log.Printf("[SUKI] DB Error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Koneksi database gagal: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	switch path {
	case "/api/login":
		handleLogin(ctx, w, r, conn)
	case "/api/orders":
		handleOrders(ctx, w, r, conn)
	case "/api/orders/status":
		handleOrderStatus(ctx, w, r, conn)
	case "/api/orders/pickup-qr":
		handleGeneratePickupQR(ctx, w, r, conn)
	case "/api/orders/verify-qr":
		handleVerifyPickupQR(ctx, w, r, conn)
	case "/api/profile":
		handleProfile(ctx, w, r, conn)
	case "/api/chat":
		handleChat(ctx, w, r, conn)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Endpoint tidak ditemukan: " + path})
	}
}

func handleLogin(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *sql.DB) {
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
	err := conn.QueryRowContext(
		ctx,
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

func handleOrders(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *sql.DB) {
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
		                     notes, created_at, est_finish, step_time, pickup_token
		              FROM orders`
		if emailFilter != "" {
			rows, err = conn.QueryContext(ctx, baseQuery+` WHERE LOWER(customer_email) = $1 ORDER BY seq_id DESC`, emailFilter)
		} else {
			rows, err = conn.QueryContext(ctx, baseQuery+` ORDER BY seq_id DESC`)
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
				&o.Notes, &o.CreatedAt, &o.EstFinish, &o.StepTime, &o.PickupToken,
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

		payMethod := strings.TrimSpace(req.PaymentMethod)
		if payMethod != "Tunai" && payMethod != "QRIS" {
			payMethod = "QRIS"
		}

		var nextNum int
		if err := conn.QueryRowContext(ctx, `SELECT nextval('order_seq')`).Scan(&nextNum); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		now := nowWIB()
		price := 7000
		estDate := now.Add(48 * time.Hour)
		est := fmt.Sprintf("%02d %s · 17.00 WIB", estDate.Day(), bulanShort[estDate.Month()])
		if req.Package == "Ekspres" {
			price = 14000
			estTime := now.Add(8 * time.Hour)
			est = fmt.Sprintf("Hari ini, %02d.%02d WIB", estTime.Hour(), estTime.Minute())
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
			PaymentMethod: payMethod,
			RackLocation:  fmt.Sprintf("B-%02d", (nextNum%9)+1),
			BagsCount:     1,
			Notes:         req.Notes,
			CreatedAt:     fmtFullWIB(now),
			EstFinish:     est,
			StepTime:      fmtShortWIB(now),
		}

		_, err := conn.ExecContext(
			ctx,
			`INSERT INTO orders (
				id, customer_name, customer_email, phone, service_type, package,
				weight, price_per_kg, perfume_fee, discount, total, status,
				payment_status, payment_method, rack_location, bags_count,
				notes, created_at, est_finish, step_time, pickup_token
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,'')`,
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

func handleOrderStatus(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *sql.DB) {
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
	err := conn.QueryRowContext(
		ctx,
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

func handleGeneratePickupQR(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Kode pesanan tidak valid"})
		return
	}

	token := "PK-" + randomToken(3)
	res, err := conn.ExecContext(
		ctx,
		`UPDATE orders SET pickup_token = $2 WHERE id = $1 AND status <> 'Selesai'`,
		req.ID, token,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Pesanan tidak ditemukan atau sudah berstatus Selesai"})
		return
	}

	qrPayload := fmt.Sprintf("SUKI|%s|%s", req.ID, token)
	writeJSON(w, http.StatusOK, map[string]string{
		"id":         req.ID,
		"token":      token,
		"qr_payload": qrPayload,
	})
}

func handleVerifyPickupQR(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *sql.DB) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	var req struct {
		QRPayload string `json:"qr_payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Payload QR tidak valid"})
		return
	}

	raw := strings.TrimSpace(req.QRPayload)
	var orderID, token string

	if strings.HasPrefix(raw, "SUKI|") {
		parts := strings.Split(raw, "|")
		if len(parts) == 3 {
			orderID = strings.TrimSpace(parts[1])
			token = strings.TrimSpace(parts[2])
		}
	} else {
		token = strings.ToUpper(raw)
	}

	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Format kode QR tidak dikenali"})
		return
	}

	nowStr := fmtShortWIB(nowWIB())
	var o Order
	var err error

	if orderID != "" {
		err = conn.QueryRowContext(
			ctx,
			`UPDATE orders
			 SET status = 'Selesai', payment_status = 'Lunas', step_time = $3, pickup_token = ''
			 WHERE id = $1 AND pickup_token = $2 AND status <> 'Selesai'
			 RETURNING id, customer_name, status, payment_status, step_time`,
			orderID, token, nowStr,
		).Scan(&o.ID, &o.CustomerName, &o.Status, &o.PaymentStatus, &o.StepTime)
	} else {
		err = conn.QueryRowContext(
			ctx,
			`UPDATE orders
			 SET status = 'Selesai', payment_status = 'Lunas', step_time = $2, pickup_token = ''
			 WHERE pickup_token = $1 AND status <> 'Selesai'
			 RETURNING id, customer_name, status, payment_status, step_time`,
			token, nowStr,
		).Scan(&o.ID, &o.CustomerName, &o.Status, &o.PaymentStatus, &o.StepTime)
	}

	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "QR tidak cocok, kedaluwarsa, atau pesanan sudah selesai"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": fmt.Sprintf("Pengambilan %s berhasil diverifikasi! Status kini Selesai.", o.ID),
		"order":   o,
	})
}

func handleProfile(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *sql.DB) {
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
	err := conn.QueryRowContext(
		ctx,
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

func handleChat(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *sql.DB) {
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
	err := conn.QueryRowContext(
		ctx,
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
		"time":  nowWIB().Format("15.04"),
	})
}