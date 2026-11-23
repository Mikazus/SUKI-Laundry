const owner = JSON.parse(localStorage.getItem('suki_user') || 'null');
if (!owner || owner.role !== 'owner') {
  window.location.href = '/login.html';
}
if (document.getElementById('ownerName')) {
  document.getElementById('ownerName').textContent = owner.name;
}

let orders = [];
let selectedPkg = 'Ekspres';
let selectedPayment = 'QRIS';
let qrPollTimer = null;

const fmtRp = (n) => 'Rp' + Number(n || 0).toLocaleString('id-ID');

function formatTodayIndo() {
  return new Date().toLocaleDateString('id-ID', {
    weekday: 'long', day: 'numeric', month: 'long', year: 'numeric'
  });
}

function formatShortToday() {
  return new Date().toLocaleDateString('id-ID', {
    weekday: 'long', day: 'numeric', month: 'short', year: 'numeric'
  });
}

function formatWeekRange() {
  const end = new Date();
  const start = new Date();
  start.setDate(end.getDate() - 6);
  const s = start.toLocaleDateString('id-ID', { day: 'numeric', month: 'short' });
  const e = end.toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' });
  return `${s}–${e}`;
}

function getGreetingIndo() {
  const h = new Date().getHours();
  if (h < 11) return 'Selamat pagi';
  if (h < 15) return 'Selamat siang';
  if (h < 18) return 'Selamat sore';
  return 'Selamat malam';
}

function getLast7DaysChart() {
  const amounts = [
    { label: '900.000', h: 55 },
    { label: '1.100.000', h: 72 },
    { label: '800.000', h: 48 },
    { label: '1.200.000', h: 80 },
    { label: '1.500.000', h: 100 },
    { label: '1.310.000', h: 86 },
    { label: '1.240.000', h: 82 }
  ];
  return amounts.map((item, idx) => {
    const d = new Date();
    d.setDate(d.getDate() - (6 - idx));
    const dayName = d.toLocaleDateString('id-ID', { weekday: 'short' });
    return { ...item, day: dayName, active: idx === 6 };
  });
}

if (document.getElementById('sidebarDate')) {
  document.getElementById('sidebarDate').textContent = formatTodayIndo();
}

async function loadOrders() {
  try {
    const res = await fetch('/api/orders');
    orders = await res.json();
  } catch (e) {
    orders = [];
  }
  refreshCurrentViewData();
  lucide.createIcons();
}

async function switchPage(page) {
  closeQRModal();

  document.querySelectorAll('.menu-btn').forEach(btn => {
    const active = btn.dataset.menu === page;
    btn.className = active
      ? 'menu-btn w-full flex items-center gap-3 px-4 py-3 rounded-2xl bg-tealbrand text-white font-bold text-sm transition cursor-pointer'
      : 'menu-btn w-full flex items-center gap-3 px-4 py-3 rounded-2xl text-ink hover:bg-canvas/50 font-semibold text-sm transition cursor-pointer';
  });

  try {
    const res = await fetch(`/owner/pages/${page}.html`);
    const html = await res.text();
    document.getElementById('main-content').innerHTML = html;
    refreshCurrentViewData();
    lucide.createIcons();
  } catch (err) {
    console.error('Gagal memuat halaman:', err);
  }
}

function refreshCurrentViewData() {
  if (document.getElementById('kpiGrid')) renderDasbor();
  if (document.getElementById('newWeight')) syncPesananBaruUI();
  if (document.getElementById('kanbanBoard')) renderKanban();
  if (document.getElementById('verifyDetailContainer')) findVerifyOrder();
  if (document.getElementById('rekapKPI')) renderRekap();
}

function renderBars(elId) {
  const el = document.getElementById(elId);
  if (!el) return;
  const chartData = getLast7DaysChart();
  el.innerHTML = chartData.map(d => `
    <div class="flex flex-col items-center gap-1.5 h-full justify-end">
      <span class="text-[11px] font-bold text-ink">${d.label}</span>
      <div class="w-full max-w-[48px] rounded-lg ${d.active ? 'bg-tealbrand' : 'bg-aqua'}" style="height:${d.h}%"></div>
      <span class="text-[11px] ${d.active ? 'font-bold text-ink' : 'text-inksoft'}">${d.day}</span>
    </div>
  `).join('');
}

