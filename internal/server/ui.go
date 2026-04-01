package server

import "net/http"

const uiHTML = `<!DOCTYPE html><html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Paddock — Stockyard</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital,wght@0,400;0,700;1,400&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
<style>:root{
  --bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;
  --rust:#c45d2c;--rust-light:#e8753a;--rust-dark:#8b3d1a;
  --leather:#a0845c;--leather-light:#c4a87a;
  --cream:#f0e6d3;--cream-dim:#bfb5a3;--cream-muted:#7a7060;
  --gold:#d4a843;--green:#5ba86e;--red:#c0392b;
  --font-serif:'Libre Baskerville',Georgia,serif;
  --font-mono:'JetBrains Mono',monospace;
}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--font-serif);min-height:100vh;overflow-x:hidden}
a{color:var(--rust-light);text-decoration:none}a:hover{color:var(--gold)}
.hdr{background:var(--bg2);border-bottom:2px solid var(--rust-dark);padding:.9rem 1.8rem;display:flex;align-items:center;justify-content:space-between;gap:1rem}
.hdr-left{display:flex;align-items:center;gap:1rem}
.hdr-brand{font-family:var(--font-mono);font-size:.75rem;color:var(--leather);letter-spacing:3px;text-transform:uppercase}
.hdr-title{font-family:var(--font-mono);font-size:1.1rem;color:var(--cream);letter-spacing:1px}
.badge{font-family:var(--font-mono);font-size:.6rem;padding:.2rem .6rem;letter-spacing:1px;text-transform:uppercase;border:1px solid}
.badge-free{color:var(--green);border-color:var(--green)}
.badge-pro{color:var(--gold);border-color:var(--gold)}
.main{max-width:1000px;margin:0 auto;padding:2rem 1.5rem}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:1rem;margin-bottom:2rem}
.card{background:var(--bg2);border:1px solid var(--bg3);padding:1.2rem 1.5rem}
.card-val{font-family:var(--font-mono);font-size:1.8rem;font-weight:700;color:var(--cream);display:block}
.card-lbl{font-family:var(--font-mono);font-size:.62rem;letter-spacing:2px;text-transform:uppercase;color:var(--leather);margin-top:.3rem}
.section{margin-bottom:2.5rem}
.section-title{font-family:var(--font-mono);font-size:.68rem;letter-spacing:3px;text-transform:uppercase;color:var(--rust-light);margin-bottom:.8rem;padding-bottom:.5rem;border-bottom:1px solid var(--bg3)}
table{width:100%;border-collapse:collapse;font-family:var(--font-mono);font-size:.78rem}
th{background:var(--bg3);padding:.5rem .8rem;text-align:left;color:var(--leather-light);font-weight:400;letter-spacing:1px;font-size:.65rem;text-transform:uppercase}
td{padding:.5rem .8rem;border-bottom:1px solid var(--bg3);color:var(--cream-dim);vertical-align:top}
tr:hover td{background:var(--bg2)}
.empty{color:var(--cream-muted);text-align:center;padding:2rem;font-style:italic}
.btn{font-family:var(--font-mono);font-size:.75rem;padding:.4rem 1rem;border:1px solid var(--leather);background:transparent;color:var(--cream);cursor:pointer;transition:all .2s}
.btn:hover{border-color:var(--rust-light);color:var(--rust-light)}
.btn-rust{border-color:var(--rust);color:var(--rust-light)}.btn-rust:hover{background:var(--rust);color:var(--cream)}
.btn-sm{font-size:.65rem;padding:.25rem .6rem}
.pill{display:inline-block;font-family:var(--font-mono);font-size:.6rem;padding:.1rem .4rem;border-radius:2px;text-transform:uppercase}
.pill-up{background:#1a3a2a;color:var(--green)}.pill-down{background:#2a1a1a;color:var(--red)}.pill-unknown{background:var(--bg3);color:var(--cream-muted)}
.mono{font-family:var(--font-mono);font-size:.78rem}
.lbl{font-family:var(--font-mono);font-size:.62rem;letter-spacing:1px;text-transform:uppercase;color:var(--leather)}
input{font-family:var(--font-mono);font-size:.78rem;background:var(--bg3);border:1px solid var(--bg3);color:var(--cream);padding:.4rem .7rem;outline:none}
input:focus{border-color:var(--leather)}
.row{display:flex;gap:.8rem;align-items:flex-end;flex-wrap:wrap;margin-bottom:1rem}
.field{display:flex;flex-direction:column;gap:.3rem}
.tabs{display:flex;gap:0;margin-bottom:1.5rem;border-bottom:1px solid var(--bg3)}
.tab{font-family:var(--font-mono);font-size:.72rem;padding:.6rem 1.2rem;color:var(--cream-muted);cursor:pointer;border-bottom:2px solid transparent;letter-spacing:1px;text-transform:uppercase}
.tab:hover{color:var(--cream-dim)}
.tab.active{color:var(--rust-light);border-bottom-color:var(--rust-light)}
.tab-content{display:none}.tab-content.active{display:block}
.uptime-bar{display:flex;gap:1px;height:20px;margin-top:.3rem}
.uptime-bar span{flex:1;border-radius:1px}
.uptime-bar .up{background:var(--green)}.uptime-bar .down{background:var(--red)}.uptime-bar .none{background:var(--bg3)}
pre{background:var(--bg3);padding:.8rem 1rem;font-family:var(--font-mono);font-size:.72rem;color:var(--cream-dim);overflow-x:auto}
</style></head><body>
<div class="hdr">
  <div class="hdr-left">
    <svg viewBox="0 0 64 64" width="22" height="22" fill="none"><rect x="8" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="28" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="48" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="8" y="27" width="48" height="7" rx="2.5" fill="#c4a87a"/></svg>
    <span class="hdr-brand">Stockyard</span>
    <span class="hdr-title">Paddock</span>
  </div>
  <div style="display:flex;gap:.8rem;align-items:center">
    <span class="badge badge-free">Free</span>
    <a href="/status" class="lbl" style="color:var(--leather)">Status</a>
    <a href="/api/stats" class="lbl" style="color:var(--leather)">API</a>
  </div>
</div>
<div class="main">

<div class="cards" id="stat-cards">
  <div class="card"><span class="card-val" id="s-monitors">—</span><span class="card-lbl">Monitors</span></div>
  <div class="card"><span class="card-val" id="s-up">—</span><span class="card-lbl">Up</span></div>
  <div class="card"><span class="card-val" id="s-down">—</span><span class="card-lbl">Down</span></div>
  <div class="card"><span class="card-val" id="s-checks">—</span><span class="card-lbl">Checks</span></div>
</div>

<div class="tabs">
  <div class="tab active" onclick="switchTab('monitors')">Monitors</div>
  <div class="tab" onclick="switchTab('add')">Add Monitor</div>
  <div class="tab" onclick="switchTab('usage')">Usage</div>
</div>

<div id="tab-monitors" class="tab-content active">
  <div class="section">
    <div class="section-title">Monitors</div>
    <table><thead><tr>
      <th>Name</th><th>URL</th><th>Status</th><th>Response</th><th>Interval</th><th>Last Check</th><th></th>
    </tr></thead><tbody id="monitors-body"></tbody></table>
  </div>
</div>

<div id="tab-add" class="tab-content">
  <div class="section">
    <div class="section-title">Add Monitor</div>
    <div class="row">
      <div class="field"><span class="lbl">Name</span><input id="c-name" placeholder="My API" style="width:160px"></div>
      <div class="field"><span class="lbl">URL</span><input id="c-url" placeholder="https://api.example.com/health" style="width:300px"></div>
      <div class="field"><span class="lbl">Interval (s)</span><input id="c-interval" placeholder="300" style="width:80px" type="number"></div>
      <button class="btn btn-rust" onclick="addMonitor()">Add</button>
    </div>
    <div id="c-result" style="margin-top:.5rem"></div>
  </div>
</div>

<div id="tab-usage" class="tab-content">
  <div class="section">
    <div class="section-title">Quick Start</div>
    <pre>
# Add a monitor
curl -X POST http://localhost:8820/api/monitors \
  -H "Content-Type: application/json" \
  -d '{"name":"My API","url":"https://api.example.com/health","interval_seconds":300}'

# List monitors
curl http://localhost:8820/api/monitors

# Check history
curl http://localhost:8820/api/monitors/{id}/history

# Public status page
open http://localhost:8820/status

# Status API (JSON)
curl http://localhost:8820/api/status
    </pre>
  </div>
</div>

</div>
<script>
function switchTab(name){
  document.querySelectorAll('.tab').forEach(t=>t.classList.toggle('active',t.textContent.toLowerCase().replace(/\s/g,'')===name||t.textContent.toLowerCase().replace(/\s/g,'')==='add'&&name==='add'));
  document.querySelectorAll('.tab-content').forEach(t=>t.classList.toggle('active',t.id==='tab-'+name));
}

async function refresh(){
  try{
    const sr=await fetch('/api/stats');const st=await sr.json();
    document.getElementById('s-monitors').textContent=st.monitors||0;
    document.getElementById('s-up').textContent=st.up||0;
    document.getElementById('s-down').textContent=st.down||0;
    document.getElementById('s-checks').textContent=fmt(st.checks||0);
  }catch(e){}
  try{
    const mr=await fetch('/api/monitors');const md=await mr.json();
    const mons=md.monitors||[];
    const tb=document.getElementById('monitors-body');
    if(!mons.length){tb.innerHTML='<tr><td colspan="7" class="empty">No monitors yet. Add one to get started.</td></tr>';return;}
    tb.innerHTML=mons.map(m=>{
      const st=m.last_status;
      const cls=st==='up'?'pill-up':st==='down'?'pill-down':'pill-unknown';
      return '<tr>'+
        '<td style="color:var(--cream);font-weight:600">'+esc(m.name)+'</td>'+
        '<td style="font-size:.68rem;word-break:break-all">'+esc(m.url)+'</td>'+
        '<td><span class="pill '+cls+'">'+st+'</span></td>'+
        '<td>'+(m.last_response_ms||'—')+'ms</td>'+
        '<td>'+m.interval_seconds+'s</td>'+
        '<td style="font-size:.68rem;color:var(--cream-muted)">'+timeAgo(m.last_checked_at)+'</td>'+
        '<td><button class="btn btn-sm" onclick="deleteMon(\''+m.id+'\')">Delete</button></td>'+
        '</tr>';
    }).join('');
  }catch(e){}
}

async function addMonitor(){
  const name=document.getElementById('c-name').value.trim();
  const url=document.getElementById('c-url').value.trim();
  const interval=parseInt(document.getElementById('c-interval').value)||300;
  if(!url){document.getElementById('c-result').innerHTML='<span style="color:var(--red)">URL is required</span>';return;}
  try{
    const r=await fetch('/api/monitors',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:name||url,url,interval_seconds:interval})});
    const d=await r.json();
    if(r.ok){
      document.getElementById('c-result').innerHTML='<span style="color:var(--green)">Monitor added</span>';
      document.getElementById('c-name').value='';document.getElementById('c-url').value='';
      refresh();
    }else{document.getElementById('c-result').innerHTML='<span style="color:var(--red)">'+esc(d.error)+'</span>';}
  }catch(e){document.getElementById('c-result').innerHTML='<span style="color:var(--red)">'+e.message+'</span>';}
}

async function deleteMon(id){
  if(!confirm('Delete this monitor?'))return;
  await fetch('/api/monitors/'+id,{method:'DELETE'});
  refresh();
}

function fmt(n){if(n>=1e6)return(n/1e6).toFixed(1)+'M';if(n>=1e3)return(n/1e3).toFixed(1)+'K';return n;}
function esc(s){const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
function timeAgo(s){if(!s)return'—';const d=new Date(s);const diff=Date.now()-d.getTime();if(diff<0)return'just now';if(diff<60000)return'just now';if(diff<3600000)return Math.floor(diff/60000)+'m ago';if(diff<86400000)return Math.floor(diff/3600000)+'h ago';return Math.floor(diff/86400000)+'d ago';}

refresh();
setInterval(refresh,8000);
</script></body></html>`

