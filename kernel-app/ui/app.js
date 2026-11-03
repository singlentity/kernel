/* ================= KERNEL UI ================= */
const I = { // иконки
  rocket:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 15c-1.5 1.5-2 5-2 5s3.5-.5 5-2M9 11a5 5 0 0 1 7-7c3 0 4 1 4 4a5 5 0 0 1-7 7l-2 2-2-2 2-2M12 8h.01" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  home:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 10.5 12 3l9 7.5M5 9.5V20h14V9.5" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  bolt:'<svg viewBox="0 0 24 24" fill="currentColor"><path d="M13 2 4 14h6l-1 8 9-12h-6z"/></svg>',
  broom:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 4 9 14M14 4l6 6M8 13l-4 7 7-4M5 19l2-5" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  grid:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/></svg>',
  globe:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c3 3.5 3 14 0 18M12 3c-3 3.5-3 14 0 18"/></svg>',
  box:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 7.5 12 3l9 4.5v9L12 21 3 16.5z"/><path d="M3 7.5 12 12l9-4.5M12 12v9" stroke-linejoin="round"/></svg>',
  shield:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6z" stroke-linejoin="round"/></svg>',
  chip:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="6" y="6" width="12" height="12" rx="2"/><path d="M9 2v3M15 2v3M9 19v3M15 19v3M2 9h3M2 15h3M19 9h3M19 15h3"/></svg>',
  list:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01" stroke-linecap="round"/></svg>',
  cpu:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="7" y="7" width="10" height="10" rx="1.5"/><path d="M10 2v3M14 2v3M10 19v3M14 19v3M2 10h3M2 14h3M19 10h3M19 14h3"/></svg>',
  gpu:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="6" width="20" height="12" rx="2"/><circle cx="8" cy="12" r="2.5"/><circle cx="15" cy="12" r="2.5"/></svg>',
  ram:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="8" width="20" height="9" rx="1.5"/><path d="M6 17v3M10 17v3M14 17v3M18 17v3M6 11v3M10 11v3M14 11v3M18 11v3"/></svg>',
  disk:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="3"/></svg>',
  win:'<svg viewBox="0 0 24 24" fill="currentColor"><path d="M3 5.5 11 4v7H3zM13 3.7 21 2.5V11h-8zM3 13h8v7l-8-1.5zM13 13h8v8.5L13 20z"/></svg>',
  warn:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3 2 20h20z" stroke-linejoin="round"/><path d="M12 9v5M12 17h.01" stroke-linecap="round"/></svg>',
  check:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4"><path d="M4 12l5 5L20 6" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  x:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4"><path d="M6 6l12 12M18 6 6 18" stroke-linecap="round"/></svg>',
  dl:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3v12m0 0 4-4m-4 4-4-4M4 19h16" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  ext:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5" stroke-linecap="round" stroke-linejoin="round"/></svg>',
};

const TABS = [
  {id:'home', name:'Главная', ic:I.home},
  {id:'fresh', name:'Старт', ic:I.rocket},
  {id:'boost', name:'Ускорение', ic:I.bolt},
  {id:'clean', name:'Очистка', ic:I.broom},
  {id:'uwp', name:'Приложения', ic:I.grid},
  {id:'edge', name:'Edge', ic:I.globe},
  {id:'apps', name:'Программы', ic:I.box},
  {id:'shield', name:'Защита', ic:I.shield},
  {id:'drivers', name:'Драйверы', ic:I.chip},
  {id:'log', name:'Журнал', ic:I.list},
];

/* ---- мост к Go (или мок для превью) ---- */
const hasGo = typeof window.sysInfo === 'function';
async function go(name, ...args){
  if (typeof window[name] === 'function') return await window[name](...args);
  return MOCK[name] ? MOCK[name](...args) : null;
}

let STATE = {tab:'home', uwp:[], tweaks:[], clean:[]};

/* ================= РЕНДЕР ================= */
function renderTabs(){
  document.getElementById('tabs').innerHTML = TABS.map(t=>
    `<div class="tab ${t.id===STATE.tab?'active':''}" data-tab="${t.id}" title="${t.name}">${t.ic}<span>${t.name}</span></div>`
  ).join('');
  document.querySelectorAll('.tab').forEach(el=>el.onclick=()=>switchTab(el.dataset.tab));
}
function switchTab(id){ STATE.tab=id; renderTabs(); renderPage(); checkUpdates(); }

