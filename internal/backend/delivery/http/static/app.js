// AquaMind — дашборд удалённого мониторинга.
// Без регистрации: выбираешь устройство из списка, смотришь состояние,
// графики истории и алерты. Данные берутся с Backend по HTTP.

const API = "/api/v1";

const state = {
  clientId: null,
  range: "24h",
  timer: null,
};

// --- Утилиты ---------------------------------------------------------------

function fmt(v, digits = 2) {
  if (v === null || v === undefined || Number.isNaN(v)) return "—";
  return Number(v).toFixed(digits);
}

function fmtTime(iso) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString("ru-RU");
}

async function getJSON(url) {
  const resp = await fetch(url);
  if (!resp.ok) throw new Error(`${resp.status} ${resp.statusText}`);
  return resp.json();
}

function setStatus(text, kind) {
  const el = document.getElementById("connStatus");
  el.textContent = text;
  el.className = "status status--" + kind;
}

// --- Список устройств ------------------------------------------------------

async function loadClients() {
  const clients = await getJSON(`${API}/clients`);
  const sel = document.getElementById("clientSelect");
  sel.innerHTML = "";
  if (!clients || clients.length === 0) {
    const opt = document.createElement("option");
    opt.textContent = "Нет устройств";
    opt.value = "";
    sel.appendChild(opt);
    return;
  }
  for (const c of clients) {
    const opt = document.createElement("option");
    opt.value = c.client_id;
    opt.textContent = `${c.client_id} (${fmtTime(c.last_seen)})`;
    sel.appendChild(opt);
  }
  state.clientId = clients[0].client_id;
  sel.value = state.clientId;
}

// --- Текущее состояние -----------------------------------------------------

async function loadState() {
  if (!state.clientId) return;
  try {
    const t = await getJSON(`${API}/state?client_id=${encodeURIComponent(state.clientId)}`);
    const r = t.reading;
    document.getElementById("valTemp").textContent = fmt(r.temp_c, 1);
    document.getElementById("valFlow").textContent = fmt(r.flow_lpm, 2);
    document.getElementById("valO2").textContent = fmt(r.o2_mgl, 2);
    document.getElementById("valPH").textContent = fmt(r.ph, 2);
    document.getElementById("updatedAt").textContent = "Обновлено: " + fmtTime(r.at);

    const a = t.actuator;
    document.getElementById("badgeNormal").classList.toggle("on", a.normal);
    document.getElementById("badgeHeating").classList.toggle("on", a.heating);
    document.getElementById("badgeAlarm").classList.toggle("on", a.alarm);
    document.getElementById("badgeSafe").classList.toggle("on", a.safe_mode);

    setStatus("онлайн", "ok");
  } catch (e) {
    setStatus("нет данных", "err");
    console.error("state:", e);
  }
}

// --- История и график ------------------------------------------------------

function rangeToFrom(range) {
  if (range === "all") return 0;
  const now = Date.now();
  const map = { "1h": 3600e3, "6h": 6 * 3600e3, "24h": 24 * 3600e3 };
  return now - (map[range] || 24 * 3600e3);
}

async function loadHistory() {
  if (!state.clientId) return;
  const from = rangeToFrom(state.range);
  const url = `${API}/ticks?client_id=${encodeURIComponent(state.clientId)}&from=${from}&limit=2000`;
  try {
    const ticks = await getJSON(url);
    // Backend отдаёт свежие первыми — для графика развернём по времени.
    ticks.reverse();
    drawChart(ticks);
  } catch (e) {
    console.error("ticks:", e);
  }
}

