const OwnerApp = (() => {
  const owner = SukiAPI.requireRole('owner');
  if (!owner) return {};
  document.getElementById('ownerName').textContent = owner.name;
  document.getElementById('sidebarDate').textContent = SukiAPI.formatTanggalLengkap();

  const ROUTES = ['dasbor', 'pesanan-baru', 'antrean', 'verifikasi', 'rekap'];
  const nextMap = { 'Antre': 'Dicuci', 'Dicuci': 'Disetrika', 'Disetrika': 'Siap Diambil' };

  let orders = [];
  let selectedPkg = 'Ekspres';
  let selectedPayment = 'QRIS';
  let pendingVerifyQuery = '#SK-1048';
  let activeQROrderId = null;
  let qrPollTimer = null;

  function getRouteFromURL() {
    const seg = window.location.pathname.replace(/^\/owner\/?/, '').split('/')[0];
    return ROUTES.includes(seg) ? seg : 'dasbor';
  }

  function updateSidebarHighlight(route) {
    document.querySelectorAll('.menu-link').forEach(a => {
      const active = a.dataset.route === route;
      a.className = active
        ? 'menu-link w-full flex items-center gap-3 px-4 py-3 rounded-2xl bg-tealbrand text-white font-bold text-sm transition'
        : 'menu-link w-full flex items-center gap-3 px-4 py-3 rounded-2xl text-ink hover:bg-canvas/50 font-semibold text-sm transition';
    });
  }

  async function navigate(route, push = true) {
    closeQRModal();
    if (!ROUTES.includes(route)) route = 'dasbor';
    if (push) history.pushState({ route }, '', `/owner/${route}`);
    updateSidebarHighlight(route);

    const html = await fetch(`/owner/pages/${route}.html`).then(r => r.text());
    document.getElementById('owner-view').innerHTML = html;

    try {
      orders = await SukiAPI.getOrders();
    } catch (e) {
      document.getElementById('owner-view').innerHTML = `
        <div class="bg-white rounded-3xl p-8 text-center space-y-2">
          <div class="text-lg font-extrabold text-red-700">Koneksi Neon DB Bermasalah</div>
          <p class="text-sm text-inksoft">${e.message}</p>
        </div>`;
      return;
    }

    const initializers = {
      'dasbor': initDasbor,
      'pesanan-baru': initPesananBaru,
      'antrean': renderKanban,
      'verifikasi': initVerifikasi,
      'rekap': initRekap
    };
    initializers[route]?.();
    lucide.createIcons();
  }

  // Hitung grafik 7 hari terakhir secara dinamis dari data Neon DB
  function buildWeeklyStats() {
    const days = SukiAPI.getLast7Days();
    const totals = days.map(d => {
      const sum = orders
        .filter(o => o.created_date === d.iso)
        .reduce((acc, o) => acc + o.total, 0);
      return { ...d, amount: sum };
    });
    const maxVal = Math.max(...totals.map(t => t.amount), 1);
    return totals.map(t => ({
      ...t,
      heightPct: t.amount > 0 ? Math.max(15, Math.round((t.amount / maxVal) * 100)) : 8
    }));
  }

  function renderBars(elId, stats) {
    document.getElementById(elId).innerHTML = stats.map(d => `
      <div class="flex flex-col items-center gap-1.5 h-full justify-end">
        <span class="text-[10px] font-bold text-ink">${d.amount > 0 ? d.amount.toLocaleString('id-ID') : '0'}</span>
        <div class="w-full max-w-[48px] rounded-lg ${d.isToday ? 'bg-tealbrand' : 'bg-aqua'}" style="height:${d.heightPct}%"></div>
        <span class="text-[11px] ${d.isToday ? 'font-bold text-ink' : 'text-inksoft'}">${d.shortDay}</span>
      </div>
    `).join('');
  }

  // Helper styling kartu radio agar background & titik radio selalu sinkron
  function applyRadioCardState(cardId, dotId, isActive) {
    const card = document.getElementById(cardId);
    const dot = document.getElementById(dotId);
    if (!card || !dot) return;

    card.className = isActive
      ? 'p-4 rounded-2xl border border-aqua bg-aqua text-left transition cursor-pointer'
      : 'p-4 rounded-2xl border border-borderline bg-white text-left transition cursor-pointer';

    dot.className = isActive
      ? 'w-4 h-4 rounded-full border-2 border-ink bg-white inline-flex items-center justify-center shrink-0'
      : 'w-4 h-4 rounded-full border-2 border-inksoft bg-transparent inline-flex items-center justify-center shrink-0';

    dot.innerHTML = isActive ? '<span class="w-2 h-2 rounded-full bg-ink block"></span>' : '';
  }

  // --- DASBOR ---
  function initDasbor() {
    const diproses = orders.filter(o => o.status === 'Dicuci' || o.status === 'Disetrika').length;
    const siap = orders.filter(o => o.status === 'Siap Diambil').length;
    const ekspres = orders.filter(o => o.package === 'Ekspres' && o.status !== 'Selesai').length;
    const totalRev = orders.reduce((s, o) => s + o.total, 0);

    document.getElementById('kpiGrid').innerHTML = `
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs text-inksoft font-semibold">Total Pesanan</div>
        <div class="text-3xl font-extrabold text-ink mt-1.5">${orders.length}</div>
        <div class="text-[11px] text-inksoft mt-1">Tersimpan di Neon DB</div>
      </div>
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs text-inksoft font-semibold">Sedang Diproses</div>
        <div class="text-3xl font-extrabold text-ink mt-1.5">${diproses}</div>
        <div class="text-[11px] text-inksoft mt-1">Dicuci & disetrika</div>
      </div>
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs text-inksoft font-semibold">Siap Diambil</div>
        <div class="text-3xl font-extrabold text-ink mt-1.5">${siap}</div>
        <div class="text-[11px] text-inksoft mt-1">Menunggu scan QR</div>
      </div>
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs text-inksoft font-semibold">Total Pendapatan</div>
        <div class="text-2xl font-extrabold text-ink mt-2">${SukiAPI.fmtRp(totalRev)}</div>
        <div class="text-[11px] text-inksoft mt-1">Dari ${orders.length} transaksi</div>
      </div>
    `;

    const stats = buildWeeklyStats();
    const weekTotal = stats.reduce((s, d) => s + d.amount, 0);
    document.getElementById('dashWeekRange').textContent = `${stats[0].labelFull} – ${stats[6].labelFull}`;
    document.getElementById('dashWeekTotal').textContent = `Total ${SukiAPI.fmtRp(weekTotal)}`;
    document.getElementById('dashWeekAvg').textContent = `Rata-rata ${SukiAPI.fmtRp(Math.round(weekTotal / 7))} per hari`;
    renderBars('weeklyChart', stats);

    document.getElementById('expressCountLabel').textContent = `${String(ekspres).padStart(2, '0')} pesanan ekspres`;
    document.getElementById('recentOrdersTable').innerHTML = orders.slice(0, 5).map(o => `
      <tr class="hover:bg-canvas/20 cursor-pointer" onclick="OwnerApp.openVerify('${o.id}')">
        <td class="py-3.5 px-4 font-bold text-ink">${o.id}</td>
        <td class="py-3.5 px-4">${o.customer_name}</td>
        <td class="py-3.5 px-4">${o.service_type} · ${o.package}</td>
        <td class="py-3.5 px-4"><span class="px-2.5 py-0.5 rounded-full bg-mutedbox/70 text-[11px] font-bold">${o.payment_method}</span></td>
        <td class="py-3.5 px-4 font-semibold">${SukiAPI.fmtRp(o.total)}</td>
        <td class="py-3.5 px-4 font-semibold">${o.status}</td>
      </tr>
    `).join('');
  }

  // --- PESANAN BARU ---
  function initPesananBaru() {
    document.getElementById('sumDateBadge').textContent = `#SK-BARU · ${SukiAPI.formatTanggalLengkap()}`;
    selectPkg('Ekspres');
    selectPayment('QRIS');
  }

  function selectPkg(pkg) {
    selectedPkg = pkg;
    applyRadioCardState('pkgReguler', 'dotReguler', pkg === 'Reguler');
    applyRadioCardState('pkgEkspres', 'dotEkspres', pkg === 'Ekspres');
    calcNewOrder();
  }

  function selectPayment(method) {
    selectedPayment = method;
    applyRadioCardState('payQris', 'dotQris', method === 'QRIS');
    applyRadioCardState('payTunai', 'dotTunai', method === 'Tunai');
    calcNewOrder();
  }

  function calcNewOrder() {
    const w = parseFloat(document.getElementById('newWeight')?.value) || 0;
    const rate = selectedPkg === 'Ekspres' ? 14000 : 7000;
    const total = Math.round(w * rate);
    const phoneClean = (document.getElementById('newPhone')?.value || '').replace(/\s+/g, '');

    document.getElementById('phoneWarn').textContent = phoneClean.length >= 10
      ? '✓ Nomor WhatsApp valid.'
      : '! Nomor WhatsApp minimal 10 digit.';
    document.getElementById('sumService').textContent = document.getElementById('newService').value;
    document.getElementById('sumWeight').textContent = `${w.toFixed(1).replace('.', ',')} kg`;
    document.getElementById('sumPkg').textContent = `Paket ${selectedPkg}`;
    document.getElementById('sumRate').textContent = `${SukiAPI.fmtRp(rate)}/kg`;
    document.getElementById('sumPayMethod').textContent = selectedPayment;
    document.getElementById('sumSub').textContent = SukiAPI.fmtRp(total);
    document.getElementById('sumTotal').textContent = SukiAPI.fmtRp(total);

    const estDate = new Date();
    if (selectedPkg === 'Ekspres') {
      estDate.setHours(estDate.getHours() + 8);
      const jam = estDate.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' }).replace(':', '.');
      document.getElementById('sumEst').textContent = `Hari ini · ${jam} WIB`;
    } else {
      estDate.setDate(estDate.getDate() + 2);
      document.getElementById('sumEst').textContent = `${SukiAPI.formatTanggalSingkat(estDate)} · 17.00 WIB`;
    }
  }

  async function submitNewOrder() {
    try {
      await SukiAPI.createOrder({
        customer_name: document.getElementById('newName').value.trim(),
        phone: document.getElementById('newPhone').value.trim(),
        service_type: document.getElementById('newService').value,
        package: selectedPkg,
        payment_method: selectedPayment,
        weight: parseFloat(document.getElementById('newWeight').value) || 0,
        notes: document.getElementById('newNotes').value.trim()
      });
      await navigate('antrean');
      showToast();
    } catch (err) {
      alert(err.message);
    }
  }

  // --- ANTREAN KERJA ---
  function renderKanban() {
    const filter = document.getElementById('filterPkg')?.value || 'ALL';
    const cols = ['Antre', 'Dicuci', 'Disetrika', 'Siap Diambil'];

    document.getElementById('kanbanBoard').innerHTML = cols.map(col => {
      const items = orders
        .filter(o => o.status === col && (filter === 'ALL' || o.package === filter))
        .sort((a, b) => (a.package === 'Ekspres' ? -1 : 1) - (b.package === 'Ekspres' ? -1 : 1));
      const isReady = col === 'Siap Diambil';

      return `
        <div class="rounded-3xl border border-borderline/80 p-3 space-y-3 bg-canvas/40">
          <div class="flex items-center justify-between px-3.5 py-2.5 rounded-2xl ${isReady ? 'bg-aqua' : 'bg-white'}">
            <span class="font-bold text-xs text-ink">${col}</span>
            <span class="font-extrabold text-xs text-ink">${items.length}</span>
          </div>
          ${items.map(o => `
            <div class="bg-white rounded-2xl p-4 shadow-sm space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-[11px] font-bold text-ink">${o.id}</span>
                <span class="text-[10px] font-semibold text-inksoft">${o.payment_method}</span>
              </div>
              ${o.package === 'Ekspres' ? `<span class="inline-block px-2.5 py-0.5 rounded-full bg-aqua text-ink text-[10px] font-bold">Ekspres</span>` : ''}
              <div class="font-extrabold text-sm text-ink">${o.customer_name}</div>
              <div class="text-[11px] text-inksoft">${o.weight} kg · ${o.service_type}</div>
              <div class="text-[11px] text-inksoft">${isReady ? `Rak ${o.rack_location} · ${o.payment_status}` : o.est_finish}</div>
              ${isReady ? `
                <button onclick="OwnerApp.openVerify('${o.id}')" class="w-full mt-2 py-2 rounded-xl border border-tealbrand/50 text-ink hover:bg-aqua/40 font-bold text-[11px] transition cursor-pointer">
                  Verifikasi pengambilan →
                </button>
              ` : `
                <button onclick="OwnerApp.advanceOrder('${o.id}', '${nextMap[col]}')" class="w-full mt-2 py-2 rounded-xl border border-tealbrand/50 text-ink hover:bg-tealbrand hover:text-white font-bold text-[11px] transition cursor-pointer">
                  Pindah ke tahap berikutnya
                </button>
              `}
            </div>
          `).join('')}
        </div>
      `;
    }).join('');
    lucide.createIcons();
  }

  async function advanceOrder(id, nextStatus) {
    if (await SukiAPI.updateOrderStatus(id, nextStatus)) {
      orders = await SukiAPI.getOrders();
      renderKanban();
      showToast();
    }
  }

  function showToast() {
    const t = document.getElementById('statusToast');
    if (!t) return;
    t.classList.remove('hidden');
    setTimeout(() => t.classList.add('hidden'), 3000);
  }

  // --- VERIFIKASI & GENERATE QR ---
  function openVerify(id) {
    pendingVerifyQuery = id;
    navigate('verifikasi');
  }

  function initVerifikasi() {
    const firstReady = orders.find(o => o.status === 'Siap Diambil');
    if (!orders.some(o => o.id === pendingVerifyQuery) && firstReady) {
      pendingVerifyQuery = firstReady.id;
    }
    document.getElementById('verifyInput').value = pendingVerifyQuery;
    findVerifyOrder();
  }

  function simulateQRScan() {
    const ready = orders.find(o => o.status === 'Siap Diambil') || orders[0];
    if (ready) {
      document.getElementById('verifyInput').value = ready.id;
      findVerifyOrder();
    }
  }

  function findVerifyOrder() {
    const q = (document.getElementById('verifyInput')?.value || '').trim().toLowerCase();
    const o = orders.find(x => x.id.toLowerCase().includes(q) || x.customer_name.toLowerCase().includes(q));
    const box = document.getElementById('verifyDetailContainer');
    if (!o) {
      box.innerHTML = `<div class="bg-white rounded-3xl p-8 text-center text-sm text-inksoft">Pesanan tidak ditemukan.</div>`;
      return;
    }

    const isDone = o.status === 'Selesai';
    box.innerHTML = `
      <div class="grid grid-cols-3 gap-5 items-start">
        <div class="col-span-2 bg-white rounded-3xl p-6 shadow-sm space-y-5">
          <div class="flex items-start justify-between">
            <div>
              <div class="text-xl font-extrabold text-ink">${o.id}</div>
              <div class="text-xs text-inksoft mt-0.5">Diterima ${o.created_at}</div>
            </div>
            <span class="px-4 py-1.5 rounded-full ${isDone ? 'bg-tealbrand text-white' : 'bg-aqua text-ink'} text-xs font-bold">✓ ${o.status}</span>
          </div>
          <div class="bg-mutedbox/70 rounded-2xl p-4 flex items-center justify-between">
            <div>
              <div class="font-bold text-sm text-ink">${o.customer_name}</div>
              <div class="text-xs text-inksoft mt-0.5">WhatsApp · ${o.phone}</div>
            </div>
            <div class="text-right">
              <div class="text-[10px] font-bold uppercase text-inksoft">Paket Layanan</div>
              <div class="text-xs font-bold text-ink mt-0.5">${o.package}</div>
            </div>
          </div>
          <table class="w-full text-xs text-left">
            <thead>
              <tr class="bg-mutedbox/70 text-inksoft text-[11px] uppercase">
                <th class="py-2 px-3 rounded-l-lg">Layanan / Barang</th>
                <th class="py-2 px-3">Berat</th>
                <th class="py-2 px-3 rounded-r-lg">Subtotal</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td class="py-3 px-3 font-bold text-ink">${o.service_type}</td>
                <td class="py-3 px-3">${o.weight} kg</td>
                <td class="py-3 px-3 font-bold text-ink">${SukiAPI.fmtRp(o.total)}</td>
              </tr>
            </tbody>
          </table>
          <div class="bg-aqua rounded-2xl p-4 space-y-2">
            <div class="text-xs font-bold text-ink">Periksa sebelum menyerahkan</div>
            <label class="flex items-center gap-2 text-xs text-ink cursor-pointer">
              <input type="checkbox" checked class="accent-tealbrand w-4 h-4" /> Kode nota sesuai dengan kode pesanan
            </label>
            <label class="flex items-center gap-2 text-xs text-ink cursor-pointer">
              <input type="checkbox" checked class="accent-tealbrand w-4 h-4" /> Jumlah cucian lengkap: ${o.bags_count} kantong
            </label>
          </div>
          <div class="text-xs text-inksoft">Catatan: ${o.notes || 'Tidak ada catatan khusus.'}</div>
        </div>

        <div class="space-y-4">
          <div class="bg-aqua rounded-3xl p-5 shadow-sm">
            <div class="text-[10px] font-bold uppercase text-ink">Lokasi Rak</div>
            <div class="text-4xl font-extrabold text-ink mt-1">${o.rack_location}</div>
            <div class="text-xs text-ink/80 mt-2">Rak ${o.rack_location.split('-')[0]} · ${o.bags_count} kantong</div>
          </div>
          <div class="bg-white rounded-3xl p-5 shadow-sm">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold text-ink">Pembayaran</span>
              <span class="px-3 py-1 rounded-full bg-aqua text-ink text-[11px] font-bold">✓ ${o.payment_status}</span>
            </div>
            <div class="text-2xl font-extrabold text-ink mt-2">${SukiAPI.fmtRp(o.total)}</div>
            <div class="text-xs text-inksoft mt-1">${o.payment_method} · ${o.created_at}</div>
          </div>
          <div class="bg-white rounded-3xl p-5 shadow-sm space-y-3">
            <div class="text-sm font-bold text-ink">Siap diserahkan</div>
            <p class="text-xs text-inksoft">Tampilkan QR agar dipindai pelanggan di halaman /pelanggan untuk menandai selesai.</p>
            <button onclick="OwnerApp.openPickupQR('${o.id}')" ${isDone ? 'disabled' : ''}
              class="w-full py-3 rounded-xl bg-tealbrand hover:bg-tealdark disabled:bg-mutedbox disabled:text-inksoft text-white font-bold text-xs flex items-center justify-center gap-2 transition cursor-pointer">
              <i data-lucide="qr-code" class="w-4 h-4"></i>
              ${isDone ? 'Sudah Diserahkan (Selesai)' : 'Konfirmasi & Generate QR'}
            </button>
          </div>
        </div>
      </div>
    `;
    lucide.createIcons();
  }

  async function openPickupQR(id) {
    try {
      const data = await SukiAPI.generatePickupQR(id);
      activeQROrderId = id;

      document.getElementById('qrOrderTitle').textContent = `Pesanan ${id}`;
      document.getElementById('qrCodeText').textContent = data.qr_payload;
      document.getElementById('qrWaitStatus').classList.remove('hidden');
      document.getElementById('qrDoneBox').classList.add('hidden');
      document.getElementById('qrManualBtn').classList.remove('hidden');

      const qrEl = document.getElementById('qrCanvas');
      qrEl.innerHTML = '';
      new QRCode(qrEl, {
        text: data.qr_payload,
        width: 220,
        height: 220,
        colorDark: '#163338',
        colorLight: '#ffffff',
        correctLevel: QRCode.CorrectLevel.M
      });

      document.getElementById('qrModal').classList.remove('hidden');
      lucide.createIcons();

      // Auto-polling setiap 2.5 detik menunggu pelanggan scan QR
      clearInterval(qrPollTimer);
      qrPollTimer = setInterval(async () => {
        try {
          const latest = await SukiAPI.getOrders();
          const target = latest.find(x => x.id === activeQROrderId);
          if (target && target.status === 'Selesai') {
            clearInterval(qrPollTimer);
            orders = latest;
            document.getElementById('qrWaitStatus').classList.add('hidden');
            document.getElementById('qrDoneBox').classList.remove('hidden');
            document.getElementById('qrManualBtn').classList.add('hidden');
            findVerifyOrder();
          }
        } catch (_) {}
      }, 2500);
    } catch (err) {
      alert(err.message);
    }
  }

  async function forceConfirmManual() {
    if (!activeQROrderId) return;
    if (await SukiAPI.updateOrderStatus(activeQROrderId, 'Selesai', 'Lunas')) {
      closeQRModal();
      orders = await SukiAPI.getOrders();
      findVerifyOrder();
    }
  }

  function closeQRModal() {
    clearInterval(qrPollTimer);
    activeQROrderId = null;
    document.getElementById('qrModal')?.classList.add('hidden');
  }

  // --- REKAP PENDAPATAN ---
  function initRekap() {
    const tunai = orders.filter(o => o.payment_method === 'Tunai').reduce((s, o) => s + o.total, 0);
    const nontunai = orders.filter(o => o.payment_method !== 'Tunai').reduce((s, o) => s + o.total, 0);
    const total = tunai + nontunai;

    document.getElementById('rekapKPI').innerHTML = `
      <div class="bg-aqua rounded-3xl p-5 shadow-sm">
        <div class="text-xs font-semibold text-ink">Total Pendapatan</div>
        <div class="text-3xl font-extrabold text-ink mt-1.5">${SukiAPI.fmtRp(total)}</div>
        <div class="text-xs text-ink/80 mt-1">${orders.length} transaksi tercatat</div>
      </div>
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs font-semibold text-inksoft">Pembayaran Tunai</div>
        <div class="text-3xl font-extrabold text-ink mt-1.5">${SukiAPI.fmtRp(tunai)}</div>
        <div class="text-xs text-inksoft mt-1">${orders.filter(o => o.payment_method === 'Tunai').length} transaksi</div>
      </div>
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs font-semibold text-inksoft">Pembayaran QRIS / Nontunai</div>
        <div class="text-3xl font-extrabold text-ink mt-1.5">${SukiAPI.fmtRp(nontunai)}</div>
        <div class="text-xs text-inksoft mt-1">${orders.filter(o => o.payment_method !== 'Tunai').length} transaksi</div>
      </div>
    `;

    const stats = buildWeeklyStats();
    renderBars('rekapChart', stats);
    const topDay = [...stats].sort((a, b) => b.amount - a.amount)[0];
    document.getElementById('rekapTopDay').innerHTML = `Hari tertinggi 7 hari terakhir: <strong class="text-ink">${topDay.labelFull} · ${SukiAPI.fmtRp(topDay.amount)}</strong>`;

    document.getElementById('txCountLabel').textContent = `${orders.length} transaksi tersimpan`;
    document.getElementById('rekapTable').innerHTML = orders.map(o => `
      <tr>
        <td class="py-3.5 px-4 font-bold text-ink">${o.id}</td>
        <td class="py-3.5 px-4">${o.customer_name}</td>
        <td class="py-3.5 px-4">${o.service_type} · ${o.package}</td>
        <td class="py-3.5 px-4 font-semibold">${SukiAPI.fmtRp(o.total)}</td>
        <td class="py-3.5 px-4">${o.payment_status} · ${o.payment_method}</td>
      </tr>
    `).join('');
  }

  function exportCSV() {
    const header = ['ID', 'Tanggal', 'Pelanggan', 'Layanan', 'Paket', 'Berat(kg)', 'Total', 'Metode', 'Status'];
    const rows = orders.map(o => [o.id, o.created_date, o.customer_name, o.service_type, o.package, o.weight, o.total, o.payment_method, o.status]);
    const csv = [header, ...rows].map(r => r.join(',')).join('\n');
    const a = document.createElement('a');
    a.href = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8;' }));
    a.download = `rekap-suki-laundry-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
  }

  function searchGlobal() {
    const q = document.getElementById('globalSearch').value.trim();
    if (!q) return;
    pendingVerifyQuery = q;
    navigate('verifikasi');
  }

  document.querySelectorAll('.menu-link').forEach(link => {
    link.addEventListener('click', (e) => {
      e.preventDefault();
      navigate(link.dataset.route);
    });
  });

  window.addEventListener('popstate', () => navigate(getRouteFromURL(), false));
  navigate(getRouteFromURL(), false);

  return {
    navigate, selectPkg, selectPayment, calcNewOrder, submitNewOrder,
    renderKanban, advanceOrder, openVerify, simulateQRScan,
    findVerifyOrder, openPickupQR, closeQRModal, forceConfirmManual,
    exportCSV, searchGlobal
  };
})();