function renderDasbor() {
  const greetEl = document.getElementById('dasborGreeting');
  if (greetEl) greetEl.textContent = `${getGreetingIndo()}, Bu ${owner.name.split(' ')[0]}`;

  const subDateEl = document.getElementById('dasborSubDate');
  if (subDateEl) subDateEl.textContent = `Ringkasan operasional laundry Anda · ${formatTodayIndo()}.`;

  const weekEl = document.getElementById('weekRangeLabel');
  if (weekEl) weekEl.textContent = formatWeekRange();

  const diproses = orders.filter(o => o.status === 'Dicuci' || o.status === 'Disetrika').length;
  const siap = orders.filter(o => o.status === 'Siap Diambil').length;
  const ekspres = orders.filter(o => o.package === 'Ekspres' && o.status !== 'Selesai').length;
  const totalRev = orders.reduce((s, o) => s + o.total, 0);

  const kpiGrid = document.getElementById('kpiGrid');
  if (kpiGrid) {
    kpiGrid.innerHTML = `
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs text-inksoft font-semibold">Pesanan Aktif</div>
        <div class="text-3xl font-extrabold text-ink mt-1.5">${orders.length}</div>
        <div class="text-[11px] text-inksoft mt-1">Tercatat di sistem</div>
      </div>
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs text-inksoft font-semibold">Sedang Diproses</div>
        <div class="text-3xl font-extrabold text-ink mt-1.5">${diproses}</div>
        <div class="text-[11px] text-inksoft mt-1">Dicuci & disetrika</div>
      </div>
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs text-inksoft font-semibold">Siap Diambil</div>
        <div class="text-3xl font-extrabold text-ink mt-1.5">${siap}</div>
        <div class="text-[11px] text-inksoft mt-1">Tersimpan rapi di rak</div>
      </div>
      <div class="bg-white rounded-3xl p-5 shadow-sm">
        <div class="text-xs text-inksoft font-semibold">Total Transaksi</div>
        <div class="text-2xl font-extrabold text-ink mt-2">${fmtRp(totalRev)}</div>
        <div class="text-[11px] text-inksoft mt-1">Dari ${orders.length} pesanan</div>
      </div>
    `;
  }

  const expressLabel = document.getElementById('expressCountLabel');
  if (expressLabel) expressLabel.textContent = `${String(ekspres).padStart(2, '0')} pesanan ekspres`;
  renderBars('weeklyChart');

  const recentTable = document.getElementById('recentOrdersTable');
  if (recentTable) {
    recentTable.innerHTML = orders.slice(0, 5).map(o => `
      <tr class="hover:bg-canvas/20 cursor-pointer" onclick="openVerify('${o.id}')">
        <td class="py-3.5 px-4 font-bold text-ink">${o.id}</td>
        <td class="py-3.5 px-4">${o.customer_name}</td>
        <td class="py-3.5 px-4">${o.service_type} · ${o.package}</td>
        <td class="py-3.5 px-4 font-semibold">${fmtRp(o.total)}</td>
        <td class="py-3.5 px-4 font-semibold">${o.status}</td>
      </tr>
    `).join('');
  }
}

function styleRadioPair(activeBtnId, inactiveBtnId, activeDotId, inactiveDotId) {
  const activeBtn = document.getElementById(activeBtnId);
  const inactiveBtn = document.getElementById(inactiveBtnId);
  const activeDot = document.getElementById(activeDotId);
  const inactiveDot = document.getElementById(inactiveDotId);
  if (!activeBtn || !inactiveBtn || !activeDot || !inactiveDot) return;

  activeBtn.className = 'p-4 rounded-2xl bg-aqua border border-aqua text-left transition cursor-pointer';
  inactiveBtn.className = 'p-4 rounded-2xl bg-white border border-borderline text-left transition cursor-pointer';
  activeDot.className = 'w-4 h-4 rounded-full border-[5px] border-ink bg-white inline-block shrink-0 transition-all';
  inactiveDot.className = 'w-4 h-4 rounded-full border-2 border-ink bg-transparent inline-block shrink-0 transition-all';
}

function syncPesananBaruUI() {
  const hdr = document.getElementById('sumOrderHeader');
  if (hdr) hdr.textContent = `#SK-BARU · ${formatShortToday()}`;
  selectPkg(selectedPkg);
  selectPayment(selectedPayment);
}

function selectPkg(pkg) {
  selectedPkg = pkg;
  if (pkg === 'Reguler') {
    styleRadioPair('pkgReguler', 'pkgEkspres', 'dotReguler', 'dotEkspres');
  } else {
    styleRadioPair('pkgEkspres', 'pkgReguler', 'dotEkspres', 'dotReguler');
  }
  calcNewOrder();
}