const LOADING = {
  clean:'Сканирую мусор…', uwp:'Загружаю приложения Windows…', fresh:'Загружаю настройки…', 
  drivers:'Опрашиваю устройства…', shield:'Проверяю статус защиты…', home:'Читаю сведения о системе…'
};
async function renderPage(){
  const c = document.getElementById('content');
  const msg = LOADING[STATE.tab];
  c.innerHTML = '<div class="page active">'+(msg?`<div class="loader"><div class="sp"></div>${msg}</div>`:'')+'</div>';
  const p = c.firstChild;
  if (STATE.tab==='home') return renderHome(p);
  if (STATE.tab==='fresh') return renderFresh(p);
  if (STATE.tab==='boost') return renderBoost(p);
  if (STATE.tab==='clean') return renderClean(p);
  if (STATE.tab==='uwp') return renderUwp(p);
  if (STATE.tab==='edge') return renderEdge(p);
  if (STATE.tab==='apps') return renderApps(p);
  if (STATE.tab==='shield') return renderShield(p);
  if (STATE.tab==='drivers') return renderDrivers(p);
  if (STATE.tab==='log') return renderLog(p);
}

/* -------- ГЛАВНАЯ -------- */
async function renderFresh(p){
  const items = await go('freshList') || [];
  const row = f=>`
    <div class="tweak" data-id="${f.id}">
      <div class="main">
        <div class="title-row"><span class="t-name">${f.name}</span>${f.applied?'<span class="badge applied">применено</span>':''}</div>
        <div class="t-desc">${f.desc}</div>
      </div>
      <label class="switch"><input type="checkbox" data-fr="${f.id}" ${f.on?'checked':''}><span class="slider"></span></label>
    </div>`;
  p.innerHTML = `
  <div class="page-head">
    <h1>Только поставил Windows? Начни отсюда</h1>
    <p>Это моя личная базовая настройка, которую я делаю на каждой свежей системе после переустановки. Windows из коробки завалена рекламой в Пуске, мусорными значками на рабочем столе и неудобными мелочами. Одной кнопкой приводим всё в человеческий вид — чисто, по делу, как у меня. Можно включить всё сразу или выбрать по пунктам.</p>
  </div>
  <div style="display:flex;gap:10px;flex-wrap:wrap;margin-bottom:18px">
    <button class="btn gold" onclick="doFreshAll()">${I.rocket} Сделать всё сразу</button>
    <button class="btn ghost" onclick="doFreshSelected()">${I.check} Применить выбранное</button>
    <button class="btn dark" onclick="doWallpaper()">${I.home} Поставить фон</button>
  </div>
  ${items.map(row).join('')}`;
}
function balanceCard(b){
  const kinds={gpu:['Упор в видеокарту','#E8C15A'],cpu:['Упор в процессор','#E8A15A'],balanced:['Связка сбалансирована','#7CD67C'],unknown:['Железо не распознано','#9A9AA2']};
  const [title,color]=kinds[b.kind||'unknown'];
  const bar=(label,tier)=>`
    <div class="bl-row">
      <div class="bl-top"><span>${label}</span><span class="bl-pct">${tier?tier+'/100':'—'}</span></div>
      <div class="bl-track"><i style="width:${tier||0}%;background:${color}"></i></div>
    </div>`;
  return `
  <div class="balance">
    <div class="bl-head"><span class="bl-dot" style="background:${color}"></span>${title}</div>
    ${bar('Процессор', b.cpuTier)}
    ${bar('Видеокарта', b.gpuTier)}
    <div class="bl-verdict">${b.verdict||''}</div>
    <div class="bl-note">Зависит от игры, разрешения и настроек графики.</div>
  </div>`;
}
async function renderHome(p){
  const s = await go('sysInfo');
  const bal = await go('balance') || {};
  p.innerHTML = `
  <div class="hero">
    <div class="hero-main">
      <h2>Один клик — и система настроена навсегда</h2>
      <p>KERNEL применит безопасный набор постоянных оптимизаций: отключит фоновую запись игр, лишнюю телеметрию и ненужные службы, настроит питание на максимум. Все изменения переживают перезагрузку и в любой момент откатываются.</p>
      <button class="btn gold" onclick="runBoostAll()">${I.bolt} Настроить всё автоматически</button>
    </div>
    ${balanceCard(bal)}
  </div>
  <div class="section-label">Ваш компьютер</div>
  <div class="specs">
    <div class="spec"><div class="ic">${I.cpu}</div><div class="lbl">Процессор</div><div class="val">${s.cpu}</div></div>
    <div class="spec"><div class="ic">${I.gpu}</div><div class="lbl">Видеокарта</div><div class="val">${String(s.gpu).replace(/ \+ /g,'<br>')}</div></div>
    <div class="spec"><div class="ic">${I.ram}</div><div class="lbl">Память</div><div class="val">${s.ram}</div></div>
    <div class="spec"><div class="ic">${I.disk}</div><div class="lbl">Система</div><div class="val">${s.os} <small>${s.build}</small></div></div>
  </div>
  <div class="tg-card" onclick="openTg()">
    <div class="ti"><svg viewBox="0 0 24 24" fill="currentColor"><path d="M21.9 4.3 18.6 19.8c-.2 1.1-.9 1.4-1.8.9l-5-3.7-2.4 2.3c-.3.3-.5.5-1 .5l.4-5.1 9.3-8.4c.4-.4-.1-.6-.6-.2L5.9 13.3l-5-1.6c-1.1-.3-1.1-1.1.2-1.6L20.6 2.6c.9-.3 1.7.2 1.3 1.7z"/></svg></div>
    <div><b>Telegram-канал kernel1lab</b><span>Обновления KERNEL, гайды по оптимизации и новые версии</span></div>
    <div class="go">t.me/kernel1lab →</div>
  </div>
  <div class="foot-note">KERNEL by kernel · изменения записываются в журнал и могут быть отменены</div>`;
}