// statusPageHTML is the public status page — no auth, shareable URL.
const statusPageHTML = `<!DOCTYPE html><html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Status — Stockyard Paddock</title>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--cream:#f0e6d3;--cream-dim:#bfb5a3;--cream-muted:#7a7060;--green:#5ba86e;--red:#c0392b;--rust-light:#e8753a;--leather:#a0845c;--gold:#d4a843;--font:'JetBrains Mono',monospace}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--font);min-height:100vh;padding:2rem}
.container{max-width:700px;margin:0 auto}
.header{text-align:center;margin-bottom:2rem}
.header h1{font-size:1.1rem;letter-spacing:2px;color:var(--cream)}
.overall{text-align:center;padding:1.2rem;margin-bottom:2rem;font-size:.85rem;letter-spacing:1px;text-transform:uppercase}
.overall-op{background:#1a3a2a;border:1px solid var(--green);color:var(--green)}
.overall-deg{background:#2a1a1a;border:1px solid var(--red);color:var(--red)}
.service{background:var(--bg2);border:1px solid var(--bg3);padding:1rem 1.2rem;margin-bottom:.5rem;display:flex;justify-content:space-between;align-items:center}
.svc-name{font-size:.82rem;color:var(--cream)}
.svc-right{display:flex;align-items:center;gap:1rem;font-size:.72rem}
.svc-uptime{color:var(--cream-dim)}
.dot{width:10px;height:10px;border-radius:50%;display:inline-block}
.dot-up{background:var(--green)}.dot-down{background:var(--red)}.dot-unknown{background:var(--cream-muted)}
.svc-ms{color:var(--cream-muted)}
.footer{text-align:center;margin-top:2rem;font-size:.65rem;color:var(--cream-muted);letter-spacing:1px}
.footer a{color:var(--rust-light);text-decoration:none}
.updated{text-align:center;font-size:.65rem;color:var(--cream-muted);margin-bottom:1.5rem}
</style></head><body>
<div class="container">
  <div class="header"><h1>System Status</h1></div>
  <div id="overall" class="overall overall-op">All Systems Operational</div>
  <div id="updated" class="updated"></div>
  <div id="services"></div>
  <div class="footer">Powered by <a href="https://stockyard.dev/paddock/">Stockyard Paddock</a></div>
</div>
<script>
async function load(){
  try{
    const r=await fetch('/api/status');const d=await r.json();
    const o=document.getElementById('overall');
    if(d.status==='operational'){o.className='overall overall-op';o.textContent='All Systems Operational';}
    else if(d.status==='degraded'){o.className='overall overall-deg';o.textContent='Some Systems Degraded';}
    else{o.className='overall';o.textContent='No monitors configured';o.style.borderColor='var(--cream-muted)';o.style.color='var(--cream-muted)';}
    document.getElementById('updated').textContent='Last updated: '+new Date().toLocaleString();
    const svcs=d.services||[];
    document.getElementById('services').innerHTML=svcs.map(s=>{
      const dc=s.status==='up'?'dot-up':s.status==='down'?'dot-down':'dot-unknown';
      return '<div class="service"><span class="svc-name">'+esc(s.name)+'</span><div class="svc-right"><span class="svc-uptime">'+s.uptime_30d.toFixed(1)+'%</span><span class="svc-ms">'+s.response_ms+'ms</span><span class="dot '+dc+'"></span></div></div>';
    }).join('')||'<div style="text-align:center;color:var(--cream-muted);padding:2rem;font-style:italic">No monitors configured</div>';
  }catch(e){console.error(e);}
}
function esc(s){const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
load();
setInterval(load,30000);
</script></body></html>`

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(uiHTML))
}