function selectPayment(method) {
  selectedPayment = method;
  if (method === 'QRIS') {
    styleRadioPair('payQRIS', 'payTunai', 'dotQRIS', 'dotTunai');
  } else {
    styleRadioPair('payTunai', 'payQRIS', 'dotTunai', 'dotQRIS');
  }
  calcNewOrder();
}

function calcNewOrder() {
  const wEl = document.getElementById('newWeight');
  if (!wEl) return;
  const w = parseFloat(wEl.value) || 0;
  const rate = selectedPkg === 'Ekspres' ? 14000 : 7000;
  const total = Math.round(w * rate);
  const phoneClean = (document.getElementById('newPhone')?.value || '').replace(/\s+/g, '');

  const phoneWarn = document.getElementById('phoneWarn');
  if (phoneWarn) {
    phoneWarn.textContent = phoneClean.length >= 10
      ? '✓ Nomor WhatsApp valid.'
      : '! Nomor WhatsApp minimal 10 digit.';
  }
  if (document.getElementById('sumService')) document.getElementById('sumService').textContent = document.getElementById('newService').value;
  if (document.getElementById('sumWeight')) document.getElementById('sumWeight').textContent = `${w.toFixed(1).replace('.', ',')} kg`;
  if (document.getElementById('sumPkg')) document.getElementById('sumPkg').textContent = `Paket ${selectedPkg}`;
  if (document.getElementById('sumRate')) document.getElementById('sumRate').textContent = `${fmtRp(rate)}/kg`;
  if (document.getElementById('sumPay')) document.getElementById('sumPay').textContent = selectedPayment;
  if (document.getElementById('sumSub')) document.getElementById('sumSub').textContent = fmtRp(total);
  if (document.getElementById('sumTotal')) document.getElementById('sumTotal').textContent = fmtRp(total);

  const sumEst = document.getElementById('sumEst');
  if (sumEst) {
    if (selectedPkg === 'Ekspres') {
      const est = new Date(Date.now() + 8 * 3600 * 1000);
      const jam = est.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' });
      sumEst.textContent = `Hari ini · ${jam} WIB`;
    } else {
      const est = new Date(Date.now() + 48 * 3600 * 1000);
      const tgl = est.toLocaleDateString('id-ID', { day: 'numeric', month: 'short' });
      sumEst.textContent = `${tgl} · 17.00 WIB`;
    }
  }
}

async function submitNewOrder() {
  const payload = {
    customer_name: document.getElementById('newName').value.trim(),
    phone: document.getElementById('newPhone').value.trim(),
    service_type: document.getElementById('newService').value,
    package: selectedPkg,
    payment_method: selectedPayment,
    weight: parseFloat(document.getElementById('newWeight').value) || 0,
    notes: document.getElementById('newNotes').value.trim()
  };
  const res = await fetch('/api/orders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  const data = await res.json();
  if (!res.ok) {
    alert(data.error || 'Gagal menyimpan pesanan');
    return;
  }
  await loadOrders();
  await switchPage('antrean');
  showToast();
}

const nextMap = { 'Antre': 'Dicuci', 'Dicuci': 'Disetrika', 'Disetrika': 'Siap Diambil' };

function renderKanban() {
  const board = document.getElementById('kanbanBoard');
  if (!board) return;
  const filter = document.getElementById('filterPkg')?.value || 'ALL';
  const cols = ['Antre', 'Dicuci', 'Disetrika', 'Siap Diambil'];

  board.innerHTML = cols.map(col => {
    let items = orders.filter(o => o.status === col && (filter === 'ALL' || o.package === filter));
    items.sort((a, b) => (a.package === 'Ekspres' ? -1 : 1) - (b.package === 'Ekspres' ? -1 : 1));
    const isReady = col === 'Siap Diambil';

    return `
      <div class="rounded-3xl border border-borderline/80 p-3 space-y-3 bg-canvas/40">
        <div class="flex items-center justify-between px-3.5 py-2.5 rounded-2xl ${isReady ? 'bg-aqua' : 'bg-white'}">
          <span class="font-bold text-xs text-ink">${col}</span>
          <span class="font-extrabold text-xs text-ink">${items.length}</span>
        </div>
        ${items.map(o => `
          <div class="bg-white rounded-2xl p-4 shadow-sm space-y-2">
            <div class="text-[11px] font-bold text-ink">${o.id}</div>
            ${o.package === 'Ekspres' ? `<span class="inline-block px-2.5 py-0.5 rounded-full bg-aqua text-ink text-[10px] font-bold">Ekspres</span>` : ''}
            <div class="font-extrabold text-sm text-ink">${o.customer_name}</div>
            <div class="text-[11px] text-inksoft">${o.weight} kg · ${o.service_type}</div>
            <div class="text-[11px] text-inksoft">${isReady ? `Rak ${o.rack_location} · ${o.payment_method}` : o.est_finish}</div>
            ${isReady ? `
              <button onclick="openVerify('${o.id}')" class="w-full mt-2 py-2 rounded-xl border border-tealbrand/50 text-ink hover:bg-aqua/40 font-bold text-[11px] transition cursor-pointer">
                Verifikasi pengambilan →
              </button>
            ` : `
              <button onclick="advanceOrder('${o.id}', '${nextMap[col]}')" class="w-full mt-2 py-2 rounded-xl border border-tealbrand/50 text-ink hover:bg-tealbrand hover:text-white font-bold text-[11px] transition cursor-pointer">
                Pindah ke tahap berikutnya
              </button>
            `}
          </div>
        `).join('')}
      </div>
    `;
  }).join('');
}

async function advanceOrder(id, nextStatus) {
  const res = await fetch('/api/orders/status', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id, status: nextStatus })
  });
  if (res.ok) {
    await loadOrders();
    showToast();
  }
}