/* -------- УСКОРЕНИЕ -------- */
async function renderBoost(p){
  const tw = await go('tweaks'); STATE.tweaks=tw;
  const groups = [...new Set(tw.map(t=>t.group))];
  let body = groups.map(g=>`
    <div class="section-label">${g}</div>
    ${tw.filter(t=>t.group===g).map(t=>tweakRow(t)).join('')}
  `).join('');
  p.innerHTML = `
  <div class="page-head"><h1>Ускорение и FPS</h1>
    <p>Постоянные настройки, которые повышают производительность в играх и убирают фоновую нагрузку. Каждая переживает перезагрузку. Наведите переключатель — и готово.</p></div>
  <div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin-bottom:18px">
    <button class="btn gold" onclick="applySelectedTweaks()">${I.check} Применить выбранное</button>
    <button class="btn ghost" onclick="revertTweaks()">Откатить всё</button>
    <span class="proc-count" id="procCount"></span>
  </div>
  ${body}`;
  go('procCount').then(n=>{ const e=document.getElementById('procCount'); if(e&&n) e.innerHTML='Сейчас запущено процессов: <b>'+n+'</b>'; });
  bindSwitches();
  refreshVbsStatus();
}
function tweakRow(t){
  const vbsExtra = t.vbs ? `
      <div class="vbs-extra">
        <span class="vbs-status" id="vbs-status">VBS: проверяю…</span>
        <a class="link-more" onclick="toggleDetails('${t.id}',this)">Подробнее</a>
        <button class="btn ghost sm" onclick="doVbsRevert()">Вернуть как было</button>
      </div>
      <div class="t-details" id="det-${t.id}" style="display:none">${(t.details||'').replace(/\n/g,'<br>')}</div>` : '';
  return `<div class="tweak" data-id="${t.id}">
    <div class="main">
      <div class="title-row">
        <span class="t-name">${t.name}</span>
        <span class="badge ${t.reboot?'reboot':'perm'}">${t.reboot?'после перезагрузки':'постоянно'}</span>
        ${t.applied?'<span class="badge applied">применено</span>':''}
      </div>
      <div class="t-desc">${t.desc}</div>
      ${vbsExtra}
    </div>
    <label class="switch"><input type="checkbox" data-tw="${t.id}" ${(t.on||t.applied)?'checked':''}><span class="slider"></span></label>
  </div>`;
}
function toggleDetails(id,a){
  const d=document.getElementById('det-'+id); if(!d) return;
  const show=d.style.display==='none';
  d.style.display=show?'block':'none';
  if(a) a.textContent=show?'Скрыть':'Подробнее';
}
async function refreshVbsStatus(){
  const el=document.getElementById('vbs-status'); if(!el) return;
  try{ const s=await go('vbsStatus');
    el.textContent='VBS: '+((s&&s.text)||'—');
    el.className='vbs-status '+(s&&s.status===2?'on':(s&&s.status===0?'off':''));
  }catch(e){ el.textContent='VBS: —'; }
}
/* Диалог-подтверждение в стиле оверлея */
function confirmDialog(title,text,okLabel){
  return new Promise(resolve=>{
    const ov=document.getElementById('confirmOverlay');
    document.getElementById('cf-icon').className='p-icon warn';
    document.getElementById('cf-icon').innerHTML=I.warn;
    document.getElementById('cf-title').textContent=title;
    document.getElementById('cf-text').textContent=text;
    const ok=document.getElementById('cf-ok'),cancel=document.getElementById('cf-cancel');
    ok.textContent=okLabel||'Продолжить';
    const done=v=>{ ov.classList.remove('show'); ok.onclick=null; cancel.onclick=null; resolve(v); };
    ok.onclick=()=>done(true); cancel.onclick=()=>done(false);
    ov.classList.add('show');
  });
}
async function doVbsRevert(){
  await act('Возвращаю VBS…','Восстанавливаю исходное состояние защиты.', ()=>go('vbsRevert'), true);
}

