package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Paddock</title>
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--orange:#d4843a;--blue:#4a7ec9;--mono:'JetBrains Mono',monospace;--serif:'Libre Baskerville',serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--serif);line-height:1.6}
a{color:var(--rust);text-decoration:none}
.header{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}
.header h1{font-family:var(--mono);font-size:.9rem;letter-spacing:2px}
.overall{font-family:var(--mono);font-size:.72rem;padding:.3rem .8rem;border:1px solid var(--bg3)}
.overall.operational{color:var(--green);border-color:var(--green)}.overall.degraded{color:var(--orange);border-color:var(--orange)}.overall.major_outage,.overall.partial_outage{color:var(--red);border-color:var(--red)}.overall.maintenance{color:var(--blue);border-color:var(--blue)}
.tabs{display:flex;border-bottom:1px solid var(--bg3);padding:0 1.5rem;gap:0;font-family:var(--mono);font-size:.75rem}
.tab{padding:.7rem 1.2rem;cursor:pointer;color:var(--cm);border-bottom:2px solid transparent}.tab:hover{color:var(--cream)}.tab.active{color:var(--rust);border-color:var(--rust)}
.content{padding:1.5rem;max-width:900px;margin:0 auto}
.card{background:var(--bg2);border:1px solid var(--bg3);margin-bottom:.8rem;padding:1rem 1.2rem}
.card-header{display:flex;justify-content:space-between;align-items:center;margin-bottom:.3rem}
.card-name{font-family:var(--mono);font-size:.82rem}
.card-desc{font-size:.78rem;color:var(--cm)}
.badge{font-family:var(--mono);font-size:.6rem;padding:.15rem .5rem;text-transform:uppercase;letter-spacing:1px}
.badge-operational{background:#4a9e5c22;color:var(--green);border:1px solid #4a9e5c44}
.badge-degraded{background:#d4843a22;color:var(--orange);border:1px solid #d4843a44}
.badge-partial_outage,.badge-major_outage{background:#c9444422;color:var(--red);border:1px solid #c9444444}
.badge-maintenance{background:#4a7ec922;color:var(--blue);border:1px solid #4a7ec944}
.badge-investigating{background:#d4843a22;color:var(--orange);border:1px solid #d4843a44}
.badge-identified{background:#d4843a22;color:var(--orange);border:1px solid #d4843a44}
.badge-monitoring{background:#4a7ec922;color:var(--blue);border:1px solid #4a7ec944}
.badge-resolved{background:#4a9e5c22;color:var(--green);border:1px solid #4a9e5c44}
.badge-minor{background:#d4843a22;color:var(--orange)}.badge-major{background:#c9444422;color:var(--red)}.badge-critical{background:#c9444422;color:var(--red)}
.btn{font-family:var(--mono);font-size:.7rem;padding:.35rem .8rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd);transition:all .15s}
.btn:hover{border-color:var(--leather);color:var(--cream)}
.btn-primary{background:var(--rust);border-color:var(--rust);color:var(--bg)}.btn-primary:hover{opacity:.85}
.btn-sm{font-size:.6rem;padding:.2rem .5rem}
.form-row{margin-bottom:.8rem}
.form-row label{display:block;font-family:var(--mono);font-size:.65rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.3rem}
.form-row input,.form-row select,.form-row textarea{width:100%;padding:.5rem .7rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.78rem}
.form-row textarea{min-height:60px;resize:vertical}
.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.6);z-index:100;align-items:center;justify-content:center}
.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:420px;max-width:90vw;max-height:90vh;overflow-y:auto}
.modal h2{font-family:var(--mono);font-size:.8rem;margin-bottom:1rem;color:var(--rust)}
.actions{display:flex;gap:.5rem;justify-content:flex-end;margin-top:1rem}
.timeline{border-left:2px solid var(--bg3);padding-left:1rem;margin:.8rem 0}
.tl-entry{margin-bottom:.8rem;position:relative}
.tl-entry::before{content:'';position:absolute;left:-1.35rem;top:.4rem;width:8px;height:8px;border-radius:50%;background:var(--bg3)}
.tl-time{font-family:var(--mono);font-size:.6rem;color:var(--cm)}
.tl-body{font-size:.8rem;color:var(--cd);margin-top:.2rem}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic}
.stats-row{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:.8rem;margin-bottom:1.5rem}
.stat{background:var(--bg2);border:1px solid var(--bg3);padding:1rem;text-align:center}
.stat-val{font-family:var(--mono);font-size:1.4rem;color:var(--cream)}
.stat-label{font-family:var(--mono);font-size:.6rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-top:.2rem}
.sub-count{font-family:var(--mono);font-size:.7rem;color:var(--cm);margin-top:.5rem}
</style>
</head>
<body>
<div class="header">
  <h1>PADDOCK</h1>
  <div id="overall" class="overall">Loading...</div>
</div>
<div class="tabs">
  <div class="tab active" onclick="showTab('components')">Components</div>
  <div class="tab" onclick="showTab('incidents')">Incidents</div>
  <div class="tab" onclick="showTab('public')">Public Page</div>
  <div class="tab" onclick="showTab('subscribers')">Subscribers</div>
</div>
<div class="content" id="main"></div>

<div class="modal-bg" id="modalBg" onclick="if(event.target===this)closeModal()">
  <div class="modal" id="modal"></div>
</div>

<script>
const API='/api';
let tab='components',components=[],incidents=[],subscribers=[];

async function load(){
  const[c,i,sub,st]=await Promise.all([
    fetch(API+'/components').then(r=>r.json()),
    fetch(API+'/incidents').then(r=>r.json()),
    fetch(API+'/subscribers').then(r=>r.json()),
    fetch(API+'/stats').then(r=>r.json()),
  ]);
  components=c.components||[];incidents=i.incidents||[];subscribers=sub.subscribers||[];
  const ov=document.getElementById('overall');
  const s=st.overall_status||'operational';
  ov.textContent=s.replace(/_/g,' ').toUpperCase();
  ov.className='overall '+s;
  render();
}

function showTab(t){
  tab=t;
  document.querySelectorAll('.tab').forEach((el,i)=>el.classList.toggle('active',['components','incidents','public','subscribers'][i]===t));
  render();
}

function render(){
  const m=document.getElementById('main');
  if(tab==='components') renderComponents(m);
  else if(tab==='incidents') renderIncidents(m);
  else if(tab==='public') renderPublic(m);
  else renderSubscribers(m);
}

function renderComponents(m){
  let h='<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem"><h2 style="font-family:var(--mono);font-size:.75rem;color:var(--leather)">COMPONENTS</h2><button class="btn btn-primary" onclick="openComponentForm()">+ Add Component</button></div>';
  if(!components.length){h+='<div class="empty">No components yet. Add your first service to monitor.</div>';}
  else{components.forEach(c=>{
    h+='<div class="card"><div class="card-header"><span class="card-name">'+esc(c.name)+'</span><div style="display:flex;gap:.5rem;align-items:center"><select class="btn-sm" style="background:var(--bg);border:1px solid var(--bg3);color:var(--cd);font-family:var(--mono);font-size:.6rem;padding:.2rem" onchange="setStatus(\''+c.id+'\',this.value)"><option value="operational"'+(c.status==='operational'?' selected':'')+'>Operational</option><option value="degraded"'+(c.status==='degraded'?' selected':'')+'>Degraded</option><option value="partial_outage"'+(c.status==='partial_outage'?' selected':'')+'>Partial Outage</option><option value="major_outage"'+(c.status==='major_outage'?' selected':'')+'>Major Outage</option><option value="maintenance"'+(c.status==='maintenance'?' selected':'')+'>Maintenance</option></select><span class="badge badge-'+c.status+'">'+c.status.replace(/_/g,' ')+'</span><button class="btn btn-sm" onclick="delComponent(\''+c.id+'\')">&#x2715;</button></div></div>';
    if(c.description)h+='<div class="card-desc">'+esc(c.description)+'</div>';
    if(c.group)h+='<div style="font-family:var(--mono);font-size:.6rem;color:var(--cm);margin-top:.2rem">Group: '+esc(c.group)+'</div>';
    h+='</div>';
  });}
  m.innerHTML=h;
}

function renderIncidents(m){
  let h='<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem"><h2 style="font-family:var(--mono);font-size:.75rem;color:var(--leather)">INCIDENTS</h2><button class="btn btn-primary" onclick="openIncidentForm()">+ New Incident</button></div>';
  const active=incidents.filter(i=>i.status!=='resolved');
  const resolved=incidents.filter(i=>i.status==='resolved');
  if(!incidents.length){h+='<div class="empty">No incidents recorded. That\'s a good thing.</div>';}
  if(active.length){h+='<div style="font-family:var(--mono);font-size:.65rem;color:var(--red);margin-bottom:.5rem;text-transform:uppercase;letter-spacing:1px">Active ('+active.length+')</div>';active.forEach(i=>{h+=incidentCard(i)});}
  if(resolved.length){h+='<div style="font-family:var(--mono);font-size:.65rem;color:var(--green);margin:1rem 0 .5rem;text-transform:uppercase;letter-spacing:1px">Resolved ('+resolved.length+')</div>';resolved.forEach(i=>{h+=incidentCard(i)});}
  m.innerHTML=h;
}

function incidentCard(i){
  let h='<div class="card"><div class="card-header"><span class="card-name">'+esc(i.title)+'</span><div style="display:flex;gap:.4rem"><span class="badge badge-'+i.impact+'">'+i.impact+'</span><span class="badge badge-'+i.status+'">'+i.status+'</span></div></div>';
  h+='<div style="font-family:var(--mono);font-size:.6rem;color:var(--cm);margin:.3rem 0">'+fmtTime(i.created_at);
  if(i.resolved_at)h+=' — Resolved '+fmtTime(i.resolved_at);
  h+='</div>';
  if(i.updates&&i.updates.length){
    h+='<div class="timeline">';
    i.updates.forEach(u=>{
      h+='<div class="tl-entry"><div class="tl-time"><span class="badge badge-'+u.status+'" style="margin-right:.3rem">'+u.status+'</span>'+fmtTime(u.created_at)+'</div><div class="tl-body">'+esc(u.body)+'</div></div>';
    });
    h+='</div>';
  }
  h+='<div style="display:flex;gap:.4rem;margin-top:.5rem"><button class="btn btn-sm" onclick="openUpdateForm(\''+i.id+'\')">Post Update</button><button class="btn btn-sm" onclick="delIncident(\''+i.id+'\')">Delete</button></div></div>';
  return h;
}

function renderPublic(m){
  const overall=document.getElementById('overall').textContent;
  let h='<div style="text-align:center;padding:2rem 0"><div style="font-family:var(--mono);font-size:.65rem;color:var(--cm);text-transform:uppercase;letter-spacing:2px;margin-bottom:.5rem">Current Status</div>';
  const s=overall.toLowerCase().replace(/ /g,'_');
  h+='<div class="badge badge-'+s+'" style="font-size:1rem;padding:.5rem 1.5rem">'+overall+'</div></div>';
  h+='<div style="font-family:var(--mono);font-size:.65rem;color:var(--leather);margin-bottom:.5rem;text-transform:uppercase;letter-spacing:1px">Components</div>';
  components.forEach(c=>{
    h+='<div style="display:flex;justify-content:space-between;padding:.5rem 0;border-bottom:1px solid var(--bg3);font-size:.82rem"><span>'+esc(c.name)+'</span><span class="badge badge-'+c.status+'">'+c.status.replace(/_/g,' ')+'</span></div>';
  });
  const active=incidents.filter(i=>i.status!=='resolved');
  if(active.length){
    h+='<div style="font-family:var(--mono);font-size:.65rem;color:var(--leather);margin:1.5rem 0 .5rem;text-transform:uppercase;letter-spacing:1px">Active Incidents</div>';
    active.forEach(i=>{h+=incidentCard(i);});
  }
  h+='<div style="text-align:center;margin-top:2rem;font-family:var(--mono);font-size:.7rem;color:var(--cm)">This is how your public status page looks to visitors.<br>Embed at <code>/api/status</code></div>';
  m.innerHTML=h;
}

function renderSubscribers(m){
  let h='<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem"><h2 style="font-family:var(--mono);font-size:.75rem;color:var(--leather)">SUBSCRIBERS</h2><div class="sub-count">'+subscribers.length+' subscribers</div></div>';
  if(!subscribers.length){h+='<div class="empty">No subscribers yet. Users can subscribe via POST /api/subscribers.</div>';}
  else{subscribers.forEach(s=>{
    h+='<div class="card" style="padding:.6rem 1rem"><div class="card-header"><span style="font-family:var(--mono);font-size:.78rem">'+esc(s.email)+'</span><div style="display:flex;gap:.4rem;align-items:center"><span style="font-family:var(--mono);font-size:.6rem;color:var(--cm)">'+fmtTime(s.created_at)+'</span><button class="btn btn-sm" onclick="unsub(\''+s.email+'\')">Remove</button></div></div></div>';
  });}
  m.innerHTML=h;
}

// ── Actions ──

async function setStatus(id,status){
  await fetch(API+'/components/'+id+'/status',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({status})});
  load();
}

async function delComponent(id){if(confirm('Delete this component?')){await fetch(API+'/components/'+id,{method:'DELETE'});load();}}
async function delIncident(id){if(confirm('Delete this incident?')){await fetch(API+'/incidents/'+id,{method:'DELETE'});load();}}
async function unsub(email){await fetch(API+'/subscribers/'+encodeURIComponent(email),{method:'DELETE'});load();}

function openComponentForm(){
  document.getElementById('modal').innerHTML='<h2>Add Component</h2><div class="form-row"><label>Name</label><input id="f-name" placeholder="e.g. API Gateway"></div><div class="form-row"><label>Description</label><input id="f-desc" placeholder="Primary REST API"></div><div class="form-row"><label>Group</label><input id="f-group" placeholder="e.g. Core Infrastructure"></div><div class="form-row"><label>Status</label><select id="f-status"><option value="operational">Operational</option><option value="degraded">Degraded</option><option value="partial_outage">Partial Outage</option><option value="major_outage">Major Outage</option><option value="maintenance">Maintenance</option></select></div><div class="actions"><button class="btn" onclick="closeModal()">Cancel</button><button class="btn btn-primary" onclick="submitComponent()">Create</button></div>';
  document.getElementById('modalBg').classList.add('open');
}

async function submitComponent(){
  await fetch(API+'/components',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:document.getElementById('f-name').value,description:document.getElementById('f-desc').value,group:document.getElementById('f-group').value,status:document.getElementById('f-status').value})});
  closeModal();load();
}