function showToast() {
  const t = document.getElementById('statusToast');
  if (!t) return;
  t.classList.remove('hidden');
  setTimeout(() => t.classList.add('hidden'), 3000);
}

function openVerify(id) {
  switchPage('verifikasi').then(() => {
    const input = document.getElementById('verifyInput');
    if (input) input.value = id;
    findVerifyOrder();
  });
}

function simulateQRScan() {
  const ready = orders.find(o => o.status === 'Siap Diambil') || orders[0];
  if (ready) {
    document.getElementById('verifyInput').value = ready.id;
    findVerifyOrder();
  }
}

function findVerifyOrder() {
  const box = document.getElementById('verifyDetailContainer');
  if (!box) return;
  const q = (document.getElementById('verifyInput')?.value || '').trim().toLowerCase();
  const o = orders.find(x => x.id.toLowerCase().includes(q) || x.customer_name.toLowerCase().includes(q));

  if (!o) {
    box.innerHTML = `<div class="bg-white rounded-3xl p-8 text-center text-sm text-inksoft">Pesanan tidak ditemukan.</div>`;
    return;
  }

  box.innerHTML = `
    <div class="grid grid-cols-3 gap-5 items-start">
      <div class="col-span-2 bg-white rounded-3xl p-6 shadow-sm space-y-5">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-xl font-extrabold text-ink">${o.id}</div>
            <div class="text-xs text-inksoft mt-0.5">Diterima ${o.created_at}</div>
          </div>
          <span class="px-4 py-1.5 rounded-full bg-aqua text-ink text-xs font-bold">✓ ${o.status}</span>
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

        <div>
          <div class="text-sm font-bold text-ink mb-2">Rincian cucian</div>
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
                <td class="py-3 px-3 font-bold text-ink">${fmtRp(o.total)}</td>
              </tr>
            </tbody>
          </table>
        </div>

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
          <div class="text-2xl font-extrabold text-ink mt-2">${fmtRp(o.total)}</div>
          <div class="text-xs text-inksoft mt-1">${o.payment_method} · ${o.created_at}</div>
        </div>

        <div class="bg-white rounded-3xl p-5 shadow-sm space-y-3">
          <div class="text-sm font-bold text-ink">Siap diserahkan</div>
          <p class="text-xs text-inksoft">Buat QR pengambilan untuk dipindai kamera pelanggan agar status otomatis menjadi Selesai.</p>
          <button onclick="openPickupQR('${o.id}')" ${o.status === 'Selesai' ? 'disabled' : ''}
            class="w-full py-3 rounded-xl bg-tealbrand hover:bg-tealdark disabled:bg-mutedbox disabled:text-inksoft text-white font-bold text-xs transition cursor-pointer">
            ${o.status === 'Selesai' ? 'Sudah Diserahkan (Selesai)' : 'Konfirmasi Pengambilan (Generate QR)'}
          </button>
          <div class="text-[11px] text-inksoft">Status akan berubah menjadi Selesai setelah QR dipindai pelanggan.</div>
        </div>
      </div>
    </div>
  `;
}