/* -------- ОЧИСТКА -------- */
async function renderClean(p){
  const items = await go('cleanScan'); STATE.clean=items;
  const total = items.reduce((a,b)=>a+b.bytes,0);
  p.innerHTML = `
  <div class="page-head"><h1>Очистка системы</h1>
    <p>Мусор, который безопасно удалить: временные файлы, кэши, журналы обновлений и корзина. Можно включить авто-очистку раз в неделю — тогда запускать KERNEL не придётся.</p></div>
  <div class="hero" style="margin-bottom:18px">
    <div class="hero-main" style="padding:24px">
      <h2 style="font-size:18px">Найдено ${fmtSize(total)} мусора</h2>
      <p style="margin-top:6px">Отметьте категории и нажмите «Очистить». Файлы удаляются безвозвратно, но все они временные.</p>
      <div style="display:flex;gap:10px;margin-top:18px">
        <button class="btn gold" onclick="runClean()">${I.broom} Очистить отмеченное</button>
        <button class="btn ghost" onclick="doToggleAutoClean()">Авто-очистка: выкл</button>
      </div>
    </div>
  </div>
  ${items.map(c=>`
    <div class="tweak" data-id="${c.id}">
      <div class="main">
        <div class="title-row"><span class="t-name">${c.name}</span><span class="badge perm">${fmtSize(c.bytes)}</span></div>
        <div class="t-desc">${c.desc}</div>
      </div>
      <label class="switch"><input type="checkbox" data-cl="${c.id}" ${c.on?'checked':''}><span class="slider"></span></label>
    </div>`).join('')}`;
}

/* -------- UWP (две колонки) -------- */
async function renderUwp(p){
  const list = await go('uwpList'); STATE.uwp=list;
  const installed = list.filter(a=>a.installed).length;
  p.innerHTML = `
  <div class="page-head"><h1>Приложения Windows (UWP)</h1>
    <p>Только встроенные приложения Microsoft. Отмеченные будут удалены. Приглушённые уже не установлены. Снимите галочку с того, что хотите оставить.</p></div>
  <div class="uwp-toolbar">
    <button class="btn ghost sm" onclick="uwpAll(true)">Отметить все</button>
    <button class="btn ghost sm" onclick="uwpAll(false)">Снять все</button>
    <button class="btn ghost sm" onclick="uwpPreset()">Только мусор</button>
    <span class="count"><b id="uwp-sel">0</b> выбрано · ${installed} установлено</span>
  </div>
  <div class="uwp-grid" id="uwp-grid">
    ${list.map(a=>`
      <div class="uwp-item ${a.installed?'':'off'} ${a.sel?'sel':''}" data-pkg="${a.id}" ${a.installed?'':'data-dis="1"'}>
        <div class="chk"></div>
        <span class="nm">${a.name}</span>
        ${a.installed?'':'<span class="tag">не установлено</span>'}
      </div>`).join('')}
  </div>
  <div style="margin-top:18px">
    <button class="btn gold" onclick="doRemoveUwp()">${I.x} Удалить отмеченные</button>
  </div>`;
  document.querySelectorAll('.uwp-item').forEach(el=>{
    if(el.dataset.dis) return;
    el.onclick=()=>{el.classList.toggle('sel');updateUwpCount();};
  });
  updateUwpCount();
}
function updateUwpCount(){
  const n=document.querySelectorAll('.uwp-item.sel').length;
  const e=document.getElementById('uwp-sel'); if(e)e.textContent=n;
}
function uwpAll(v){document.querySelectorAll('.uwp-item:not([data-dis])').forEach(el=>el.classList.toggle('sel',v));updateUwpCount();}
function uwpPreset(){document.querySelectorAll('.uwp-item:not([data-dis])').forEach(el=>{
  const junk=STATE.uwp.find(a=>a.id===el.dataset.pkg)?.junk; el.classList.toggle('sel',!!junk);});updateUwpCount();}

