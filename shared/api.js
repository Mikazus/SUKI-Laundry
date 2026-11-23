window.SukiAPI = {
  fmtRp(n) {
    return 'Rp' + Number(n || 0).toLocaleString('id-ID');
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
  async parseResponse(res) {
    const raw = await res.text();
    let data = {};
    try {
      data = raw ? JSON.parse(raw) : {};
    } catch {
      throw new Error(`Respons server bukan JSON (HTTP ${res.status}).`);
    }
    if (!res.ok) {
      throw new Error(data.error || `Gagal (HTTP ${res.status})`);
    }
    return data;
  },
  async login(identifier, password) {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ identifier, password })
    });
    return this.parseResponse(res);
  },
  async getOrders(email = '') {
    const q = email ? `?email=${encodeURIComponent(email)}` : '';
    const res = await fetch('/api/orders' + q);
    return this.parseResponse(res);
  },
  async createOrder(payload) {
    const res = await fetch('/api/orders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    return this.parseResponse(res);
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
    return this.parseResponse(res);
  },
  async verifyPickupQR(qr_payload) {
    const res = await fetch('/api/orders/verify-qr', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ qr_payload })
    });
    return this.parseResponse(res);
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
    return this.parseResponse(res);
  }
};