function openIncidentForm(){
  let opts=components.map(c=>'<option value="'+c.id+'">'+esc(c.name)+'</option>').join('');
  document.getElementById('modal').innerHTML='<h2>New Incident</h2><div class="form-row"><label>Title</label><input id="f-title" placeholder="e.g. Elevated API latency"></div><div class="form-row"><label>Impact</label><select id="f-impact"><option value="minor">Minor</option><option value="major">Major</option><option value="critical">Critical</option></select></div><div class="form-row"><label>Affected Component</label><select id="f-comp"><option value="">None</option>'+opts+'</select></div><div class="form-row"><label>Initial Update</label><textarea id="f-body" placeholder="We are investigating reports of..."></textarea></div><div class="actions"><button class="btn" onclick="closeModal()">Cancel</button><button class="btn btn-primary" onclick="submitIncident()">Create</button></div>';
  document.getElementById('modalBg').classList.add('open');
}

async function submitIncident(){
  const inc={title:document.getElementById('f-title').value,impact:document.getElementById('f-impact').value,component_id:document.getElementById('f-comp').value};
  const r=await fetch(API+'/incidents',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(inc)}).then(r=>r.json());
  const body=document.getElementById('f-body').value;
  if(body&&r.id){await fetch(API+'/incidents/'+r.id+'/updates',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({status:'investigating',body})});}
  closeModal();load();
}

function openUpdateForm(incId){
  document.getElementById('modal').innerHTML='<h2>Post Incident Update</h2><div class="form-row"><label>Status</label><select id="f-ustatus"><option value="investigating">Investigating</option><option value="identified">Identified</option><option value="monitoring">Monitoring</option><option value="resolved">Resolved</option></select></div><div class="form-row"><label>Update</label><textarea id="f-ubody" placeholder="Describe the current situation..."></textarea></div><div class="actions"><button class="btn" onclick="closeModal()">Cancel</button><button class="btn btn-primary" onclick="submitUpdate(\''+incId+'\')">Post</button></div>';
  document.getElementById('modalBg').classList.add('open');
}

async function submitUpdate(incId){
  await fetch(API+'/incidents/'+incId+'/updates',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({status:document.getElementById('f-ustatus').value,body:document.getElementById('f-ubody').value})});
  closeModal();load();
}

function closeModal(){document.getElementById('modalBg').classList.remove('open');}
function esc(s){const d=document.createElement('div');d.textContent=s;return d.innerHTML;}
function fmtTime(t){if(!t)return'';const d=new Date(t);return d.toLocaleDateString()+' '+d.toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'});}

load();
</script>
</body>
</html>`