/* -------- EDGE / STORE / WEBVIEW -------- */
async function renderEdge(p){
  const st = await go('edgeStatus') || {};
  const chip = on => on ? '<span class="st-chip on"><i></i>Установлен</span>' : '<span class="st-chip off"><i></i>Удалён</span>';
  const btn = (k,on,label) => on ? `<button class="btn gold" onclick="doRemoveComp('${k}')">${I.x} ${label}</button>`
                                 : `<button class="btn dark" disabled>${I.check} Уже удалён</button>`;
  p.innerHTML = `
  <div class="page-head"><h1>Microsoft Edge · Store</h1>
    <p>Эти компоненты удаляются по отдельности их штатными деинсталляторами. После удаления статус сменится на «Удалён». Перед удалением Edge поставьте другой браузер на вкладке «Программы».</p></div>
  <div class="trio duo">
    <div class="card">
      <div class="ic">${I.globe}</div>${chip(st.edge)}<h3>Microsoft Edge</h3>
      <p>Жёстко удаляет Edge: собственный деинсталлятор + Appx, отключает автообновление и блокирует повторную установку через Windows Update.</p>
      <div class="warn">${I.warn}<span>Сначала поставьте другой браузер. Редко после сноса Edge Windows может повторять попытки обновления — это некритично и не ломает систему.</span></div>
      ${btn('edge',st.edge,'Удалить Edge')}
    </div>
    <div class="card">
      <div class="ic">${I.box}</div>${chip(st.store)}<h3>Microsoft Store</h3>
      <p>Удаляет магазин приложений. Установка программ — через вкладку «Программы».</p>
      <div class="warn">${I.warn}<span>Обновление UWP-приложений через магазин перестанет работать.</span></div>
      ${btn('store',st.store,'Удалить Store')}
    </div>
  </div>`;
}

/* -------- ПРОГРАММЫ -------- */
async function renderApps(p){
  const apps = await go('appsList'); STATE.apps=apps;
  const card = a=>`
    <div class="appcard">
      <div class="top">
        <div class="ico" style="background:${a.bg};color:${a.fg}">${a.letter}</div>
        <div><h4>${a.name}</h4><div class="cat">${a.cat}</div></div>
      </div>
      <p>${a.desc}${a.installed&&a.path?`<span class="app-path" title="${a.path}">${a.path}</span>`:''}</p>
      <div class="foot">
        ${a.installed
          ? `<span class="inst-ok">${I.check} Установлено</span>
             <button class="btn ghost sm" onclick="doLaunch('${a.id}')">Открыть</button>`
          : `<button class="btn gold sm" onclick="doInstall('${a.id}')">${I.dl} Установить</button>
             <span class="offline-dot ${a.offline?'':'no'}"><i></i>${a.offline?'установщик скачан':'скачается при установке'}</span>`}
      </div>
    </div>`;
  const n = apps.filter(a=>a.installed).length;
  p.innerHTML = `
  <div class="page-head"><h1>Программы и браузеры</h1>
    <p>KERNEL скачивает официальный установщик, ставит программу тихо и проверяет, что она действительно установилась. Всё скачанное лежит в папке <b>«KERNEL Программы»</b> на рабочем столе — повторная установка пойдёт без интернета. Raven — портативная программа: он просто распакуется туда же и запустится.</p></div>
  <div style="display:flex;gap:10px;align-items:center;margin-bottom:18px">
    <button class="btn ghost" onclick="doDownloadAll()">${I.dl} Скачать всё для офлайна</button>
    <span style="color:var(--dim);font-size:12.5px">Установлено: ${n} из ${apps.length}</span>
  </div>
  <div class="section-label">Приложения</div>
  <div class="apps-grid">${apps.filter(a=>a.cat!=='Браузер').map(card).join('')}</div>
  <div class="section-label">Браузеры</div>
  <div class="apps-grid">${apps.filter(a=>a.cat==='Браузер').map(card).join('')}</div>`;
}

/* -------- ЗАЩИТА -------- */
async function renderShield(p){
  const d = await go('defenderStatus');
  p.innerHTML = `
  <div class="page-head"><h1>Защита системы</h1>
    <p>Статус встроенной защиты Windows. KERNEL не отключает антивирус за вас — но показывает состояние и открывает нужное окно настроек одной кнопкой.</p></div>
  <div class="shield-status">
    <div class="shield-card ${d.realtime?'on':'off'}">
      <div class="ring">${d.realtime?I.check:I.x}</div>
      <div><div class="st">Защита в реальном времени</div><div class="ds">${d.realtime?'включена':'отключена'}</div></div>
    </div>
    <div class="shield-card ${d.tamper?'on':'off'}">
      <div class="ring">${d.tamper?I.check:I.x}</div>
      <div><div class="st">Защита от подделки</div><div class="ds">${d.tamper?'включена':'отключена'}</div></div>
    </div>
  </div>
  <div class="card">
    <div style="display:flex;gap:14px;align-items:flex-start">
      <div style="color:var(--amber);flex-shrink:0;width:22px">${I.warn}</div>
      <div>
        <div style="font-size:15px;font-weight:600;margin-bottom:6px">Выключить защиту можно только вручную</div>
        <p style="color:var(--dim);font-size:13px;line-height:1.6;max-width:640px">Отключение Windows Defender — это ваш осознанный выбор, а не действие «оптимизатора». Нажмите кнопку ниже: откроется окно «Безопасность Windows», где вы сами переключите нужные настройки. Так вы всегда понимаете, что защита выключена именно вами.</p>
        <button class="btn gold" style="margin-top:16px" onclick="doOpenDefender()">${I.shield} Открыть «Безопасность Windows»</button>
      </div>
    </div>
  </div>`;
}

