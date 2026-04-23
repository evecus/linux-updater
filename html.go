package main

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Linux 更新管理器</title>
<style>
  *{box-sizing:border-box;margin:0;padding:0}
  :root{
    --bg:#0f1117;--card:#1a1d2e;--card2:#22263a;--border:#2e3250;
    --accent:#6c63ff;--accent2:#4ecca3;--danger:#ff4757;--warn:#ffa502;
    --text:#e2e8f0;--text2:#8892a4;--green:#2ed573;--shadow:0 4px 24px rgba(0,0,0,.4);
  }
  body{background:var(--bg);color:var(--text);font-family:'Segoe UI',system-ui,sans-serif;min-height:100vh}
  header{background:var(--card);border-bottom:1px solid var(--border);padding:16px 24px;
    display:flex;align-items:center;justify-content:space-between;position:sticky;top:0;z-index:100}
  header h1{font-size:1.25rem;font-weight:700;display:flex;align-items:center;gap:10px}
  header h1 span.logo{font-size:1.5rem}
  .subtitle{color:var(--text2);font-size:.8rem;margin-top:2px}
  .btn{display:inline-flex;align-items:center;gap:6px;padding:8px 16px;border:none;border-radius:8px;
    cursor:pointer;font-size:.875rem;font-weight:500;transition:all .2s;white-space:nowrap}
  .btn-primary{background:var(--accent);color:#fff}
  .btn-primary:hover{background:#5a52e0;transform:translateY(-1px)}
  .btn-success{background:var(--accent2);color:#0f1117}
  .btn-success:hover{background:#3dd9b3}
  .btn-danger{background:var(--danger);color:#fff}
  .btn-danger:hover{background:#e83a48}
  .btn-ghost{background:transparent;color:var(--text2);border:1px solid var(--border)}
  .btn-ghost:hover{background:var(--card2);color:var(--text)}
  .btn-warn{background:var(--warn);color:#0f1117}
  .btn-warn:hover{background:#e6940f}
  .btn-sm{padding:5px 10px;font-size:.8rem;border-radius:6px}
  .btn:disabled{opacity:.5;cursor:not-allowed;transform:none!important}
  main{padding:24px;max-width:1400px;margin:0 auto}
  .grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(360px,1fr));gap:20px}
  .card{background:var(--card);border:1px solid var(--border);border-radius:14px;
    padding:20px;transition:box-shadow .2s;position:relative;overflow:hidden}
  .card:hover{box-shadow:var(--shadow)}
  .card-top-bar{height:3px;border-radius:3px 3px 0 0;position:absolute;top:0;left:0;right:0}
  .card-header{display:flex;align-items:flex-start;justify-content:space-between;margin-bottom:14px}
  .card-title{font-size:1rem;font-weight:600;display:flex;align-items:center;gap:8px}
  .type-badge{font-size:.7rem;padding:2px 8px;border-radius:20px;font-weight:600;letter-spacing:.5px}
  .badge-core{background:rgba(108,99,255,.2);color:#a39bff}
  .badge-file{background:rgba(78,204,163,.2);color:#4ecca3}
  .card-body{display:grid;gap:8px;margin-bottom:16px}
  .info-row{display:flex;align-items:center;gap:8px;font-size:.83rem}
  .info-label{color:var(--text2);min-width:70px;flex-shrink:0}
  .info-val{color:var(--text);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;flex:1}
  .version-tag{background:rgba(78,204,163,.15);color:var(--accent2);padding:2px 8px;
    border-radius:6px;font-family:monospace;font-size:.82rem}
  .status-dot{width:8px;height:8px;border-radius:50%;flex-shrink:0}
  .dot-idle{background:var(--text2)}
  .dot-ok{background:var(--green)}
  .dot-error{background:var(--danger)}
  .dot-checking,.dot-updating{background:var(--warn);animation:pulse 1s infinite}
  .dot-update_available{background:#ff6b35;animation:pulse .8s infinite}
  @keyframes pulse{0%,100%{opacity:1}50%{opacity:.4}}
  .card-actions{display:flex;gap:8px;flex-wrap:wrap}
  .empty-state{text-align:center;padding:80px 20px;color:var(--text2)}
  .empty-state .icon{font-size:4rem;margin-bottom:16px}
  .empty-state p{font-size:1rem;margin-bottom:8px;color:var(--text)}
  .empty-state small{font-size:.85rem}

  /* Modal */
  .modal-bg{position:fixed;inset:0;background:rgba(0,0,0,.7);backdrop-filter:blur(4px);
    z-index:200;display:flex;align-items:center;justify-content:center;padding:20px}
  .modal-bg.hidden{display:none}
  .modal{background:var(--card);border:1px solid var(--border);border-radius:16px;
    width:100%;max-width:640px;max-height:90vh;overflow-y:auto;box-shadow:var(--shadow)}
  .modal-header{padding:20px 24px 0;display:flex;align-items:center;justify-content:space-between}
  .modal-header h2{font-size:1.1rem;font-weight:600}
  .modal-close{background:none;border:none;color:var(--text2);font-size:1.4rem;cursor:pointer;
    width:32px;height:32px;display:flex;align-items:center;justify-content:center;border-radius:8px}
  .modal-close:hover{background:var(--card2);color:var(--text)}
  .modal-body{padding:20px 24px}
  .modal-footer{padding:0 24px 20px;display:flex;gap:10px;justify-content:flex-end}
  .form-group{margin-bottom:16px}
  .form-row{display:grid;grid-template-columns:1fr 1fr;gap:12px}
  label{display:block;font-size:.83rem;color:var(--text2);margin-bottom:6px;font-weight:500}
  label .req{color:var(--danger)}
  input,select,textarea{width:100%;background:var(--card2);border:1px solid var(--border);
    border-radius:8px;padding:9px 12px;color:var(--text);font-size:.875rem;outline:none;
    transition:border-color .2s;font-family:inherit}
  input:focus,select:focus,textarea:focus{border-color:var(--accent)}
  input::placeholder,textarea::placeholder{color:var(--text2)}
  textarea{resize:vertical;min-height:60px}
  select option{background:var(--card2)}
  .hint{font-size:.75rem;color:var(--text2);margin-top:4px}
  .divider{border:none;border-top:1px solid var(--border);margin:16px 0}

  /* Log modal */
  .log-box{background:#0a0c16;border:1px solid var(--border);border-radius:8px;
    padding:14px;font-family:'Courier New',monospace;font-size:.78rem;
    max-height:500px;overflow-y:auto;white-space:pre-wrap;word-break:break-all;
    color:#c8d3f0;line-height:1.6}
  .log-box .log-sep{color:#3a4060}
  .log-box .log-ok{color:var(--green)}
  .log-box .log-err{color:var(--danger)}
  .log-box .log-info{color:var(--accent2)}
  .log-box .log-warn{color:var(--warn)}

  .toast-container{position:fixed;top:20px;right:20px;z-index:999;display:flex;flex-direction:column;gap:8px}
  .toast{background:var(--card);border:1px solid var(--border);border-radius:10px;
    padding:12px 16px;font-size:.85rem;box-shadow:var(--shadow);
    display:flex;align-items:center;gap:8px;animation:slideIn .3s ease;max-width:320px}
  @keyframes slideIn{from{transform:translateX(100%);opacity:0}to{transform:translateX(0);opacity:1}}
  .toast.success{border-color:var(--green)}
  .toast.error{border-color:var(--danger)}
  .toast.info{border-color:var(--accent)}

  .repo-link{color:var(--accent2);text-decoration:none;font-size:.8rem}
  .repo-link:hover{text-decoration:underline}
  .cron-tag{font-size:.75rem;background:rgba(108,99,255,.15);color:#a39bff;
    padding:1px 7px;border-radius:4px;font-family:monospace}
  .last-time{font-size:.75rem;color:var(--text2)}
</style>
</head>
<body>
<header>
  <div>
    <h1><span class="logo">🔄</span> Linux 更新管理器</h1>
    <div class="subtitle">自动监控 GitHub Release，保持程序最新</div>
  </div>
  <button class="btn btn-primary" onclick="openModal()">＋ 添加任务</button>
</header>

<main>
  <div class="grid" id="taskGrid"></div>
  <div id="emptyState" class="empty-state hidden">
    <div class="icon">📦</div>
    <p>暂无更新任务</p>
    <small>点击右上角「添加任务」开始配置</small>
  </div>
</main>

<!-- Add/Edit Modal -->
<div class="modal-bg hidden" id="modalBg">
  <div class="modal">
    <div class="modal-header">
      <h2 id="modalTitle">添加任务</h2>
      <button class="modal-close" onclick="closeModal()">✕</button>
    </div>
    <div class="modal-body">
      <div class="form-row">
        <div class="form-group">
          <label>任务名称 <span class="req">*</span></label>
          <input id="fName" placeholder="例如：sing-box">
        </div>
        <div class="form-group">
          <label>更新类型 <span class="req">*</span></label>
          <select id="fType">
            <option value="core">核心（可执行文件）</option>
            <option value="file">文件（原样替换）</option>
          </select>
        </div>
      </div>

      <div class="form-group">
        <label>GitHub 项目地址 <span class="req">*</span></label>
        <input id="fRepo" placeholder="https://github.com/owner/repo">
      </div>

      <div class="form-row">
        <div class="form-group">
          <label>当前版本</label>
          <input id="fVersion" placeholder="留空则立即执行一次更新">
          <div class="hint">例如：v1.2.3</div>
        </div>
        <div class="form-group">
          <label>下载文件关键词 <span class="req">*</span></label>
          <input id="fKeyword" placeholder="例如：linux amd64">
          <div class="hint">空格分隔多个关键词，模糊匹配评分</div>
        </div>
      </div>

      <div class="form-row">
        <div class="form-group">
          <label>文件/核心重命名</label>
          <input id="fRename" placeholder="留空则保持原文件名">
        </div>
        <div class="form-group">
          <label>文件/核心放置路径 <span class="req">*</span></label>
          <input id="fTarget" placeholder="/usr/local/bin">
          <div class="hint">绝对路径，目录不存在会自动创建</div>
        </div>
      </div>

      <hr class="divider">

      <div class="form-group">
        <label>更新前执行</label>
        <textarea id="fPre" rows="2" placeholder="例如：systemctl stop myservice"></textarea>
      </div>
      <div class="form-group">
        <label>更新后执行</label>
        <textarea id="fPost" rows="2" placeholder="例如：systemctl start myservice"></textarea>
      </div>
      <div class="form-group">
        <label>定时任务（Cron）</label>
        <input id="fCron" placeholder="例如：0 2 * * *  (留空则仅手动触发)">
        <div class="hint">标准 5 段 cron 表达式。留空则只能在面板手动触发更新。</div>
      </div>
    </div>
    <div class="modal-footer">
      <button class="btn btn-ghost" onclick="closeModal()">取消</button>
      <button class="btn btn-primary" onclick="saveTask()">💾 保存</button>
    </div>
  </div>
</div>

<!-- Log Modal -->
<div class="modal-bg hidden" id="logModalBg">
  <div class="modal" style="max-width:760px">
    <div class="modal-header">
      <h2 id="logModalTitle">运行日志</h2>
      <button class="modal-close" onclick="closeLogModal()">✕</button>
    </div>
    <div class="modal-body">
      <div class="log-box" id="logBox">（暂无日志）</div>
    </div>
    <div class="modal-footer">
      <button class="btn btn-ghost btn-sm" onclick="refreshLog()">🔃 刷新</button>
      <button class="btn btn-ghost" onclick="closeLogModal()">关闭</button>
    </div>
  </div>
</div>

<div class="toast-container" id="toasts"></div>

<script>
const $ = id => document.getElementById(id);

// ---- Toast ----
function toast(msg, type='info', dur=3000){
  const el = document.createElement('div');
  el.className = 'toast ' + type;
  const icon = {info:'ℹ️', success:'✅', error:'❌'}[type]||'ℹ️';
  el.innerHTML = '<span>'+icon+'</span><span>'+msg+'</span>';
  $('toasts').appendChild(el);
  setTimeout(()=>el.remove(), dur);
}

// ---- State ----
let tasks = [];
let editID = null;
let logTaskID = null;

// ---- API ----
async function api(method, path, body){
  const opts = {method, headers:{'Content-Type':'application/json'}};
  if(body) opts.body = JSON.stringify(body);
  const r = await fetch(path, opts);
  const data = await r.json();
  if(!r.ok) throw new Error(data.error||'请求失败');
  return data;
}

// ---- Load tasks ----
async function loadTasks(){
  try{
    tasks = await api('GET','/api/tasks');
    renderTasks();
  }catch(e){toast(e.message,'error')}
}

function statusLabel(s){
  const map={
    idle:'空闲',ok:'正常',error:'错误',
    checking:'检测中…',updating:'更新中…',
    update_available:'有新版本'
  };
  return map[s]||s;
}

function timeFmt(ts){
  if(!ts||ts==='0001-01-01T00:00:00Z') return '—';
  const d = new Date(ts);
  const now = new Date();
  const diff = Math.floor((now-d)/1000);
  if(diff<60) return diff+'秒前';
  if(diff<3600) return Math.floor(diff/60)+'分钟前';
  if(diff<86400) return Math.floor(diff/3600)+'小时前';
  return d.toLocaleDateString('zh-CN');
}

function repoShort(url){
  return url.replace('https://github.com/','').replace('http://github.com/','');
}

function renderTasks(){
  const grid = $('taskGrid');
  const empty = $('emptyState');
  if(!tasks.length){
    grid.innerHTML='';
    empty.classList.remove('hidden');
    return;
  }
  empty.classList.add('hidden');
  grid.innerHTML = tasks.map(t=>{
    const isCore = t.update_type==='core';
    const barColor = isCore?'#6c63ff':'#4ecca3';
    const dotCls = 'dot-'+(t.status||'idle');
    return \`
    <div class="card" id="card-\${t.id}">
      <div class="card-top-bar" style="background:\${barColor}"></div>
      <div class="card-header">
        <div>
          <div class="card-title">
            \${t.name}
            <span class="type-badge \${isCore?'badge-core':'badge-file'}">\${isCore?'CORE':'FILE'}</span>
          </div>
          <a class="repo-link" href="\${t.repo_url}" target="_blank">↗ \${repoShort(t.repo_url)}</a>
        </div>
        <button class="btn btn-ghost btn-sm" onclick="openEdit('\${t.id}')" title="编辑">✏️</button>
      </div>
      <div class="card-body">
        <div class="info-row">
          <span class="info-label">版本</span>
          <span class="version-tag">\${t.current_version||'未知'}</span>
        </div>
        <div class="info-row">
          <span class="info-label">目标路径</span>
          <span class="info-val" title="\${t.target_path}">\${t.target_path}</span>
        </div>
        <div class="info-row">
          <span class="info-label">关键词</span>
          <span class="info-val">\${t.file_keyword}</span>
        </div>
        \${t.cron?'<div class="info-row"><span class="info-label">定时</span><span class="cron-tag">'+t.cron+'</span></div>':''}
        <div class="info-row">
          <div class="status-dot \${dotCls}"></div>
          <span class="info-val">\${statusLabel(t.status||'idle')}
          \${t.last_error?'<span style="color:var(--danger);font-size:.75rem"> — '+escHtml(t.last_error.substring(0,60))+'</span>':''}</span>
        </div>
        <div class="info-row">
          <span class="info-label">上次检测</span>
          <span class="last-time">\${timeFmt(t.last_check)}</span>
          \${t.last_update&&t.last_update!=='0001-01-01T00:00:00Z'?'<span class="info-label" style="margin-left:8px">更新</span><span class="last-time">'+timeFmt(t.last_update)+'</span>':''}
        </div>
      </div>
      <div class="card-actions">
        <button class="btn btn-ghost btn-sm" onclick="checkTask('\${t.id}')" id="btnCheck-\${t.id}">🔍 检测</button>
        <button class="btn btn-success btn-sm" onclick="updateTask('\${t.id}')" id="btnUpdate-\${t.id}">⬆ 更新</button>
        <button class="btn btn-ghost btn-sm" onclick="openLog('\${t.id}','\${escHtml(t.name)}')">📋 日志</button>
        <button class="btn btn-danger btn-sm" onclick="deleteTask('\${t.id}','\${escHtml(t.name)}')">🗑</button>
      </div>
    </div>\`;
  }).join('');
}

function escHtml(s){
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

// ---- Actions ----
async function checkTask(id){
  const btn = $('btnCheck-'+id);
  if(btn){btn.disabled=true;btn.textContent='检测中…'}
  try{
    await api('POST','/api/check/'+id);
    toast('开始检测，稍后刷新查看结果','info');
    setTimeout(loadTasks,3000);
    setTimeout(loadTasks,8000);
  }catch(e){toast(e.message,'error')}
  finally{if(btn){btn.disabled=false;btn.textContent='🔍 检测'}}
}

async function updateTask(id){
  if(!confirm('确定立即执行更新？')) return;
  const btn = $('btnUpdate-'+id);
  if(btn){btn.disabled=true;btn.textContent='更新中…'}
  try{
    await api('POST','/api/update/'+id);
    toast('更新任务已启动','info');
    setTimeout(loadTasks,3000);
    setTimeout(loadTasks,10000);
    setTimeout(loadTasks,30000);
  }catch(e){toast(e.message,'error')}
  finally{if(btn){btn.disabled=false;btn.textContent='⬆ 更新'}}
}

async function deleteTask(id, name){
  if(!confirm('确定删除任务「'+name+'」？')) return;
  try{
    await api('DELETE','/api/tasks/'+id);
    toast('已删除','success');
    loadTasks();
  }catch(e){toast(e.message,'error')}
}

// ---- Modal ----
function openModal(){
  editID = null;
  $('modalTitle').textContent='添加任务';
  ['fName','fRepo','fVersion','fKeyword','fRename','fTarget','fPre','fPost','fCron']
    .forEach(id=>$(id).value='');
  $('fType').value='core';
  $('modalBg').classList.remove('hidden');
}

function openEdit(id){
  const t = tasks.find(x=>x.id===id);
  if(!t) return;
  editID = id;
  $('modalTitle').textContent='编辑任务';
  $('fName').value=t.name||'';
  $('fType').value=t.update_type||'core';
  $('fRepo').value=t.repo_url||'';
  $('fVersion').value=t.current_version||'';
  $('fKeyword').value=t.file_keyword||'';
  $('fRename').value=t.rename||'';
  $('fTarget').value=t.target_path||'';
  $('fPre').value=t.pre_cmd||'';
  $('fPost').value=t.post_cmd||'';
  $('fCron').value=t.cron||'';
  $('modalBg').classList.remove('hidden');
}

function closeModal(){
  $('modalBg').classList.add('hidden');
}

async function saveTask(){
  const name=$('fName').value.trim();
  const repo=$('fRepo').value.trim();
  const keyword=$('fKeyword').value.trim();
  const target=$('fTarget').value.trim();
  if(!name){toast('请填写任务名称','error');return}
  if(!repo){toast('请填写 GitHub 项目地址','error');return}
  if(!keyword){toast('请填写下载文件关键词','error');return}
  if(!target){toast('请填写文件放置路径','error');return}

  const body={
    name, repo_url:repo,
    update_type:$('fType').value,
    current_version:$('fVersion').value.trim(),
    file_keyword:keyword,
    rename:$('fRename').value.trim(),
    target_path:target,
    pre_cmd:$('fPre').value.trim(),
    post_cmd:$('fPost').value.trim(),
    cron:$('fCron').value.trim(),
  };

  try{
    if(editID){
      await api('PUT','/api/tasks/'+editID, body);
      toast('已保存','success');
    }else{
      await api('POST','/api/tasks', body);
      toast('任务已创建','success');
    }
    closeModal();
    loadTasks();
  }catch(e){toast(e.message,'error')}
}

// ---- Log Modal ----
function openLog(id, name){
  logTaskID = id;
  $('logModalTitle').textContent = '日志 — '+name;
  $('logModalBg').classList.remove('hidden');
  refreshLog();
}

function closeLogModal(){
  $('logModalBg').classList.add('hidden');
  logTaskID = null;
}

async function refreshLog(){
  if(!logTaskID) return;
  try{
    const data = await api('GET','/api/logs/'+logTaskID);
    const box = $('logBox');
    if(!data.log){box.textContent='（暂无日志）';return}
    box.innerHTML = colorizeLog(escHtml(data.log));
    box.scrollTop = box.scrollHeight;
  }catch(e){toast(e.message,'error')}
}

function colorizeLog(text){
  return text
    .replace(/(={10,}.*?={10,})/g,'<span class="log-sep">$1</span>')
    .replace(/(✅[^\n]*|🎉[^\n]*|✓[^\n]*)/g,'<span class="log-ok">$1</span>')
    .replace(/(❌[^\n]*)/g,'<span class="log-err">$1</span>')
    .replace(/(⬇[^\n]*|📦[^\n]*|🔍[^\n]*|📋[^\n]*|📂[^\n]*|⚙[^\n]*)/g,'<span class="log-info">$1</span>')
    .replace(/(⚠[^\n]*|🆕[^\n]*)/g,'<span class="log-warn">$1</span>');
}

// Close modals on background click
$('modalBg').addEventListener('click',e=>{if(e.target===$('modalBg'))closeModal()});
$('logModalBg').addEventListener('click',e=>{if(e.target===$('logModalBg'))closeLogModal()});

// Auto-refresh every 10s
setInterval(loadTasks, 10000);
loadTasks();
</script>
</body>
</html>`
