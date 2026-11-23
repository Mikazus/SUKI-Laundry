window.SukiAPI = {
  fmtRp(n) {
    return 'Rp' + Number(n || 0).toLocaleString('id-ID');
  },
  formatTanggalLengkap(date = new Date()) {
    return new Intl.DateTimeFormat('id-ID', {
      weekday: 'long', day: 'numeric', month: 'long', year: 'numeric'
    }).format(date);
  },
  formatTanggalSingkat(date = new Date()) {
    return new Intl.DateTimeFormat('id-ID', {
      day: 'numeric', month: 'short', year: 'numeric'
    }).format(date);
  },
  getLast7Days() {
    const days = [];
    const dayNames = ['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab'];
    for (let i = 6; i >= 0; i--) {
      const d = new Date();
      d.setDate(d.getDate() - i);
      const yyyy = d.getFullYear();
      const mm = String(d.getMonth() + 1).padStart(2, '0');
      const dd = String(d.getDate()).padStart(2, '0');
      days.push({
        iso: `${yyyy}-${mm}-${dd}`,
        shortDay: dayNames[d.getDay()],
        labelFull: `${dayNames[d.getDay()]}, ${d.getDate()} ${new Intl.DateTimeFormat('id-ID', { month: 'short' }).format(d)}`,
        isToday: i === 0
      });
    }
    return days;
  },
  requireRole(expectedRole) {
    const user = JSON.parse(localStorage.getItem('suki_user') || 'null');
    if (!user || user.role !== expectedRole) {
      window.location.href = '/login.html';
      return null;
    }
    return user;
  },
  logout() {
    localStorage.removeItem('suki_user');
    window.location.href = '/login.html';
  },
  async login(identifier, password) {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ identifier, password })
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Gagal masuk');
    return data;
  },
  async getOrders(email = '') {
    const q = email ? `?email=${encodeURIComponent(email)}` : '';
    const res = await fetch('/api/orders' + q);
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Gagal memuat pesanan');
    return data;
  },
  async createOrder(payload) {
    const res = await fetch('/api/orders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Gagal menyimpan pesanan');
    return data;
  },
  async updateOrderStatus(id, status, payment_status = '') {
    const res = await fetch('/api/orders/status', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id, status, payment_status })
    });
    return res.ok;
  },
  async generatePickupQR(id) {
    const res = await fetch('/api/orders/pickup-qr', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id })
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Gagal membuat QR pengambilan');
    return data;
  },
  async verifyPickupQR(qr_payload, email) {
    const res = await fetch('/api/orders/verify-pickup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ qr_payload, email })
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Verifikasi QR gagal');
    return data;
  },
  async updateProfile(payload) {
    const res = await fetch('/api/profile', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    return res.ok;
  },
  async sendChat(message, email) {
    const res = await fetch('/api/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message, email })
    });
    return res.json();
  }
};