/* -------- ДРАЙВЕРЫ -------- */
async function renderDrivers(p){
  if (STATE.drvScan===undefined) { scanDrivers(); return; } // при первом открытии сразу ищем обновления
  const d = await go('driverList') || [];
  const scan = STATE.drvScan; // 'busy' — идёт поиск, {ok,list,nvidia,msg} — результат
  const nv = scan && scan.nvidia;
  const ageTxt = m => m<0 ? '' : m<1 ? 'свежий' : m<12 ? m+' мес. назад' : Math.floor(m/12)+' г. '+(m%12?m%12+' мес. ':'')+'назад';
  const devRow = x => {
    let st = '';
    if (nv && x.cls==='display' && /nvidia|geforce|rtx|gtx|quadro/i.test(x.name))
      st = nv.update ? `<div class="st upd">доступна ${nv.latest}</div>` : `<div class="st ok">актуально</div>`;
    return `
    <div class="drv-row">
      <div class="di">${x.cls==='display'?I.gpu:I.chip}</div>
      <div style="min-width:0"><div class="dn">${x.name}</div>
        <div class="dv">версия <b>${x.cur}</b>${x.date?' · от '+x.date+' ('+ageTxt(x.age)+')':''}${x.provider?' · '+x.provider:''}</div></div>
      ${st}
    </div>`;
  };
  const list = scan && scan.list || [];
  let found = '';
  if (scan==='busy') found = `<div class="loader"><div class="sp"></div>Проверяю Windows Update и сервер NVIDIA… обычно до минуты</div>`;
  else if (scan && !scan.ok) found = `<div class="warn">${I.warn}<div>${scan.msg||'Поиск не удался.'}</div></div>`;
  else if (scan && !list.length) found = `<div class="drv-row"><div class="di">${I.check}</div><div><div class="dn">Все драйверы актуальны</div><div class="dv">Новых версий нет ни в Windows Update${nv?', ни у NVIDIA':''}.</div></div></div>`;
  else if (scan) found = `
    <div class="uwp-grid" style="grid-template-columns:1fr">${list.map(u=>`
      <div class="uwp-item sel" data-drv="${u.id}" onclick="this.classList.toggle('sel');updDrvCount()">
        <div class="chk"></div>
        <div style="min-width:0"><div class="nm">${u.title}</div>
          <div class="dv" style="font-size:11.5px;color:var(--dim);margin-top:2px">${[u.provider,u.date?'от '+u.date:''].filter(Boolean).join(' · ')}</div></div>
      </div>`).join('')}
    </div>`;
  p.innerHTML = `
  <div class="page-head"><h1>Драйверы</h1>
    <p>KERNEL сам находит и ставит новые драйверы: для всех устройств — из Windows Update (только подписанные Microsoft), для видеокарты NVIDIA — последнюю Game Ready версию напрямую с серверов NVIDIA. Перед установкой проверяется цифровая подпись, настройки драйвера сохраняются.</p></div>
  <div style="display:flex;gap:10px;flex-wrap:wrap;margin-bottom:18px">
    ${list.length?`<button class="btn gold" id="drvInstall" onclick="doUpdateDrivers()">${I.dl} Обновить всё (${list.length})</button>`:''}
    <button class="btn ${list.length?'ghost':'gold'}" onclick="scanDrivers()" ${scan==='busy'?'disabled':''}>${I.chip} Проверить ещё раз</button>
    <button class="btn dark" onclick="doBackupDrivers()">Резервная копия</button>
  </div>
  <div class="section-label">Найденные обновления</div>${found}
  <div class="section-label">Ваши устройства</div>
  ${d.length?d.map(devRow).join(''):'<div class="drv-row"><div class="dn">Устройства не найдены</div></div>'}`;
}
function updDrvCount(){ const n=document.querySelectorAll('[data-drv].sel').length, b=document.getElementById('drvInstall');
  if(b){ b.innerHTML=I.dl+' Обновить выбранное ('+n+')'; b.disabled=!n; } }

