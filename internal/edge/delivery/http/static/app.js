"use strict";

// --- Переключение вкладок ---
const tabs = document.querySelectorAll(".tab");
const panels = document.querySelectorAll(".panel");

tabs.forEach((tab) => {
  tab.addEventListener("click", () => {
    tabs.forEach((t) => t.classList.remove("active"));
    panels.forEach((p) => p.classList.remove("active"));
    tab.classList.add("active");
    document.getElementById("panel-" + tab.dataset.tab).classList.add("active");

    // При открытии вкладки «Пороги» подгружаем актуальные значения.
    if (tab.dataset.tab === "thresholds") {
      loadThresholds();
    }
  });
});

// --- Индикатор соединения ---
const connDot = document.getElementById("conn-dot");
const connText = document.getElementById("conn-text");

function setConn(online) {
  connDot.classList.toggle("online", online);
  connDot.classList.toggle("offline", !online);
  connText.textContent = online ? "онлайн" : "нет связи";
}

// --- Дашборд: опрос /api/state ---
const POLL_MS = 2000;

function fmt(v, digits) {
  if (v === null || v === undefined) return "—";
  return Number(v).toFixed(digits);
}

// setText обновляет текст только если он реально изменился —
// чтобы не дёргать DOM и не вызывать перерисовку каждые 2 секунды.
function setText(id, value) {
  const el = document.getElementById(id);
  if (el && el.textContent !== value) el.textContent = value;
}

// Кэш порогов (для индикации «мало/норма/высоко» на дашборде).
let thresholds = null;

// setSensorStatus выставляет подпись и цвет статуса датчика.
// level: "low" | "ok" | "high" | "" (нет данных).
function setSensorStatus(id, level, text) {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = text;
  el.className = "sensor-status" + (level ? " " + level : "");
}

// renderTempStatus — статус температуры по порогам подогрева и критическим.
function renderTempStatus(v) {
  if (!thresholds || v === null || v === undefined) {
    setSensorStatus("st-temp", "", "");
    return;
  }
  if (v <= thresholds.temp_critical_low) {
    setSensorStatus("st-temp", "low", "критически низкая");
  } else if (v < thresholds.temp_low) {
    setSensorStatus("st-temp", "low", "мало");
  } else if (v >= thresholds.temp_critical_high) {
    setSensorStatus("st-temp", "high", "критически высокая");
  } else {
    setSensorStatus("st-temp", "ok", "норма");
  }
}

// renderFlowStatus — статус потока по порогу flow_min.
function renderFlowStatus(v) {
  if (!thresholds || v === null || v === undefined) {
    setSensorStatus("st-flow", "", "");
    return;
  }
  if (v < thresholds.flow_min) {
    setSensorStatus("st-flow", "low", "мало");
  } else {
    setSensorStatus("st-flow", "ok", "норма");
  }
}

// renderO2Status — статус кислорода по порогам o2_low / o2_high.
function renderO2Status(v) {
  if (!thresholds || v === null || v === undefined) {
    setSensorStatus("st-o2", "", "");
    return;
  }
  if (v < thresholds.o2_low) {
    setSensorStatus("st-o2", "low", "мало");
  } else if (v > thresholds.o2_high) {
    setSensorStatus("st-o2", "high", "высоко");
  } else {
    setSensorStatus("st-o2", "ok", "норма");
  }
}

// renderPHStatus — статус pH по порогам ph_low / ph_high.
function renderPHStatus(v) {
  if (!thresholds || v === null || v === undefined) {
    setSensorStatus("st-ph", "", "");
    return;
  }
  if (v < thresholds.ph_low) {
    setSensorStatus("st-ph", "low", "низкий");
  } else if (v > thresholds.ph_high) {
    setSensorStatus("st-ph", "high", "высокий");
  } else {
    setSensorStatus("st-ph", "ok", "норма");
  }
}

function renderState(data) {
  // Показания датчиков.
  const r = data.reading;
  setText("v-temp", r ? fmt(r.temp_c, 1) : "—");
  setText("v-flow", r ? fmt(r.flow_lpm, 1) : "—");
  setText("v-o2", r ? fmt(r.o2_mg_l, 1) : "—");
  setText("v-ph", r ? fmt(r.ph, 2) : "—");

  // Индикация «мало/норма/высоко» для всех датчиков.
  renderTempStatus(r ? r.temp_c : null);
  renderFlowStatus(r ? r.flow_lpm : null);
  renderO2Status(r ? r.o2_mg_l : null);
  renderPHStatus(r ? r.ph : null);

  // Алерты: активные сверху, под ними — история последних событий.
  // История кэшируется и обновляется отдельным таймером (см. ниже).
  lastActiveAlerts = data.alerts || [];
  renderAlerts(lastActiveAlerts, alertHistory);

  // Время обновления.
  const at = new Date(data.at);
  setText("updated-at", at.toLocaleTimeString("ru-RU"));
}

// --- Алерты: активные + история ---
const HISTORY_LIMIT = 6;

