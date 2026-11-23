const PelangganApp = (() => {
  const user = SukiAPI.requireRole('pelanggan');
  if (!user) return {};

  const ROUTES = ['beranda', 'lacak', 'nota', 'bantuan', 'profil'];
  const statusConfig = {
    'Antre': { pct: 25, label: 'Sedang antre', desc: 'Pesanan sudah diterima oleh outlet.', step: 0 },
    'Dicuci': { pct: 50, label: 'Sedang dicuci', desc: 'Cucian sedang dibersihkan dengan teliti.', step: 1 },
    'Disetrika': { pct: 75, label: 'Sedang disetrika', desc: 'Sebentar lagi rapi dan siap menemanimu.', step: 2 },
    'Siap Diambil': { pct: 100, label: 'Siap diambil', desc: 'Scan QR kasir saat mengambil cucianmu.', step: 3 },
    'Selesai': { pct: 100, label: 'Selesai', desc: 'Terima kasih sudah mencuci di SUKI Laundry.', step: 4 }
  };

  let orders = [];
  let selectedOrderId = null;
  let html5Scanner = null;
  let isVerifyingQR = false;
  let chatHistory = [
    { sender: 'ai', text: `Halo, ${user.name.split(' ')[0]}! Aku bisa bantu cek cucian, harga, atau jam buka outlet.`, time: new Date().toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' }) }
  ];

  function getActiveOrder() {
    return orders.find(o => o.id === selectedOrderId) || orders[0];
  }

  function getRouteFromURL() {
    const seg = window.location.pathname.replace(/^\/pelanggan\/?/, '').split('/')[0];
    return ROUTES.includes(seg) ? seg : 'beranda';
  }

  function updateNavHighlight(route) {
    document.querySelectorAll('.nav-link').forEach(a => {
      const active = a.dataset.route === route;
      a.querySelector('.nav-pill').className = active
        ? 'nav-pill px-4 py-1 rounded-full bg-tealbrand text-white'
        : 'nav-pill px-4 py-1 rounded-full text-inksoft';
      a.querySelector('.nav-label').className = active
        ? 'nav-label text-[11px] font-bold text-ink'
        : 'nav-label text-[11px] font-medium text-inksoft';
    });
  }

  async function navigate(route, push = true) {
    await closeScanner();
    if (!ROUTES.includes(route)) route = 'beranda';
    if (push) history.pushState({ route }, '', `/pelanggan/${route}`);
    updateNavHighlight(route);

    const html = await fetch(`/pelanggan/pages/${route}.html`).then(r => r.text());
    document.getElementById('app-view').innerHTML = html;

    try {
      orders = await SukiAPI.getOrders(user.email);
    } catch (e) {
      document.getElementById('app-view').innerHTML = `<div class="bg-white rounded-3xl p-6 text-sm text-red-700">${e.message}</div>`;
      return;
    }

    if (!selectedOrderId && orders.length) selectedOrderId = orders[0].id;
    const initializers = {
      beranda: initBeranda,
      lacak: initLacak,
      nota: initNota,
      bantuan: initBantuan,
      profil: initProfil
    };
    initializers[route]?.();
    lucide.createIcons();
  }

  function openOrder(id, targetRoute = 'lacak') {
    selectedOrderId = id;
    navigate(targetRoute);
  }

  // --- KAMERA SCANNER QR ---
  function showScanFeedback(text, isSuccess) {
    const el = document.getElementById('scanFeedback');
    el.textContent = text;
    el.className = isSuccess
      ? 'rounded-2xl px-4 py-3 text-xs font-bold bg-aqua text-ink'
      : 'rounded-2xl px-4 py-3 text-xs font-semibold bg-red-50 border border-red-200 text-red-700';
    el.classList.remove('hidden');
  }

  async function processScannedPayload(payload) {
    if (isVerifyingQR || !payload.trim()) return;
    isVerifyingQR = true;
    showScanFeedback('Memverifikasi kode QR ke kasir...', true);
    try {
      const res = await SukiAPI.verifyPickupQR(payload.trim(), user.email);
      showScanFeedback(`✓ ${res.message}`, true);
      if (html5Scanner) {
        await html5Scanner.stop().catch(() => {});
        html5Scanner = null;
      }
      setTimeout(async () => {
        await closeScanner();
        navigate(getRouteFromURL(), false);
      }, 1500);
    } catch (err) {
      showScanFeedback(err.message, false);
    } finally {
      setTimeout(() => { isVerifyingQR = false; }, 1200);
    }
  }

  async function openScanner() {
    document.getElementById('scanModal').classList.remove('hidden');
    document.getElementById('scanFeedback').classList.add('hidden');
    document.getElementById('manualQrInput').value = '';
    lucide.createIcons();

    try {
      html5Scanner = new Html5Qrcode('qrReader');
      await html5Scanner.start(
        { facingMode: 'environment' },
        { fps: 10, qrbox: { width: 220, height: 220 } },
        (decodedText) => processScannedPayload(decodedText),
        () => {}
      );
    } catch (err) {
      showScanFeedback('Kamera tidak tersedia atau izin ditolak (wajib HTTPS / localhost). Gunakan kolom kode manual di bawah untuk menguji.', false);
    }
  }

  async function closeScanner() {
    if (html5Scanner) {
      await html5Scanner.stop().catch(() => {});
      html5Scanner.clear();
      html5Scanner = null;
    }
    document.getElementById('scanModal')?.classList.add('hidden');
  }

  function submitManualQR() {
    const val = document.getElementById('manualQrInput').value;
    processScannedPayload(val);
  }

  // --- CONTROLLERS ---
  function initBeranda() {
    document.getElementById('greetName').textContent = user.name.split(' ')[0];
    const hero = orders.find(o => o.status !== 'Selesai') || orders[0];
    if (!hero) return;
    const cfg = statusConfig[hero.status] || statusConfig['Antre'];

    document.getElementById('heroOrderCard').innerHTML = `
      <div class="flex items-center justify-between">
        <span class="font-bold text-sm text-ink">${hero.id}</span>
        <span class="px-3.5 py-1 rounded-full bg-white text-ink text-xs font-bold">${hero.package}</span>
      </div>
      <h2 class="text-[26px] font-extrabold text-ink mt-3 leading-tight">${cfg.label}</h2>
      <p class="text-xs text-ink/80 mt-1">${cfg.desc}</p>
      <div class="w-full h-2 bg-white rounded-full overflow-hidden mt-3">
        <div class="h-full bg-tealbrand rounded-full transition-all" style="width:${cfg.pct}%"></div>
      </div>
      <div class="flex items-center justify-between mt-4">
        <div>
          <div class="text-xs text-ink/80">Estimasi selesai</div>
          <div class="text-sm font-bold text-ink mt-0.5">${hero.est_finish}</div>
        </div>
        <i data-lucide="clock" class="w-6 h-6 text-ink"></i>
      </div>
      <button onclick="PelangganApp.openOrder('${hero.id}', 'lacak')" class="w-full mt-5 pt-3 border-t border-ink/10 flex items-center justify-between text-sm font-bold text-ink cursor-pointer">
        <span>Lacak cucianmu</span><i data-lucide="arrow-right" class="w-5 h-5"></i>
      </button>
    `;

    const others = orders.filter(o => o.id !== hero.id);
    document.getElementById('otherOrdersCount').textContent = `${others.length} pesanan`;
    document.getElementById('otherOrdersList').innerHTML = others.map(o => `
      <div onclick="PelangganApp.openOrder('${o.id}', 'lacak')" class="bg-white rounded-3xl p-4 shadow-sm flex items-center justify-between cursor-pointer hover:bg-white/90">
        <div>
          <div class="font-bold text-sm text-ink">${o.id}</div>
          <div class="text-xs text-inksoft mt-0.5">${o.created_at.split('·')[0].trim()} · ${o.package} · ${o.weight} kg</div>
        </div>
        <span class="px-3.5 py-1.5 rounded-full text-xs font-bold ${o.status === 'Siap Diambil' ? 'bg-aqua text-ink' : 'bg-mutedbox text-ink'}">${o.status}</span>
      </div>
    `).join('');
  }

  function initLacak() {
    const o = getActiveOrder();
    if (!o) return;
    const curStep = (statusConfig[o.status] || statusConfig['Antre']).step;

    document.getElementById('trackHeaderCard').innerHTML = `
      <div class="flex items-center justify-between">
        <span class="text-lg font-extrabold text-ink">${o.id}</span>
        <span class="px-3.5 py-1 rounded-full bg-aqua text-ink text-xs font-bold">${o.package}</span>
      </div>
      <div class="text-xs text-inksoft mt-2">${o.weight} kg · ${o.service_type}</div>
    `;

    const steps = [
      { name: 'Antre', time: o.created_at, desc: 'Pesanan diterima oleh outlet.' },
      { name: 'Dicuci', time: 'Proses pencucian', desc: 'Cucian sudah bersih dan kering.' },
      { name: 'Disetrika', time: o.step_time, desc: 'Tim kami sedang merapikan cucian.' },
      { name: 'Siap Diambil', time: curStep >= 3 ? o.step_time : 'Menunggu proses selesai', desc: curStep >= 3 ? `Tersimpan di Rak ${o.rack_location}. Scan QR kasir saat ambil.` : 'Kami kabari saat cucian siap.' }
    ];

    document.getElementById('trackTimeline').innerHTML = steps.map((st, idx) => {
      const done = idx < curStep || o.status === 'Selesai';
      const active = idx === curStep && o.status !== 'Selesai';
      const circleCls = active ? 'bg-tealbrand text-white' : (done ? 'bg-aqua text-ink' : 'bg-mutedbox text-inksoft');
      const iconName = done ? 'check' : (active ? 'shirt' : 'lock');
      const boxCls = active ? 'bg-aqua rounded-2xl p-3.5' : 'py-1';
      return `
        <div class="flex gap-3.5 items-start">
          <div class="flex flex-col items-center">
            <div class="w-9 h-9 rounded-full ${circleCls} flex items-center justify-center shrink-0">
              <i data-lucide="${iconName}" class="w-4 h-4"></i>
            </div>
            ${idx < steps.length - 1 ? `<div class="w-0.5 h-10 ${done ? 'bg-tealbrand' : 'bg-mutedbox'} mt-1"></div>` : ''}
          </div>
          <div class="flex-1 ${boxCls}">
            <div class="font-bold text-sm text-ink">${st.name}</div>
            <div class="text-xs text-ink/80 mt-0.5">${st.time}</div>
            <div class="text-xs text-inksoft mt-0.5">${st.desc}</div>
          </div>
        </div>
      `;
    }).join('');

    document.getElementById('trackEstTime').textContent = o.est_finish;
  }

  function initNota() {
    const o = getActiveOrder();
    if (!o) return;
    const base = o.weight * o.price_per_kg;
    document.getElementById('notaActiveContainer').innerHTML = `
      <div class="bg-white rounded-3xl p-5 shadow-sm space-y-4">
        <div class="flex items-start justify-between">
          <div>
            <div class="text-lg font-extrabold text-ink">${o.id}</div>
            <div class="text-xs text-inksoft mt-0.5">${o.created_at}</div>
          </div>
          <span class="px-3.5 py-1 rounded-full bg-aqua text-ink text-xs font-bold">${o.payment_status} · ${o.payment_method}</span>
        </div>
        <div class="bg-mutedbox/70 rounded-2xl p-3.5">
          <div class="text-sm font-bold text-ink">${o.package} · ${o.service_type}</div>
          <div class="text-xs text-inksoft mt-0.5">Berat cucian: ${o.weight} kg · Status: ${o.status}</div>
        </div>
        <div class="space-y-2 text-sm border-b border-mutedbox pb-3">
          <div class="flex justify-between"><span>${o.weight} kg × ${SukiAPI.fmtRp(o.price_per_kg)}</span><span>${SukiAPI.fmtRp(base)}</span></div>
          ${o.perfume_fee ? `<div class="flex justify-between"><span>Pewangi tambahan</span><span>${SukiAPI.fmtRp(o.perfume_fee)}</span></div>` : ''}
          ${o.discount ? `<div class="flex justify-between"><span>Diskon mahasiswa</span><span>-${SukiAPI.fmtRp(o.discount)}</span></div>` : ''}
        </div>
        <div class="flex items-center justify-between">
          <span class="font-bold text-sm">Total</span>
          <span class="text-2xl font-extrabold text-ink">${SukiAPI.fmtRp(o.total)}</span>
        </div>
      </div>
    `;

    const doneList = orders.filter(x => x.status === 'Selesai');
    document.getElementById('historyCount').textContent = `${doneList.length} pesanan`;
    document.getElementById('notaHistoryList').innerHTML = doneList.map(h => `
      <div onclick="PelangganApp.openOrder('${h.id}', 'nota')" class="bg-white rounded-3xl p-4 shadow-sm flex items-center justify-between cursor-pointer">
        <div>
          <div class="font-bold text-sm text-ink">${h.id}</div>
          <div class="text-xs text-inksoft mt-0.5">${h.created_at.split('·')[0].trim()} · ${h.package} · ${h.weight} kg</div>
        </div>
        <div class="text-right">
          <span class="inline-block px-3 py-0.5 rounded-full bg-aqua text-ink text-xs font-bold mb-1">${h.status}</span>
          <div class="font-extrabold text-sm text-ink">${SukiAPI.fmtRp(h.total)}</div>
        </div>
      </div>
    `).join('');
  }

  function setNotaMode(mode) {
    const segN = document.getElementById('segNota');
    const segR = document.getElementById('segRiwayat');
    const activeBox = document.getElementById('notaActiveContainer');
    const isNota = mode === 'nota';
    segN.className = isNota ? 'py-2.5 rounded-xl bg-tealbrand text-white font-bold text-sm transition cursor-pointer' : 'py-2.5 rounded-xl text-ink font-semibold text-sm transition cursor-pointer';
    segR.className = !isNota ? 'py-2.5 rounded-xl bg-tealbrand text-white font-bold text-sm transition cursor-pointer' : 'py-2.5 rounded-xl text-ink font-semibold text-sm transition cursor-pointer';
    activeBox.classList.toggle('hidden', !isNota);
  }

  function renderChatBox() {
    const box = document.getElementById('chatBox');
    if (!box) return;
    box.innerHTML = chatHistory.map(m => m.sender === 'user' ? `
      <div class="flex justify-end">
        <div class="bg-tealbrand text-white rounded-3xl rounded-tr-md p-4 max-w-[82%] shadow-sm">
          <p class="text-sm font-semibold leading-relaxed">${m.text}</p>
          <div class="text-[11px] text-white/80 mt-1.5">${m.time} · Terkirim</div>
        </div>
      </div>
    ` : `
      <div class="bg-white rounded-3xl rounded-tl-md p-4 max-w-[85%] shadow-sm">
        <p class="text-sm text-ink leading-relaxed">${m.text}</p>
        <div class="text-[11px] text-inksoft mt-2 font-medium">Asisten AI · ${m.time}</div>
      </div>
    `).join('');
  }

  function initBantuan() {
    renderChatBox();
    document.getElementById('chatForm').addEventListener('submit', (e) => {
      e.preventDefault();
      const inp = document.getElementById('chatInput');
      sendChat(inp.value);
      inp.value = '';
    });
  }

  async function sendChat(text) {
    if (!text.trim()) return;
    const now = new Date().toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' });
    chatHistory.push({ sender: 'user', text, time: now });
    renderChatBox();

    const data = await SukiAPI.sendChat(text, user.email);
    chatHistory.push({ sender: 'ai', text: data.reply, time: data.time });
    renderChatBox();
  }

  function initProfil() {
    document.getElementById('profName').textContent = user.name;
    document.getElementById('profSub').textContent = user.subtitle;
    document.getElementById('profEmail').textContent = user.email;
    document.getElementById('profPhone').value = user.phone;
    renderToggles();
  }

  function renderToggles() {
    const tEnd = document.getElementById('toggleEnd');
    const tChg = document.getElementById('toggleChg');
    if (!tEnd || !tChg) return;
    tEnd.className = `w-12 h-7 rounded-full p-1 transition flex cursor-pointer ${user.notif_end ? 'bg-tealbrand justify-end' : 'bg-mutedbox justify-start'}`;
    tChg.className = `w-12 h-7 rounded-full p-1 transition flex cursor-pointer ${user.notif_chg ? 'bg-tealbrand justify-end' : 'bg-mutedbox justify-start'}`;
  }

  async function toggleNotif(which) {
    if (which === 'end') user.notif_end = !user.notif_end;
    if (which === 'chg') user.notif_chg = !user.notif_chg;
    renderToggles();
    await saveProfile();
  }

  async function saveProfile() {
    const inp = document.getElementById('profPhone');
    if (inp) user.phone = inp.value;
    const ok = await SukiAPI.updateProfile({
      email: user.email, phone: user.phone,
      notif_end: user.notif_end, notif_chg: user.notif_chg
    });
    if (ok) {
      localStorage.setItem('suki_user', JSON.stringify(user));
      const msg = document.getElementById('profMsg');
      if (msg) msg.textContent = '✓ Pengaturan akun berhasil disimpan ke Neon DB.';
    }
  }

  document.querySelectorAll('.nav-link').forEach(link => {
    link.addEventListener('click', (e) => {
      e.preventDefault();
      navigate(link.dataset.route);
    });
  });

  window.addEventListener('popstate', () => navigate(getRouteFromURL(), false));
  navigate(getRouteFromURL(), false);

  return {
    navigate, openOrder, setNotaMode, sendChat,
    toggleNotif, saveProfile, openScanner, closeScanner, submitManualQR
  };
})();