/* -------- ЖУРНАЛ -------- */
async function renderLog(p){
  const log = await go('logLines');
  p.innerHTML = `
  <div class="page-head"><h1>Журнал изменений</h1>
    <p>Все действия KERNEL с временем и результатом. Файл: %LOCALAPPDATA%\\KERNEL\\journal.log</p></div>
  <div class="log">${log.map(l=>`
    <div class="ln"><span class="ts">${l.ts}</span><span class="lv ${l.lv}">${l.lv}</span><span class="msg">${l.msg}</span></div>`).join('')}</div>`;
}

/* ================= ДЕЙСТВИЯ ================= */
function bindSwitches(){}
function fmtSize(b){ if(b>1e9)return (b/1073741824).toFixed(1)+' ГБ'; if(b>1e6)return Math.round(b/1048576)+' МБ'; return Math.round(b/1024)+' КБ';}

/* ---- Реальное выполнение: ждём ответ Go и показываем настоящий итог ---- */
let OV_BUSY=false;
function openOverlay(title, sub){
  OV_BUSY=true;
  document.getElementById('p-icon').className='p-icon';
  document.getElementById('p-icon').innerHTML='<div class="sp"></div>';
  document.getElementById('p-title').textContent=title;
  document.getElementById('p-cur').textContent=sub||'Это может занять некоторое время, не закрывайте KERNEL.';
  document.getElementById('p-ok').style.display='none';
  document.getElementById('overlay').classList.add('show');
}
function showResult(r){
  const ok = r && r.ok!==false;
  const ic=document.getElementById('p-icon');
  ic.className='p-icon '+(ok?'ok':'err'); ic.innerHTML= ok?I.check:I.x;
  document.getElementById('p-title').textContent= ok?'Готово':'Не получилось';
  document.getElementById('p-cur').textContent=(r&&r.msg)||(ok?'Изменения применены. Подробности — во вкладке «Журнал».':'Подробности — во вкладке «Журнал».');
  document.getElementById('p-ok').style.display='inline-flex';
  OV_BUSY=false;
}
function showReboot(){ document.getElementById('rebootBar').classList.add('show'); }
function hideReboot(){ document.getElementById('rebootBar').classList.remove('show'); }
async function doReboot(){ await go('reboot'); openOverlay('Перезагрузка через 20 секунд','Сохраните открытые файлы. Отменить: Win+R → shutdown /a'); document.getElementById('p-icon').className='p-icon ok'; document.getElementById('p-icon').innerHTML=I.check; document.getElementById('p-ok').style.display='inline-flex'; OV_BUSY=false; }
function closeOverlay(){ if(OV_BUSY) return; document.getElementById('overlay').classList.remove('show'); }
async function act(title, sub, fn, needReboot){
  if(OV_BUSY) return;
  openOverlay(title, sub);
  let r;
  try{ r = await fn(); }catch(e){ r={ok:false,msg:String(e)}; }
  if(r===true||r===undefined||r===null) r={ok:true};
  if(r===false) r={ok:false};
  showResult(r);
  if((needReboot || (r&&r.reboot)) && (r.ok!==false)){ showReboot(); if(r.msg) document.getElementById('p-cur').textContent=r.msg+' Изменения вступят в силу после перезагрузки.'; }
  renderPage();
}

async function runBoostAll(){ await act('Настраиваю систему…','Создаю точку восстановления и применяю постоянные настройки. Обычно 1–2 минуты.',
  async()=>{ await go('applyAll'); return {ok:true,msg:'Система настроена. Все изменения постоянные и записаны в Журнал.'}; }, true); }
async function applySelectedTweaks(){ const ids=[...document.querySelectorAll('[data-tw]:checked')].map(e=>e.dataset.tw);
  if(!ids.length) return;
  if(ids.includes('vbs') && !(STATE.tweaks.find(t=>t.id==='vbs')||{}).applied){
    const ok = await confirmDialog('Отключить VBS?',
      'Это снизит защиту Windows от вредоносных драйверов и некоторых атак. Перестанут работать WSL2, Hyper-V, Windows Sandbox и Android-эмуляторы на базе Hyper-V. Нужна перезагрузка. Откат — кнопкой «Вернуть как было».',
      'Отключить VBS');
    if(!ok) return;
  }
  const rb = ids.some(i=>(STATE.tweaks.find(t=>t.id===i)||{}).reboot);
  await act('Применяю настройки…', ids.length+' шт.', async()=>{ await go('applyTweaks',ids); return {ok:true,msg:'Настройки применены и сохранятся навсегда.'}; }, rb); }