function fmtTime(iso) {
  const d = new Date(iso);
  return d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function makeAlertItem(al, kind) {
  // kind: "active" | "history"
  const li = document.createElement("li");
  li.className = "alert-item " + kind;
  if (al.severity === "warning") li.classList.add("warning");

  const msg = document.createElement("span");
  msg.className = "alert-msg";
  msg.textContent = al.message || al.type;
  li.appendChild(msg);

  const meta = document.createElement("span");
  meta.className = "alert-meta";
  if (kind === "active") {
    meta.textContent = "активен";
  } else {
    const stateText = al.state === "cleared" ? "снят" : "поднят";
    meta.textContent = fmtTime(al.at) + " · " + stateText;
  }
  li.appendChild(meta);
  return li;
}

// Кэш истории алертов (обновляется отдельным таймером).
let alertHistory = [];
// Последние активные алерты (из /api/state) — нужны при перерисовке истории.
let lastActiveAlerts = [];
// Подпись последнего отрисованного списка — чтобы не перерисовывать DOM зря.
let alertsSignature = null;

// Подпись состояния алертов: если не изменилась — DOM не трогаем.
function alertsSig(active, history) {
  const part = (arr, kind) =>
    (arr || []).map((a) => kind + "|" + a.type + "|" + a.severity + "|" + a.message + "|" + a.at + "|" + (a.state || "")).join(";");
  return part(active, "a") + "#" + part(history, "h");
}

function renderAlerts(activeAlerts, history) {
  const active = activeAlerts || [];
  const hist = history || [];

  const sig = alertsSig(active, hist);
  if (sig === alertsSignature) return; // ничего не изменилось — не дёргаем DOM
  alertsSignature = sig;

  const list = document.getElementById("alerts-list");
  list.innerHTML = "";

  if (active.length === 0) {
    const li = document.createElement("li");
    li.className = "alerts-empty";
    li.textContent = "Нет активных алертов";
    list.appendChild(li);
  } else {
    active.forEach((al) => list.appendChild(makeAlertItem(al, "active")));
  }

  if (hist.length > 0) {
    const sep = document.createElement("li");
    sep.className = "alerts-sep";
    sep.textContent = "История (последние " + hist.length + ")";
    list.appendChild(sep);
    hist.forEach((al) => list.appendChild(makeAlertItem(al, "history")));
  }
}

// Загрузка истории алертов (отдельный, более редкий таймер).
async function loadAlertHistory() {
  try {
    const resp = await fetch("/api/alerts?limit=" + HISTORY_LIMIT, { cache: "no-store" });
    if (!resp.ok) return;
    alertHistory = await resp.json();
    // Перерисуем с уже известными активными алертами (из последнего /api/state).
    renderAlerts(lastActiveAlerts, alertHistory);
  } catch (e) {
    // тихо игнорируем — индикатор соединения покажет проблему
  }
}

async function pollState() {
  try {
    const resp = await fetch("/api/state", { cache: "no-store" });
    if (!resp.ok) throw new Error("HTTP " + resp.status);
    const data = await resp.json();
    renderState(data);
    setConn(true);
  } catch (e) {
    setConn(false);
  }
}

// --- Пороги ---
const thresholdsForm = document.getElementById("thresholds-form");
const thresholdsMsg = document.getElementById("thresholds-msg");

async function loadThresholds() {
  try {
    const resp = await fetch("/api/thresholds", { cache: "no-store" });
    if (!resp.ok) throw new Error("HTTP " + resp.status);
    const t = await resp.json();
    thresholds = t; // кэш для индикации на дашборде
    for (const key of Object.keys(t)) {
      const input = thresholdsForm.elements[key];
      if (input) input.value = t[key];
    }
    // Успешная загрузка — сбрасываем возможное старое сообщение об ошибке.
    thresholdsMsg.textContent = "";
    thresholdsMsg.className = "form-msg";
  } catch (e) {
    // Фоновая подгрузка: не пугаем пользователя ошибкой.
    // Реальную проблему он увидит при попытке сохранить.
  }
}

// Фоновая загрузка порогов (для индикации на дашборде, без формы).
async function refreshThresholdsCache() {
  try {
    const resp = await fetch("/api/thresholds", { cache: "no-store" });
    if (resp.ok) thresholds = await resp.json();
  } catch (e) {
    // тихо игнорируем
  }
}

thresholdsForm.addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const payload = {};
  for (const el of thresholdsForm.elements) {
    if (el.name) payload[el.name] = parseFloat(el.value);
  }
  try {
    const resp = await fetch("/api/thresholds", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    const data = await resp.json();
    if (!resp.ok) throw new Error(data.error || "HTTP " + resp.status);
    thresholds = data; // обновляем кэш для дашборда
    thresholdsMsg.textContent = "Сохранено";
    thresholdsMsg.className = "form-msg ok";
  } catch (e) {
    thresholdsMsg.textContent = "Ошибка: " + e.message;
    thresholdsMsg.className = "form-msg err";
  }
});

// --- Старт ---
const HISTORY_POLL_MS = 5000;

pollState();
loadAlertHistory();
refreshThresholdsCache();
setInterval(pollState, POLL_MS);
setInterval(loadAlertHistory, HISTORY_POLL_MS);
setInterval(refreshThresholdsCache, HISTORY_POLL_MS);
