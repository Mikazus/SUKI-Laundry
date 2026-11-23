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
    return res.json();
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