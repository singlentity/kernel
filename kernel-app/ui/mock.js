/* Демо-данные только для превью дизайна (в .exe их заменяет Go) */
const MOCK = {
  sysInfo:()=>({score:65, cpu:'AMD Ryzen 7 9800X3D 8-Core', gpu:'AMD Radeon(TM) Graphics + NVIDIA GeForce RTX 5070', ram:'32 ГБ', os:'Windows 11 Pro', build:'26H2'}),
  tweaks:()=>[
    {id:'power',group:'Производительность и FPS',name:'Схема питания «Максимальная производительность»',desc:'Процессор перестаёт сбрасывать частоты в простое — стабильнее FPS. Остаётся после перезагрузки.',on:true,applied:false,reboot:false},
    {id:'gamedvr',group:'Производительность и FPS',name:'Отключить запись игр (Game DVR)',desc:'Фоновая запись Xbox отнимает 5–10% FPS даже когда вы не записываете. Отключаем навсегда.',on:true,applied:false,reboot:false},
    {id:'hags',group:'Производительность и FPS',name:'Аппаратное ускорение планирования GPU (HAGS)',desc:'⚠ Спорная настройка: на одних видеокартах снижает задержку, на других вызывает фризы и просадку FPS. По умолчанию ВЫКЛ — включайте вручную и проверяйте в своей игре.',on:false,applied:false,reboot:true},
    {id:'gamemode',group:'Производительность и FPS',name:'Игровой режим Windows',desc:'Включает штатный «Игровой режим» — Windows сама отдаёт приоритет активной игре. Безопасно и поддерживается Microsoft.',on:true,applied:false,reboot:false},
    {id:'bgapps',group:'Фоновые процессы',name:'Запретить работу приложений в фоне',desc:'UWP-приложения перестают работать и тратить ресурсы, когда вы ими не пользуетесь.',on:true,applied:false,reboot:false},
    {id:'telemetry',group:'Фоновые процессы',name:'Отключить телеметрию и сбор данных',desc:'Останавливает службу диагностики Windows, которая постоянно шлёт данные.',on:true,applied:false,reboot:false},
    {id:'sysmain',group:'Фоновые процессы',name:'Отключить SysMain (Superfetch)',desc:'На SSD эта служба бесполезна и только грузит диск. На HDD лучше оставить.',on:false,applied:false,reboot:false},
    {id:'mouseaccel',group:'Производительность и FPS',name:'Отключить акселерацию мыши',desc:'Убирает «улучшение точности указателя». Прицел в играх становится предсказуемым — движение мыши 1:1.',on:true,applied:false,reboot:false},
    {id:'nagle',group:'Производительность и FPS',name:'Снизить задержку в онлайн-играх (отключить Nagle)',desc:'Убирает объединение мелких сетевых пакетов — ниже пинг и меньше рывков. Нужна перезагрузка.',on:false,applied:false,reboot:true},
    {id:'vbs',group:'Производительность и FPS',name:'Отключить VBS (виртуализация безопасности)',desc:'Снижает задержку памяти и может дать прирост FPS в играх.',on:false,applied:false,reboot:true,vbs:true,details:'VBS (Virtualization-Based Security) — функция защиты Windows, которая запускает часть ядра системы в отдельной виртуальной машине. Она защищает от вредоносных драйверов и кражи паролей, но постоянно работающий гипервизор добавляет задержку при работе с памятью и процессором.\n\nЧто даст отключение:\n• ниже задержка памяти (обычно на 3–7 нс в AIDA64);\n• прирост FPS в части игр, сильнее всего — в 1% low (меньше просадок);\n• чуть меньше нагрузки на систему в фоне.\n\nЧем вы жертвуете:\n• защитой ядра от вредоносных драйверов и некоторых видов атак;\n• перестанут работать WSL2, Hyper-V, Windows Sandbox и Android-эмуляторы на базе Hyper-V.\n\nИзменения вступают в силу после перезагрузки и полностью откатываются кнопкой «Вернуть как было».'},
    {id:'reserved',group:'Фоновые процессы',name:'Освободить ~7 ГБ «зарезервированного хранилища»',desc:'Windows держит 7 ГБ про запас под обновления. Отключаем — место возвращается. Нужна перезагрузка.',on:false,applied:false,reboot:true},
    {id:'tips',group:'Интерфейс',name:'Убрать рекламу и «советы» Windows',desc:'Отключает предложения в меню Пуск, подсказки и рекламные уведомления.',on:true,applied:true,reboot:false},
  ],
  cleanScan:()=>[
    {id:'temp',name:'Временные файлы',desc:'Папки TEMP пользователя и Windows.',bytes:2843000000,on:true},
    {id:'update',name:'Кэш Windows Update',desc:'Скачанные установщики обновлений, которые уже применены.',bytes:1200000000,on:true},
    {id:'thumbs',name:'Кэш эскизов',desc:'Миниатюры изображений — пересоздадутся сами.',bytes:340000000,on:true},
    {id:'recycle',name:'Корзина',desc:'Удалённые файлы на всех дисках.',bytes:5100000000,on:false},
    {id:'shadercache',name:'Кэш шейдеров видеокарты',desc:'Кэш DirectX/NVIDIA/AMD. Иногда убирает фризы компиляции шейдеров.',bytes:890000000,on:true},
    {id:'crashdumps',name:'Отчёты об ошибках и дампы',desc:'Дампы памяти после сбоев — весят много, не нужны.',bytes:620000000,on:true},
    {id:'logs',name:'Журналы системы',desc:'Старые лог-файлы Windows.',bytes:210000000,on:true},
  ],
  uwpList:()=>{
    const j=['Копилот','Новости','Погода','Bing Поиск','Clipchamp','Xbox','Xbox Game Bar','Solitaire Collection','Советы','Люди','Your Phone','Groove Музыка','Кино и ТВ','Office Hub','Teams','To Do','Power Automate','Начало работы','Обратная связь','Карты','OneNote','Outlook (новый)','Dev Home','Быстрая поддержка','3D Viewer','Paint 3D','Mixed Reality'];
    const keep=['Калькулятор','Камера','Фотографии','Ножницы','Запись голоса','Будильники и часы','Терминал','Блокнот','Магазин','Диспетчер задач'];
    const spec=['Значок «Сведения об изображении» (Spotlight)','Виджеты (погода/новости на панели)'];
    const not=['Portal виртуальной реальности','Skype','Spotify'];
    const mk=(n,inst,junk)=>({id:n,name:n,installed:inst,junk:junk,sel:junk&&inst});
    return [...j.map(n=>mk(n,true,true)), ...spec.map(n=>mk(n,true,true)), ...keep.map(n=>mk(n,true,false)), ...not.map(n=>mk(n,false,true))];
  },
  appsList:()=>[
    {id:'npp',name:'Notepad++',cat:'Редактор',letter:'N+',bg:'#1E3A24',fg:'#7CD67C',desc:'Быстрый текстовый редактор с подсветкой кода. Замена Блокноту.',offline:true,installed:true,path:'C:\\Program Files\\Notepad++\\notepad++.exe'},
    {id:'mpc',name:'MPC-HC',cat:'Видео',letter:'▶',bg:'#1E2A3A',fg:'#7AA7D9',desc:'Лёгкий видеоплеер, играет почти всё. Замена «Кино и ТВ».',offline:true},
    {id:'imageglass',name:'ImageGlass 8.12',cat:'Фото',letter:'◆',bg:'#3A2E1A',fg:'#E8C15A',desc:'Быстрый просмотр изображений. Замена «Фотографиям».',offline:true},
    {id:'raven',name:'Raven',cat:'Утилита',letter:'✦',bg:'#2A1E3A',fg:'#B98AD9',desc:'Установка приложений из Microsoft Store без самого Store.',offline:false},
    {id:'firefox',installed:true,path:'C:\\Program Files\\Mozilla Firefox\\firefox.exe',name:'Mozilla Firefox',cat:'Браузер',letter:'F',bg:'#3A2410',fg:'#E8A15A',desc:'Независимый браузер на своём движке.',offline:true},
    {id:'chrome',name:'Google Chrome',cat:'Браузер',letter:'◉',bg:'#2A2A2E',fg:'#E8E8EE',desc:'Самый популярный браузер на Chromium.',offline:true},
    {id:'brave',name:'Brave',cat:'Браузер',letter:'B',bg:'#3A1E10',fg:'#E87A4A',desc:'Chromium со встроенной блокировкой рекламы.',offline:false},
    {id:'yandex',name:'Yandex Browser',cat:'Браузер',letter:'Я',bg:'#3A1010',fg:'#E85C5C',desc:'Chromium-браузер от Яндекса.',offline:false},
  ],
  vbsStatus:()=>({status:2,text:'работает'}),
  vbsRevert:()=>({ok:true,msg:'VBS возвращён к исходному состоянию.'}),
  appVersion:()=>'1.5.0',
  checkUpdate:()=>({available:true,current:'1.5.0',latest:'1.6.0',url:'https://t.me/kernel1lab',notes:'Новый анализ баланса системы'}),
  balance:()=>({cpu:'AMD Ryzen 7 9800X3D 8-Core',gpu:'AMD Radeon(TM) Graphics + NVIDIA GeForce RTX 5070',cpuTier:100,gpuTier:72,kind:'gpu',known:true,verdict:'Видеокарта — главное ограничение. Процессор с большим запасом: упор идёт в GPU почти при любом качестве, а на высоком разрешении (4K, DLAA) — тем более. CPU раскрывается только в киберспорте на низких настройках ради сотен FPS.'}),
  freshList:()=>[
    {id:'startclean',name:'Чистое меню Пуск',desc:'Убирает из Пуска рекомендации, рекламу, «недавно добавленные» и советы Microsoft. Остаётся только то, что закрепили вы.',on:true,applied:true},
    {id:'cleandesktop',name:'Чистый рабочий стол',desc:'Убирает с рабочего стола все системные значки Windows (Корзина, Этот компьютер, Сеть, папка пользователя, «Сведения об изображении»). Ваши файлы не трогаются.',on:true,applied:true},
    {id:'recyclebin',name:'Убрать корзину с рабочего стола',desc:'Прячет значок корзины с рабочего стола, чтобы не мозолил глаз (будет находиться в меню Пуск). Сама корзина продолжает работать.',on:false},
    {id:'fileext',name:'Показывать расширения файлов',desc:'Включает отображение .exe, .txt, .jpg в именах файлов. Первое, что делает любой опытный пользователь.',on:true},
    {id:'classicmenu',name:'Классическое контекстное меню',desc:'Возвращает полное меню по правому клику из Windows 10 — без «Показать дополнительные параметры».',on:true},
    {id:'darkmode',name:'Тёмная тема Windows',desc:'Переключает систему и приложения в тёмный режим.',on:true},
    {id:'hidewidgets',name:'Убрать виджеты с панели задач',desc:'Прячет кнопку погоды/новостей слева на панели задач.',on:true},
    {id:'hidesearch',name:'Скрыть поиск с панели задач',desc:'Полностью убирает строку поиска с панели задач (режим «Скрыть»). Поиск по-прежнему работает по кнопке Пуск или Win+S.',on:true},
  ],
  freshApply:()=>({ok:true,msg:'Готово. Состояние сохранено.'}),
  setWallpaper:()=>({ok:true,msg:'Обои установлены.'}),
  pinInfo:()=>({blocked:true,msg:'Windows 11 хранит закреплённые приложения в закрытом файле start2.bin и не даёт менять их скриптом. Проще закрепить вручную: правый клик по приложению → «Закрепить на начальном экране».'}),
  defenderStatus:()=>({realtime:true, tamper:true}),
  driverList:()=>[
    {name:'NVIDIA GeForce RTX 5070',cur:'32.0.16.2101',date:'2026-10-19',age:0,provider:'NVIDIA',cls:'display',vendor:'https://www.nvidia.com/ru-ru/software/nvidia-app/'},
    {name:'AMD SMBUS',cur:'2.0.0.29',date:'2025-08-07',age:14,provider:'Advanced Micro Devices, Inc',cls:'system',vendor:'https://www.amd.com/ru/support/download/drivers.html'},
    {name:'Realtek Gaming 2.5GbE Family Controller',cur:'1125.31.50.603',date:'2026-03-06',age:7,provider:'Realtek',cls:'net',vendor:''},
    {name:'Realtek High Definition Audio',cur:'6.0.9982.1',date:'2026-04-27',age:6,provider:'Realtek Semiconductor Corp.',cls:'media',vendor:''},
  ],
  driverScan:()=>new Promise(r=>setTimeout(()=>r({ok:true,list:[
    {id:'a1',title:'Realtek - Net - 10.72.0.0',provider:'Realtek',date:'2026-09-12'},
    {id:'b2',title:'Realtek Semiconductor Corp. - MEDIA - 6.0.9990.1',provider:'Realtek Semiconductor Corp.',date:'2026-08-30'},
  ]}),1500)),
  procCount:()=>214,
  logLines:()=>[
    {ts:'20:11:04',lv:'INFO',msg:'KERNEL запущен · Windows 11 Pro 23H2 · права администратора'},
    {ts:'20:11:22',lv:'INFO',msg:'Создана точка восстановления «KERNEL»'},
    {ts:'20:11:23',lv:'OK',msg:'Схема питания «Максимальная производительность» активирована'},
    {ts:'20:11:23',lv:'OK',msg:'Game DVR отключён (реестр, постоянно)'},
    {ts:'20:11:24',lv:'OK',msg:'Служба телеметрии DiagTrack остановлена и отключена'},
    {ts:'20:11:25',lv:'WARN',msg:'Пакет Xbox не удалён: защищён системой, помечен для ручного удаления'},
    {ts:'20:11:26',lv:'OK',msg:'Очищено 2.8 ГБ временных файлов'},
    {ts:'20:11:26',lv:'ERR',msg:'Нет доступа к C:\\Windows\\Temp\\locked.tmp — пропущено'},
  ],
  applyAll:()=>true, applyTweaks:()=>true, revertAll:()=>true, clean:()=>true,
  removeUwp:()=>({ok:true,msg:'Удалено приложений: 5'}), removeComp:()=>({ok:false,msg:'Microsoft Edge: штатный деинсталлятор отказался удалять'}), install:(id)=>({ok:true,msg:'Notepad++ установлен'}), downloadAll:()=>({ok:true,msg:'Все установщики (8) сохранены'}),
  edgeStatus:()=>({edge:true,store:false,webview:true}), openUrl:()=>true, launchApp:()=>({ok:true}),
  openDefender:()=>true, updateDrivers:()=>true, backupDrivers:()=>true, toggleAutoClean:()=>true,
};