async function revertTweaks(){ await act('Откатываю изменения…','', async()=>{ await go('revertAll'); return {ok:true,msg:'Все настройки KERNEL возвращены к стандартным.'}; }); }
async function runClean(){ const ids=[...document.querySelectorAll('[data-cl]:checked')].map(e=>e.dataset.cl);
  if(!ids.length) return;
  await act('Очищаю…','Удаляю выбранные временные файлы.', async()=>{ await go('clean',ids); return {ok:true,msg:'Очистка завершена. Сколько освобождено — в Журнале.'}; }); }
async function doRemoveUwp(){ const ids=[...document.querySelectorAll('.uwp-item.sel')].map(e=>e.dataset.pkg);
  if(!ids.length) return;
  await act('Удаляю приложения…', 'Выбрано: '+ids.length+'. Каждое удаление проверяется.', ()=>go('removeUwp',ids), true); }
const COMP_NAMES={edge:'Microsoft Edge',store:'Microsoft Store'};
async function doRemoveComp(k){ await act('Удаляю '+COMP_NAMES[k]+'…','Запускаю штатный деинсталлятор и проверяю результат.', ()=>go('removeComp',k), true); }
async function doInstall(id){ const a=(STATE.apps||[]).find(x=>x.id===id);
  await act('Устанавливаю '+(a?a.name:id)+'…','Скачиваю официальный установщик на рабочий стол и ставлю программу. Это может занять пару минут.', ()=>go('install',id)); }
async function doDownloadAll(){ await act('Скачиваю установщики…','Сохраняю всё в папку «KERNEL Программы» на рабочем столе.', ()=>go('downloadAll')); }
async function doOpenDefender(){ await go('openDefender'); }
async function scanDrivers(){
  STATE.drvScan='busy'; renderPage();
  const r = await go('driverScan');
  STATE.drvScan = r || {ok:false};
  if (STATE.tab==='drivers') renderPage();
}
async function doUpdateDrivers(){ const ids=[...document.querySelectorAll('[data-drv].sel')].map(e=>e.dataset.drv);
  if(!ids.length) return;
  await act('Обновляю драйверы…','Выбрано: '+ids.length+'. Скачиваю и ставлю'+(ids.some(i=>i.startsWith('nvidia:'))?' (драйвер NVIDIA весит около 1 ГБ)':'')+' — может занять 5–15 минут. Экран может мигнуть.', ()=>go('updateDrivers',ids).then(r=>{ STATE.drvScan=undefined; return r; })); }
async function doBackupDrivers(){ await act('Сохраняю драйверы…','', async()=>{ await go('backupDrivers'); return {ok:true,msg:'Копия драйверов сохранена в %LOCALAPPDATA%\\KERNEL\\DriverBackup'}; }); }
function doToggleAutoClean(){ go('toggleAutoClean'); }
async function doLaunch(id){ const r=await go('launchApp',id); if(r&&r.ok===false){ openOverlay('',''); showResult(r);} }

async function doFreshAll(){ const ids=(STATE.freshIds||[]); await act('Настраиваю Windows…','Чищу Пуск, рабочий стол и панель задач. Проводник перезапустится.', async()=>{ const list=await go('freshList'); return go('freshApply', list.map(f=>f.id)); }); }
async function doFreshSelected(){ const ids=[...document.querySelectorAll('[data-fr]:checked')].map(e=>e.dataset.fr); if(!ids.length) return; await act('Применяю настройки…', ids.length+' шт.', ()=>go('freshApply',ids)); }
async function doWallpaper(){ await act('Ставлю обои…','', ()=>go('setWallpaper','')); }

/* ---- Проверка обновлений ---- */
let UPDATE_URL='';
async function checkUpdates(){
  try{
    const v=await go('appVersion'); const e=document.getElementById('ver'); if(e&&v) e.textContent='v'+v;
    const r=await go('checkUpdate');
    if(r && r.available){
      UPDATE_URL=r.url||'';
      document.getElementById('ub-title').textContent='Вышло обновление KERNEL '+r.latest;
      document.getElementById('ub-sub').textContent=(r.notes?r.notes+' · ':'')+'У вас '+r.current+'. Нажмите «Скачать», чтобы обновиться.';
      document.getElementById('updateBar').classList.add('show');
    }
  }catch(e){}
}
function hideUpdate(){ document.getElementById('updateBar').classList.remove('show'); }
function doUpdate(){ if(UPDATE_URL) go('openUrl', UPDATE_URL); }

function openTg(){ go('openUrl','https://t.me/kernel1lab'); }

renderTabs(); renderPage(); checkUpdates();