async function openPickupQR(id) {
  try {
    const res = await fetch('/api/orders/pickup-qr', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id })
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Gagal membuat QR pengambilan');

    document.getElementById('qrOrderId').textContent = data.id;
    document.getElementById('qrTokenText').textContent = data.token;
    document.getElementById('qrWaitStatus').innerHTML = `
      <span class="w-2 h-2 rounded-full bg-tealbrand animate-ping"></span>
      Menunggu pelanggan memindai QR...
    `;

    const box = document.getElementById('qrCanvasBox');
    box.innerHTML = '';
    new QRCode(box, {
      text: data.qr_payload,
      width: 210,
      height: 210,
      colorDark: '#163338',
      colorLight: '#FFFFFF',
      correctLevel: QRCode.CorrectLevel.M
    });

    document.getElementById('qrModal').classList.remove('hidden');
    lucide.createIcons();

    clearInterval(qrPollTimer);
    qrPollTimer = setInterval(async () => {
      try {
        const pollRes = await fetch('/api/orders');
        const latest = await pollRes.json();
        const found = latest.find(x => x.id === id);
        if (found && found.status === 'Selesai') {
          clearInterval(qrPollTimer);
          orders = latest;
          document.getElementById('qrWaitStatus').innerHTML = `✓ Pengambilan terverifikasi! Status kini Selesai.`;
          setTimeout(() => {
            closeQRModal();
            findVerifyOrder();
          }, 1500);
        }
      } catch (_) {}
    }, 2500);
  } catch (err) {
    alert(err.message);
  }
}

function closeQRModal() {
  clearInterval(qrPollTimer);
  qrPollTimer = null;
  const modal = document.getElementById('qrModal');
  if (modal) modal.classList.add('hidden');
}

function renderRekap() {
  const kpi = document.getElementById('rekapKPI');
  if (!kpi) return;
  const tunai = orders.filter(o => o.payment_method === 'Tunai').reduce((s, o) => s + o.total, 0);
  const nontunai = orders.filter(o => o.payment_method !== 'Tunai').reduce((s, o) => s + o.total, 0);
  const total = tunai + nontunai;

  kpi.innerHTML = `
    <div class="bg-aqua rounded-3xl p-5 shadow-sm">
      <div class="text-xs font-semibold text-ink">Total Pendapatan</div>
      <div class="text-3xl font-extrabold text-ink mt-1.5">${fmtRp(total)}</div>
      <div class="text-xs text-ink/80 mt-1">${orders.length} transaksi tercatat</div>
    </div>
    <div class="bg-white rounded-3xl p-5 shadow-sm">
      <div class="text-xs font-semibold text-inksoft">Pembayaran Tunai</div>
      <div class="text-3xl font-extrabold text-ink mt-1.5">${fmtRp(tunai)}</div>
      <div class="text-xs text-inksoft mt-1">${orders.filter(o => o.payment_method === 'Tunai').length} transaksi</div>
    </div>
    <div class="bg-white rounded-3xl p-5 shadow-sm">
      <div class="text-xs font-semibold text-inksoft">Pembayaran Nontunai</div>
      <div class="text-3xl font-extrabold text-ink mt-1.5">${fmtRp(nontunai)}</div>
      <div class="text-xs text-inksoft mt-1">${orders.filter(o => o.payment_method !== 'Tunai').length} transaksi</div>
    </div>
  `;
  renderBars('rekapChart');
  if (document.getElementById('txCountLabel')) {
    document.getElementById('txCountLabel').textContent = `${orders.length} transaksi · ${formatTodayIndo()}`;
  }
  const rekapTable = document.getElementById('rekapTable');
  if (rekapTable) {
    rekapTable.innerHTML = orders.map(o => `
      <tr>
        <td class="py-3.5 px-4 font-bold text-ink">${o.id}</td>
        <td class="py-3.5 px-4">${o.customer_name}</td>
        <td class="py-3.5 px-4">${o.service_type} · ${o.package}</td>
        <td class="py-3.5 px-4 font-semibold">${fmtRp(o.total)}</td>
        <td class="py-3.5 px-4">${o.payment_status} · ${o.payment_method}</td>
      </tr>
    `).join('');
  }
}

function exportCSV() {
  const header = ['ID', 'Pelanggan', 'Layanan', 'Paket', 'Berat(kg)', 'Total', 'Metode', 'Status'];
  const rows = orders.map(o => [o.id, o.customer_name, o.service_type, o.package, o.weight, o.total, o.payment_method, o.status]);
  const csv = [header, ...rows].map(r => r.join(',')).join('\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = 'rekap-suki-laundry.csv';
  a.click();
}

function searchGlobal() {
  const q = document.getElementById('globalSearch').value.trim();
  if (!q) return;
  switchPage('verifikasi').then(() => {
    const input = document.getElementById('verifyInput');
    if (input) input.value = q;
    findVerifyOrder();
  });
}

function logout() {
  localStorage.removeItem('suki_user');
  window.location.href = '/login.html';
}

// Load initial view (dasbor)
switchPage('dasbor');
loadOrders();