// drawChart рисует три линии (температура, кислород, pH) на canvas.
function drawChart(ticks) {
  const canvas = document.getElementById("chart");
  const dpr = window.devicePixelRatio || 1;
  const cssW = canvas.clientWidth;
  const cssH = 260;
  canvas.width = cssW * dpr;
  canvas.height = cssH * dpr;
  const ctx = canvas.getContext("2d");
  ctx.scale(dpr, dpr);
  ctx.clearRect(0, 0, cssW, cssH);

  if (!ticks || ticks.length === 0) {
    ctx.fillStyle = "#8b9bb0";
    ctx.font = "14px system-ui";
    ctx.fillText("Нет данных за период", 12, 24);
    return;
  }

  const pad = { l: 44, r: 12, t: 12, b: 24 };
  const w = cssW - pad.l - pad.r;
  const h = cssH - pad.t - pad.b;

  const series = [
    { key: "temp_c", color: "#e67e22", label: "°C" },
    { key: "o2_mgl", color: "#3498db", label: "O₂" },
    { key: "ph", color: "#2ecc71", label: "pH" },
  ];

  // Общий диапазон значений по всем сериям.
  let min = Infinity, max = -Infinity;
  for (const t of ticks) {
    for (const s of series) {
      const v = t.reading[s.key];
      if (v < min) min = v;
      if (v > max) max = v;
    }
  }
  if (min === max) { min -= 1; max += 1; }
  const padV = (max - min) * 0.1;
  min -= padV; max += padV;

  const x = (i) => pad.l + (ticks.length === 1 ? w / 2 : (i / (ticks.length - 1)) * w);
  const y = (v) => pad.t + h - ((v - min) / (max - min)) * h;

  // Сетка.
  ctx.strokeStyle = "#26384a";
  ctx.fillStyle = "#8b9bb0";
  ctx.font = "11px system-ui";
  ctx.lineWidth = 1;
  for (let i = 0; i <= 4; i++) {
    const gy = pad.t + (i / 4) * h;
    ctx.beginPath();
    ctx.moveTo(pad.l, gy);
    ctx.lineTo(pad.l + w, gy);
    ctx.stroke();
    const val = max - (i / 4) * (max - min);
    ctx.fillText(val.toFixed(1), 4, gy + 4);
  }

  // Линии.
  for (const s of series) {
    ctx.strokeStyle = s.color;
    ctx.lineWidth = 2;
    ctx.beginPath();
    ticks.forEach((t, i) => {
      const px = x(i);
      const py = y(t.reading[s.key]);
      if (i === 0) ctx.moveTo(px, py);
      else ctx.lineTo(px, py);
    });
    ctx.stroke();
  }

  // Легенда.
  let lx = pad.l;
  ctx.font = "12px system-ui";
  for (const s of series) {
    ctx.fillStyle = s.color;
    ctx.fillRect(lx, 2, 10, 10);
    ctx.fillStyle = "#e6edf3";
    ctx.fillText(s.label, lx + 14, 11);
    lx += 60;
  }
}

// --- Алерты ----------------------------------------------------------------

async function loadAlerts() {
  if (!state.clientId) return;
  const url = `${API}/alerts?client_id=${encodeURIComponent(state.clientId)}&limit=100`;
  try {
    const alerts = await getJSON(url);
    const body = document.getElementById("alertsBody");
    body.innerHTML = "";
    if (!alerts || alerts.length === 0) {
      body.innerHTML = '<tr><td colspan="5" class="empty">Нет алертов</td></tr>';
      return;
    }
    for (const a of alerts) {
      const tr = document.createElement("tr");
      tr.innerHTML = `
        <td>${fmtTime(a.at)}</td>
        <td>${a.type}</td>
        <td class="sev-${a.severity}">${a.severity}</td>
        <td>${a.message}</td>
        <td class="state-${a.state}">${a.state}</td>`;
      body.appendChild(tr);
    }
  } catch (e) {
    console.error("alerts:", e);
  }
}

// --- Обновление и события --------------------------------------------------

async function refreshAll() {
  await Promise.all([loadState(), loadHistory(), loadAlerts()]);
}

function startAutoRefresh() {
  if (state.timer) clearInterval(state.timer);
  state.timer = setInterval(refreshAll, 5000);
}

function bindEvents() {
  document.getElementById("clientSelect").addEventListener("change", (e) => {
    state.clientId = e.target.value;
    refreshAll();
  });

  document.querySelectorAll(".range-btn").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.querySelectorAll(".range-btn").forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
      state.range = btn.dataset.range;
      loadHistory();
    });
  });

  window.addEventListener("resize", () => loadHistory());
}

async function init() {
  bindEvents();
  try {
    await loadClients();
    await refreshAll();
    startAutoRefresh();
  } catch (e) {
    setStatus("ошибка связи", "err");
    console.error("init:", e);
  }
}

init();
