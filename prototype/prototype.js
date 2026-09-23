/* NewLDAP 产品原型 · 交互与模拟数据（纯前端演示） */
"use strict";
const $ = (s, r=document) => r.querySelector(s);
const $$ = (s, r=document) => Array.from(r.querySelectorAll(s));
const ic = (name, cls="ic ic-14") => `<i data-lucide="${name}" class="${cls}"></i>`;
const esc = s => String(s).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;");
function refreshIcons(){ lucide.createIcons(); }

/* ---------- Toast ---------- */
function toast(msg, kind="ok", icon){
  const t = document.createElement("div");
  t.className = "toast " + (kind==="ok"?"ok":kind);
  t.innerHTML = ic(icon || (kind==="warn"?"triangle-alert":"circle-check"),"ic ic-16") + `<span>${msg}</span>`;
  $("#toasts").appendChild(t);
  refreshIcons();
  setTimeout(()=>{ t.style.transition="opacity .25s"; t.style.opacity="0"; setTimeout(()=>t.remove(),260); }, 3200);
}

/* ---------- 模拟数据 ---------- */
const ROOT = "dc=example,dc=cn";
const TREE = {
  name:"示例公司", dn:ROOT, icon:"building-2", children:[
    { name:"技术研发部", dn:"ou=tech,"+ROOT, icon:"folder", children:[
      { name:"平台研发组", dn:"ou=platform,ou=tech,"+ROOT, icon:"folder", children:[] },
      { name:"应用研发组", dn:"ou=app,ou=tech,"+ROOT, icon:"folder", children:[] },
      { name:"运维组", dn:"ou=ops,ou=tech,"+ROOT, icon:"folder", children:[] },
    ]},
    { name:"产品设计部", dn:"ou=product,"+ROOT, icon:"folder", children:[] },
    { name:"市场运营部", dn:"ou=market,"+ROOT, icon:"folder", children:[] },
    { name:"人事行政部", dn:"ou=hr,"+ROOT, icon:"folder", children:[] },
    { name:"用户组", dn:"ou=groups,"+ROOT, icon:"boxes", children:[] },
  ]
};
const PEOPLE = [
  { name:"张伟", uid:"zhangwei", emp:"E1001", mail:"zhangwei@example.cn", phone:"138-0101-2233", ou:"ou=tech,"+ROOT, ouName:"技术研发部", title:"平台研发工程师", st:"active", posix:true },
  { name:"李娜", uid:"lina", emp:"E1002", mail:"lina@example.cn", phone:"139-2210-3344", ou:"ou=tech,"+ROOT, ouName:"技术研发部", title:"应用研发工程师", st:"active", posix:true },
  { name:"王强", uid:"wangqiang", emp:"E1003", mail:"wangqiang@example.cn", phone:"137-5520-8899", ou:"ou=tech,"+ROOT, ouName:"技术研发部", title:"运维工程师", st:"disabled", posix:true },
  { name:"刘洋", uid:"liuyang", emp:"E1004", mail:"liuyang@example.cn", phone:"135-8080-1212", ou:"ou=tech,"+ROOT, ouName:"技术研发部", title:"安全工程师", st:"active", posix:false },
  { name:"陈静", uid:"chenjing", emp:"E1005", mail:"chenjing@example.cn", phone:"186-0110-6767", ou:"ou=tech,"+ROOT, ouName:"技术研发部", title:"测试工程师", st:"active", posix:false },
  { name:"赵磊", uid:"zhaolei", emp:"E1006", mail:"zhaolei@example.cn", phone:"188-3345-9090", ou:"ou=platform,ou=tech,"+ROOT, ouName:"平台研发组", title:"架构师", st:"active", posix:true },
  { name:"周杰", uid:"zhoujie", emp:"E1008", mail:"zhoujie@example.cn", phone:"133-9292-4545", ou:"ou=app,ou=tech,"+ROOT, ouName:"应用研发组", title:"前端工程师", st:"active", posix:false },
  { name:"郑浩", uid:"zhenghao", emp:"E1010", mail:"zhenghao@example.cn", phone:"130-5566-7788", ou:"ou=ops,ou=tech,"+ROOT, ouName:"运维组", title:"SRE", st:"active", posix:true },
  { name:"冯雪", uid:"fengxue", emp:"E1011", mail:"fengxue@example.cn", phone:"131-2424-5656", ou:"ou=product,"+ROOT, ouName:"产品设计部", title:"产品经理", st:"active", posix:false },
  { name:"蒋鑫", uid:"jiangxin", emp:"E1012", mail:"jiangxin@example.cn", phone:"132-8686-3434", ou:"ou=product,"+ROOT, ouName:"产品设计部", title:"UI 设计师", st:"active", posix:false },
  { name:"韩梅", uid:"hanmei", emp:"E1013", mail:"hanmei@example.cn", phone:"155-0909-2323", ou:"ou=market,"+ROOT, ouName:"市场运营部", title:"市场专员", st:"active", posix:false },
  { name:"曹颖", uid:"caoying", emp:"E1014", mail:"caoying@example.cn", phone:"156-7676-8989", ou:"ou=hr,"+ROOT, ouName:"人事行政部", title:"HRBP", st:"active", posix:false },
];
const GROUPS = [
  { name:"VPN 权限组", dn:"cn=vpn-access,ou=groups,"+ROOT, type:"auth", oc:"groupOfNames", desc:"远程 VPN 接入授权", members:["zhangwei","liuyang","zhaolei","fengxue","caoying"] },
  { name:"代码管理员组", dn:"cn=git-admin,ou=groups,"+ROOT, type:"auth", oc:"groupOfNames", desc:"Git 仓库管理权限", members:["zhaolei","zhoujie"] },
  { name:"sudo 权限组", dn:"cn=sudoers,ou=groups,"+ROOT, type:"auth", oc:"groupOfNames", desc:"生产主机 sudo 授权", members:["zhenghao","wangqiang"] },
  { name:"开发机登录组", dn:"cn=dev-login,ou=groups,"+ROOT, type:"posix", oc:"posixGroup", desc:"Linux 开发机登录（gidNumber 5000）", members:["zhangwei","lina","zhaolei","zhoujie","zhenghao","liuyang"] },
  { name:"看板只读组", dn:"cn=board-readonly,ou=groups,"+ROOT, type:"auth", oc:"groupOfNames", desc:"监控看板只读访问", members:["chenjing","hanmei"] },
];
const AUDIT = [
  { ts:"2026-09-21 10:32:05", who:"cn=admin,dc=example,dc=cn", src:"10.8.0.22（LAM）", op:"修改属性", dn:"uid=fengxue,ou=product,dc=example,dc=cn", res:"ok" },
  { ts:"2026-09-21 10:18:41", who:"cn=admin,dc=example,dc=cn", src:"10.8.0.15", op:"重置密码", dn:"uid=wangqiang,ou=tech,dc=example,dc=cn", res:"ok" },
  { ts:"2026-09-21 09:56:12", who:"uid=caoying,ou=hr,dc=example,dc=cn", src:"192.168.3.44", op:"登录", dn:"—", res:"ok" },
  { ts:"2026-09-21 09:40:03", who:"cn=admin,dc=example,dc=cn", src:"10.8.0.15", op:"导入批次", dn:"ou=platform + 18 条人员", res:"ok" },
  { ts:"2026-09-20 18:22:37", who:"cn=admin,dc=example,dc=cn", src:"10.8.0.22（LAM）", op:"新建人员", dn:"uid=hanmei,ou=market,dc=example,dc=cn", res:"ok" },
  { ts:"2026-09-20 17:05:19", who:"uid=zhaolei,ou=platform,…", src:"192.168.3.71", op:"登录", dn:"—", res:"fail" },
  { ts:"2026-09-20 16:47:52", who:"cn=admin,dc=example,dc=cn", src:"10.8.0.15", op:"删除部门", dn:"ou=实习组,ou=tech,dc=example,dc=cn", res:"ok" },
  { ts:"2026-09-20 15:31:08", who:"cn=admin,dc=example,dc=cn", src:"10.8.0.15", op:"修改属性", dn:"uid=wangqiang,ou=tech,dc=example,dc=cn", res:"fail" },
];
const PLAN = [
  { row:"3",  type:"dept",  name:"数据智能部", dept:"—", emp:"—", detail:"新建部门 ou=data" },
  { row:"4",  type:"dept",  name:"算法组", dept:"数据智能部", emp:"—", detail:"新建子部门（路径自动创建）" },
  { row:"7",  type:"new",   name:"许倩", dept:"数据智能部", emp:"E1021", detail:"新建人员 · uid=xuqian" },
  { row:"8",  type:"new",   name:"邓超", dept:"算法组", emp:"E1022", detail:"新建人员 · uid=dengchao" },
  { row:"9",  type:"new",   name:"崔敏", dept:"数据智能部", emp:"E1023", detail:"新建人员 · uid=cuimin" },
  { row:"23", type:"err",   name:"李娜", dept:"技术研发部", emp:"E1002", detail:"工号已存在：与现有人员「李娜 / E1002」冲突，请核对是否同一人" },
  { row:"24", type:"update",name:"张伟", dept:"技术研发部", emp:"E1001", detail:"更新：手机 138-0101-2233 → 139-0101-2233" },
  { row:"25", type:"update",name:"陈静", dept:"技术研发部", emp:"E1005", detail:"更新：部门 人事行政部 → 技术研发部" },
  { row:"31", type:"err",   name:"（空）", dept:"—", emp:"—", detail:"邮箱列格式不正确：「fengxue#example.cn」" },
];
const PW_ERRS = {
  quality:  ["dgr","密码不符合策略","新密码强度不足：至少 8 位，且需同时包含字母和数字。这是服务器密码策略（ppolicy）的要求，与网站无关。"],
  history:  ["dgr","密码与历史重复","新密码与你最近 5 次使用过的密码重复，请换一个没有用过的。"],
  interval: ["warn","修改太频繁","距离上次修改密码未满 24 小时，暂时不能再次修改。如确有紧急情况，请联系管理员。"],
  must:     ["warn","需要先改密","管理员已重置你的密码。首次登录必须设置一个新密码后才能继续使用。"],
  bind:     ["dgr","当前密码不正确","「当前密码」验证失败（第 3 次，剩余 2 次尝试）。请确认输入的是修改前的旧密码。"],
  rate:     ["dgr","尝试次数过多","连续失败次数过多，此入口已临时锁定 10 分钟。这是为了防止有人反复尝试猜测密码。"],
};

/* ---------- 全局状态 ---------- */
const S = {
  view:"login", page:"people", ou:"ou=tech,"+ROOT, ouName:"技术研发部",
  expanded:new Set([ROOT,"ou=tech,"+ROOT]), mode:"simple",
  pFilter:"", pStatus:"", scope:"sub", expertPerson:null,
};

/* ---------- 视图 / 页面切换 ---------- */
function showView(v){
  S.view = v;
  document.body.dataset.view = v;
  if(v.startsWith("admin-")){
    document.body.dataset.view = "admin";
    const page = v.slice(6) || "people";
    showPage(page);
  }
  refreshIcons();
}
function showPage(p){
  S.page = p;
  $$("#mainNav .item").forEach(n => n.classList.toggle("on", n.dataset.page===p));
  $$(".page").forEach(pg => pg.classList.toggle("on", pg.id==="page-"+p));
  $$("#modeSeg button").forEach(b => b.classList.toggle("on", (b.dataset.mode==="expert")===(p==="expert")));
  refreshIcons();
}

/* ---------- 目录树 ---------- */
function peopleInOu(dn, sub=true){
  return PEOPLE.filter(p => p.ou===dn || (sub && p.ou.startsWith(dn+",")));
}
function buildTree(node, depth){
  const hasKids = node.children && node.children.length;
  const exp = S.expanded.has(node.dn);
  const cnt = node.dn===ROOT ? "" : peopleInOu(node.dn,true).length;
  let h = `<div class="node ${exp?"exp":""} ${S.ou===node.dn?"on":""}" data-dn="${node.dn}" data-name="${node.name}" style="padding-left:${4+depth*14}px">
    <span class="caret ${hasKids?"":"leaf"}">${hasKids?ic("chevron-right"):ic("chevron-right")}</span>
    <span class="nicon">${ic(node.icon,"ic")}</span><span>${node.name}</span>
    ${cnt!==""?`<span class="cnt">${cnt}</span>`:""}</div>`;
  if(hasKids){
    h += `<div class="tree-kids" style="${exp?"":"display:none"}">`;
    for(const k of node.children) h += buildTree(k, depth+1);
    h += `</div>`;
  }
  return h;
}
function renderTree(){
  $("#tree").innerHTML = buildTree(TREE,0);
  $$("#tree .node").forEach(n => n.addEventListener("click", e => {
    const dn = n.dataset.dn;
    if(e.target.closest(".caret") && n.nextElementSibling){ S.expanded.has(dn) ? S.expanded.delete(dn) : S.expanded.add(dn); renderTree(); return; }
    S.ou = dn; S.ouName = n.dataset.name;
    if(S.mode==="expert"){ showPage("expert"); renderExpert(); }
    else { showPage("people"); }
    renderTree(); renderPeople();
  }));
  refreshIcons();
}
function findOu(dn, node=TREE){
  if(node.dn===dn) return node;
  for(const k of (node.children||[])){ const r=findOu(dn,k); if(r) return r; }
  return null;
}

/* ---------- 人员 ---------- */
function renderPeople(){
  const list = peopleInOu(S.ou, S.scope==="sub").filter(p =>
    (!S.pStatus || p.st===S.pStatus) &&
    (!S.pFilter || [p.name,p.uid,p.emp,p.mail].join(" ").toLowerCase().includes(S.pFilter.toLowerCase())));
  $("#ouTitle").textContent = S.ouName;
  $("#ouDn").textContent = S.ou;
  $("#ouCount").textContent = `${peopleInOu(S.ou,false).length} 名人员`;
  $("#pTotal").textContent = `共 ${list.length} 人`;
  $("#pgInfo").textContent = `第 1-${list.length} 条 · 共 ${list.length} 条`;
  const body = $("#peopleBody");
  if(!list.length){
    body.innerHTML = `<tr><td colspan="7"><div class="empty">${ic("users","ic")}<div class="t">这里还没有人员</div><div style="font-size:12px">在左侧选中这个部门后，点右上角「新建人员」，或用 Excel 批量导入</div></div></td></tr>`;
  } else {
    body.innerHTML = list.map(p=>`<tr class="clickable" data-uid="${p.uid}">
      <td><div style="display:flex;align-items:center;gap:10px"><span class="avatar ${p.st==="disabled"?"gray":""}">${p.name[0]}</span><div><div>${p.name}</div><div class="muted" style="font-size:11.5px">${p.title||""}${p.posix?" · <span class=\'term\' data-tip=\'posixAccount：附加后可登录 Linux 主机，自动分配 uidNumber。不需要的员工不要开启。\'>Linux 登录</span>":""}</div></div></div></td>
      <td class="mono">${p.uid}</td><td class="mono">${p.emp}</td><td class="mono" style="font-size:12px">${p.mail}</td><td class="mono">${p.phone}</td>
      <td>${p.st==="active"?'<span class="badge ok"><span class="dot"></span>在职</span>':'<span class="badge plain"><span class="dot"></span>已禁用</span>'}</td>
      <td><div class="ops">
        <button class="btn ghost icon-btn" data-op="edit" title="编辑">${ic("pencil-line")}</button>
        <button class="btn ghost icon-btn" data-op="pw" title="重置密码">${ic("key-round")}</button>
        <button class="btn ghost icon-btn" data-op="more" title="更多">${ic("ellipsis")}</button>
      </div></td></tr>`).join("");
    $$("#peopleBody tr").forEach(tr=>{
      tr.addEventListener("click", e=>{
        const p = PEOPLE.find(x=>x.uid===tr.dataset.uid);
        const op = e.target.closest("[data-op]");
        if(op && op.dataset.op==="edit"){ openPerson(p); return; }
        if(op && op.dataset.op==="pw"){ openPerson(p,"pw"); return; }
        if(op && op.dataset.op==="more"){ personMenu(op, p); return; }
        openPerson(p);
      });
    });
  }
  refreshIcons();
}

/* ---------- 用户组 ---------- */
function renderGroups(){
  const f = ($("#gFilter").value||"").toLowerCase();
  const seg = $("#gTypeSeg .on").textContent.trim();
  const list = GROUPS.filter(g=>(!f||g.name.toLowerCase().includes(f))&&(seg==="全部"||(seg==="权限组")===(g.type==="auth")));
  $("#gTotal").textContent = `共 ${list.length} 个组`;
  $("#groupsBody").innerHTML = list.map(g=>`<tr class="clickable" data-dn="${g.dn}">
    <td><b>${g.name}</b><div class="muted mono" style="font-size:11px">${g.dn}</div></td>
    <td>${g.type==="auth"
      ? '<span class="badge info"><i data-lucide="shield" class="ic ic-14"></i>权限组</span><div class="muted" style="font-size:11px;margin-top:3px">'+g.oc+'</div>'
      : '<span class="badge plain"><i data-lucide="server" class="ic ic-14"></i>登录组</span><div class="muted" style="font-size:11px;margin-top:3px">'+g.oc+'</div>'}</td>
    <td style="color:var(--t2)">${g.desc}</td>
    <td><b>${g.members.length}</b> <span class="muted">名成员</span></td>
    <td><div class="ops">
      <button class="btn ghost icon-btn" data-op="edit">${ic("pencil-line")}</button>
      <button class="btn ghost icon-btn del" data-op="del">${ic("trash-2")}</button>
    </div></td></tr>`).join("");
  $$("#groupsBody tr").forEach(tr=>{
    tr.addEventListener("click", e=>{
      const g = GROUPS.find(x=>x.dn===tr.dataset.dn);
      const op = e.target.closest("[data-op]");
      if(op && op.dataset.op==="del"){ toast(`已请求删除「${g.name}」——正式版会先展示成员影响并要求二次确认（原型演示）`,"warn"); return; }
      openGroup(g);
    });
  });
  refreshIcons();
}

/* ---------- 抽屉 ---------- */
function openDrawer(html){ $("#drawer").innerHTML = html; document.body.classList.add("drawer-open"); refreshIcons(); }
function closeDrawer(){ document.body.classList.remove("drawer-open"); }

function ouOptions(sel){
  const walk=(n,path)=>{ let r=[]; const pp = n.dn===ROOT?n.name:path+" / "+n.name;
    if(n.dn!==ROOT && !n.dn.startsWith("ou=groups")) r.push(`<option ${n.dn===sel?"selected":""} value="${n.dn}">${pp}</option>`);
    for(const k of (n.children||[])) r=r.concat(walk(k,pp)); return r; };
  return walk(TREE,"").join("");
}
function openPerson(p, focus){
  S.expertPerson = p;
  openDrawer(`
  <div class="d-h"><div class="t"><span class="avatar lg">${p.name[0]}</span><div>${p.name}
    ${p.st==="active"?'<span class="badge ok" style="margin-left:6px"><span class="dot"></span>在职</span>':'<span class="badge plain" style="margin-left:6px"><span class="dot"></span>已禁用</span>'}
    <div class="muted mono" style="font-size:11px;font-weight:400;margin-top:3px">uid=${p.uid},${p.ou}</div></div></div>
    <button class="btn ghost icon-btn" onclick="closeDrawer()">${ic("x")}</button></div>
  <div class="d-b">
    <div class="sec-t">基本信息</div>
    <div class="form-grid">
      <div class="field"><label>姓名<span class="req">*</span></label><input class="input" value="${p.name}"></div>
      <div class="field"><label>账号（<span class="term" data-tip="uid：登录用的账号名，创建后不再变化，类似工牌号。">uid</span>）</label><input class="input mono" value="${p.uid}" readonly style="background:#FAFBFC"></div>
      <div class="field"><label>工号</label><input class="input mono" value="${p.emp}"></div>
      <div class="field"><label>职务</label><input class="input" value="${p.title||""}"></div>
      <div class="field"><label>邮箱</label><input class="input mono" value="${p.mail}" style="font-size:12px"></div>
      <div class="field"><label>手机</label><input class="input mono" value="${p.phone}"></div>
      <div class="field full"><label>所属部门（<span class="term" data-tip="OU（Organizational Unit）：目录树中的“部门”节点，可以多层嵌套，像公司的组织架构。">OU</span>）</label>
        <select class="select">${ouOptions(p.ou)}</select><span class="hint">修改部门 = 在目录树中移动该人员（modrdn），历史属性全部保留。</span></div>
    </div>
    <div class="sec-t">账号与安全</div>
    <div style="display:flex;flex-direction:column;gap:14px">
      <div style="display:flex;align-items:center;gap:10px">
        <button class="switch ${p.posix?"on":""}" id="posixSw"></button>
        <div><b style="font-size:13px">Linux 主机登录</b><div class="muted" style="font-size:12px;margin-top:2px">开启后附加 posixAccount，自动分配 uidNumber（当前 ${p.posix?"5001":"—"}）</div></div>
      </div>
      ${focus==="pw"||true?`
      <div style="border:1px solid var(--line-2);border-radius:10px;padding:12px 14px">
        <div style="display:flex;align-items:center;gap:10px;margin-bottom:8px">${ic("key-round","ic ic-18")}<b style="font-size:13px">重置密码</b></div>
        <div style="display:flex;gap:8px;flex-wrap:wrap">
          <input class="input mono" style="flex:1;min-width:170px" id="resetPw" value="Xk9!mQ2#vL7p" readonly>
          <button class="btn" data-act="genpw">${ic("refresh-cw")}重新生成</button>
          <button class="btn" data-act="copy-pw">${ic("copy")}复制</button>
        </div>
        <label class="check on" style="margin-top:10px" id="forceChg"><span class="box">${ic("check","ic ic-14")}</span>员工下次登录时必须修改密码（ppolicy）</label>
      </div>`:""}
    </div>
    <div class="sec-t">原始属性</div>
    <div class="alert info">${ic("info")}<div>该条目共 14 个属性（含 <span class="term" data-tip="objectClass：条目的“类型模板”，决定它有哪些字段。人员默认是 inetOrgPerson。">objectClass</span> 5 个）。<a class="link" data-act="to-expert">切换专家模式查看全部 →</a></div></div>
  </div>
  <div class="d-f">
    <button class="btn danger sm" data-act="disable">${p.st==="active"?ic("power")+ "禁用账号":ic("power")+ "启用账号"}</button>
    <span style="flex:1"></span>
    <span class="muted" style="font-size:11px">${ic("history","ic ic-14")} 版本 10:32 · <a class="link" data-demo-conflict style="font-size:11px">原型：模拟他人已修改</a></span>
    <button class="btn" onclick="closeDrawer()">取消</button>
    <button class="btn primary" data-act="save-person">保存修改</button>
  </div>`);
}
function openNewPerson(){
  openDrawer(`
  <div class="d-h"><div class="t">${ic("user-plus","ic ic-18")}新建人员<span class="badge plain">${S.ouName}</span></div>
    <button class="btn ghost icon-btn" onclick="closeDrawer()">${ic("x")}</button></div>
  <div class="d-b">
    <div class="form-grid">
      <div class="field"><label>姓名<span class="req">*</span></label><input class="input" placeholder="如：张伟"></div>
      <div class="field"><label>账号 uid<span class="req">*</span></label><input class="input mono" placeholder="如：zhangwei"><span class="hint">将创建于 uid=…,${S.ou}</span></div>
      <div class="field"><label>工号</label><input class="input mono" placeholder="如：E1001"><span class="hint">导入与匹配的唯一标识</span></div>
      <div class="field"><label>职务</label><input class="input" placeholder="选填"></div>
      <div class="field"><label>邮箱</label><input class="input mono" placeholder="选填"></div>
      <div class="field"><label>手机</label><input class="input mono" placeholder="选填"></div>
    </div>
    <div class="sec-t">初始密码</div>
    <div style="display:flex;gap:8px;flex-wrap:wrap">
      <input class="input mono" style="flex:1;min-width:170px" value="Xk9!mQ2#vL7p" readonly>
      <button class="btn" data-act="genpw">${ic("refresh-cw")}重新生成</button>
    </div>
    <label class="check on" style="margin-top:10px"><span class="box">${ic("check","ic ic-14")}</span>首次登录必须修改密码</label>
    <div class="alert info" style="margin-top:14px">${ic("lightbulb")}<div>不需要懂 objectClass：工具按所选模板自动写入 <b>inetOrgPerson</b> 等类型，专家模式可全程自定义。</div></div>
  </div>
  <div class="d-f"><span style="flex:1"></span><button class="btn" onclick="closeDrawer()">取消</button>
    <button class="btn primary" data-act="save-new">${ic("check")}创建</button></div>`);
}
function openGroup(g){
  const members = g.members.map(uid=>PEOPLE.find(p=>p.uid===uid)).filter(Boolean);
  openDrawer(`
  <div class="d-h"><div class="t">${ic("users-round","ic ic-20")}<div>${g.name}
      <div class="muted mono" style="font-size:11px;font-weight:400;margin-top:3px">${g.dn}</div></div></div>
    <button class="btn ghost icon-btn" onclick="closeDrawer()">${ic("x")}</button></div>
  <div class="d-b">
    <div class="form-grid">
      <div class="field"><label>组名<span class="req">*</span></label><input class="input" value="${g.name}"></div>
      <div class="field"><label>类型</label><input class="input" value="${g.oc}" readonly style="background:#FAFBFC"><span class="hint">${g.type==="auth"?"权限组：成员引用 DN，用于业务授权":"登录组：含 gidNumber，用于 Linux 登录"}</span></div>
      <div class="field full"><label>说明</label><input class="input" value="${g.desc}"></div>
    </div>
    <div class="sec-t">成员（${members.length}）</div>
    <div class="input-wrap" style="margin-bottom:12px"><span class="lead">${ic("search")}</span>
      <input class="input" placeholder="搜索并添加成员（姓名 / 账号）"></div>
    ${members.map(p=>`<div class="member"><span class="avatar ${p.st==="disabled"?"gray":""}">${p.name[0]}</span>
      <div class="info"><b>${p.name} <span class="muted mono" style="font-weight:400;font-size:11px">${p.uid} · ${p.ouName}</span></b></div>
      <button class="rm">${ic("x")}</button></div>`).join("")}
    <div class="alert info" style="margin-top:6px">${ic("info")}<div>成员关系直接写入目录（member 属性），LAM 等客户端实时可见。人员详情页可反向查看其所属组。</div></div>
  </div>
  <div class="d-f"><span style="flex:1"></span><button class="btn" onclick="closeDrawer()">取消</button>
    <button class="btn primary" data-act="save-group">保存修改</button></div>`);
}

/* ---------- 专家模式 ---------- */
const SYNTAX = { cn:"directoryString", sn:"directoryString", uid:"directoryString", mail:"IA5 String", mobile:"telephoneNumber", employeeNumber:"directoryString", title:"directoryString", ou:"directoryString", uidNumber:"INTEGER", gidNumber:"INTEGER", homeDirectory:"directoryString", loginShell:"directoryString", userPassword:"八角串(加密)", description:"directoryString" };
function personAttrs(p){
  const a = [["cn",p.name],["sn",p.name[0]],["uid",p.uid],["employeeNumber",p.emp],["mail",p.mail],["mobile",p.phone],["title",p.title||"—"]];
  if(p.posix) a.push(["uidNumber","5001"],["gidNumber","5000 (dev-login)"],["homeDirectory",`/home/${p.uid}`],["loginShell","/bin/bash"]);
  return a;
}
function renderExpert(){
  const p = S.expertPerson || peopleInOu(S.ou,true)[0] || PEOPLE[0];
  S.expertPerson = p;
  const dns = p.ou+","+ROOT;
  const full = `uid=${p.uid},${p.ou}`;
  $("#expDn").innerHTML = full.split(",").reverse().map((c,i,arr)=>
    `<span class="seg-dn">${esc(c)}</span>${i<arr.length-1?'<span class="sep">,</span>':""}`).join("");
  const ocs = ["top","person","organizationalPerson","inetOrgPerson"].concat(p.posix?["posixAccount"]:[]);
  $("#expOc").innerHTML = `<span class="muted" style="font-size:12px;align-self:center">objectClass：</span>` +
    ocs.map(o=>`<span class="badge plain mono">${o}</span>`).join("");
  const attrs = personAttrs(p);
  $("#expAttrCnt").textContent = attrs.length + " 个用户属性";
  $("#expAttrs").innerHTML = attrs.map(([k,v])=>`<tr><td class="mono" style="font-weight:600">${k}</td><td class="mono" style="font-size:12px">${esc(v)}</td><td class="muted mono" style="font-size:11px">${SYNTAX[k]||"directoryString"}</td></tr>`).join("")
    + `<tr><td class="mono" style="font-weight:600;color:var(--t2)">entryCSN</td><td class="mono" style="font-size:12px;color:var(--t2)">20260921073512.123456Z#000000#000#000000</td><td class="muted mono" style="font-size:11px">版本标记</td></tr>`;
  const ldif = [`dn: ${full}`, ...ocs.map(o=>`objectClass: ${o}`), ...attrs.map(([k,v])=>`${k}: ${v}`),
    "", "# —— 操作属性（只读）——", "structuralObjectClass: inetOrgPerson", "entryCSN: 20260921073512.123456Z#000000#000#000000",
    "modifiersName: cn=admin,dc=example,dc=cn", "modifyTimestamp: 20260921073512Z"].join("\n");
  $("#expLdif").innerHTML = ldif.split("\n").map(l=>{
    if(l.startsWith("#")) return `<span class="c">${esc(l)}</span>`;
    const m = l.match(/^([^:]+): (.*)$/);
    return m ? `<span class="k">${esc(m[1])}</span>: <span class="v">${esc(m[2])}</span>` : esc(l);
  }).join("\n");
  refreshIcons();
}

/* ---------- 弹出菜单 ---------- */
function openMenu(anchor, items){
  const m = $("#ctxMenu");
  m.innerHTML = items.map((it,i)=> it==="-" ? '<div class="sep"></div>' :
    `<div class="mi ${it.danger?"dgr":""}" data-i="${i}">${ic(it.icon)}<span>${it.label}</span></div>`).join("");
  m.classList.add("open");
  const r = anchor.getBoundingClientRect();
  m.style.top = Math.min(r.bottom+6, innerHeight-m.offsetHeight-10)+"px";
  m.style.left = Math.min(r.left, innerWidth-m.offsetWidth-10)+"px";
  refreshIcons();
  $$("#ctxMenu .mi").forEach(mi=>mi.addEventListener("click",()=>{
    const it = items[+mi.dataset.i]; closeMenu(); it.fn && it.fn();
  }));
}
function closeMenu(){ $("#ctxMenu").classList.remove("open"); }
document.addEventListener("click",e=>{ if(!e.target.closest("#ctxMenu")) closeMenu(); });

function personMenu(anchor,p){
  openMenu(anchor,[
    {icon:"arrow-right-left",label:"调动部门…",fn:()=>openPerson(p)},
    {icon:p.st==="active"?"power":"badge-check",label:p.st==="active"?"禁用账号":"启用账号",fn:()=>toast(`${p.name} 账号已${p.st==="active"?"禁用（登录将被拒绝）":"启用"}`)},
    {icon:"copy",label:"复制 DN",fn:()=>toast(`已复制：uid=${p.uid},${p.ou}`)},
    "-",
    {icon:"braces",label:"在专家模式中查看",fn:()=>{S.expertPerson=p;S.mode="expert";showPage("expert");renderExpert();}},
    {icon:"trash-2",label:"删除人员",danger:true,fn:()=>toast(`删除需输入条目名确认，且会展示影响范围（原型演示）`,"warn")},
  ]);
}
function ouMenu(anchor){
  openMenu(anchor,[
    {icon:"pencil-line",label:"重命名部门",fn:()=>toast("重命名 = 修改 RDN（modrdn），子部门与人员随之移动（原型演示）")},
    {icon:"arrow-right-left",label:"移动部门",fn:()=>toast("移动前会预览新 DN 与影响条目数（原型演示）")},
    {icon:"download",label:"导出本部门人员",fn:()=>toast("已开始导出（不含密码）")},
    "-",
    {icon:"trash-2",label:"删除部门",danger:true,fn:()=>toast("非空部门删除前会列出全部子孙条目并要求勾选「级联删除」（原型演示）","warn")},
  ]);
}

/* ---------- 向导（连接） ---------- */
let connectStep = 1;
function connectGo(n){
  connectStep = n;
  $$("#connectSteps .step").forEach((s,i)=>{ s.classList.toggle("cur",i===n-1); s.classList.toggle("done",i<n-1); });
  [1,2,3].forEach(i=>$("#connectP"+i).style.display = i===n?"":"none");
  $("#connectPrev").disabled = n===1;
  const next = $("#connectNext");
  if(n===1){ next.innerHTML = `测试连接${ic("arrow-right")}`; $("#connectHint").textContent="原型的下一步会模拟一次真实探测"; }
  if(n===2){ next.innerHTML = `下一步${ic("arrow-right")}`; $("#connectHint").textContent="";
    next.disabled = true;
    $("#probeLoading").style.display="flex"; $("#probeResult").style.display="none";
    setTimeout(()=>{ $("#probeLoading").style.display="none"; $("#probeResult").style.display="block"; next.disabled=false; refreshIcons(); }, 1400);
  }
  if(n===3){ next.innerHTML = `保存并继续${ic("arrow-right")}`; $("#connectHint").textContent=""; }
  refreshIcons();
}
$("#connectPrev").addEventListener("click",()=>connectGo(connectStep-1));
$("#connectNext").addEventListener("click",()=>connectStep<3 ? connectGo(connectStep+1) : (toast("连接档案已保存（加密）","ok","plug"), showView("setup")));
$$("#tlsSeg button").forEach(b=>b.addEventListener("click",()=>{ $$("#tlsSeg button").forEach(x=>x.classList.remove("on")); b.classList.add("on"); }));

/* ---------- 向导（建库） ---------- */
let setupStep = 1;
function setupGo(n){
  setupStep = n;
  $$("#setupSteps .step").forEach((s,i)=>{ s.classList.toggle("cur",i===n-1); s.classList.toggle("done",i<n-1); });
  [1,2,3,4].forEach(i=>$("#setupP"+i).style.display = i===n?"":"none");
  $("#setupPrev").disabled = n===1;
  $("#setupNext").innerHTML = n===4 ? `${ic("check")}进入管理台` : `下一步${ic("arrow-right")}`;
  refreshIcons();
}
$("#setupPrev").addEventListener("click",()=>setupGo(setupStep-1));
$("#setupNext").addEventListener("click",()=>setupStep<4 ? setupGo(setupStep+1) : (toast("目录初始化完成","ok","list-tree"), showView("admin-people")));
$$("#tplGrid .tpl").forEach(t=>t.addEventListener("click",()=>{ $$("#tplGrid .tpl").forEach(x=>x.classList.remove("on")); t.classList.add("on"); }));

/* ---------- 导入向导 ---------- */
let impStep = 1;
function impGo(n, skipAnim){
  impStep = n;
  $$("#impSteps .step").forEach((s,i)=>{ s.classList.toggle("cur",i===n-1); s.classList.toggle("done",i<n-1); });
  [1,2,3,4].forEach(i=>$("#impP"+i).style.display = i===n?"":"none");
  $("#impStepBadge").textContent = `第 ${n} 步 / 共 4 步`;
  $("#impPrev").disabled = n===1; $("#impPrev").style.visibility = n===4?"hidden":"visible";
  const next = $("#impNext");
  next.disabled = false;
  if(n===1) next.innerHTML = `下一步：上传文件${ic("arrow-right")}`;
  if(n===2) next.innerHTML = `解析并预览${ic("arrow-right")}`;
  if(n===3) next.innerHTML = `${ic("circle-check")}开始执行（25 项）`;
  if(n===4){ next.innerHTML = `${ic("check")}完成`;
    if(!skipAnim) runExec(); }
  refreshIcons();
}
function runExec(){
  const log = $("#execLog"), bar = $("#execBar"), pct = $("#execPct");
  $("#execArea").style.display=""; $("#reportArea").style.display="none";
  const lines = [
    "创建部门 ou=data,dc=example,dc=cn … ok",
    "创建部门 ou=algo,ou=data,… ok",
    "新建人员 uid=xuqian,ou=data,… ok",
    "新建人员 uid=dengchao,ou=algo,… ok",
    "新建人员 uid=cuimin,ou=data,… ok",
    "更新人员 uid=zhangwei（mobile）… ok",
    "更新人员 uid=chenjing（部门移动 modrdn）… ok",
    "跳过错误行 #23（工号冲突）",
    "跳过错误行 #31（邮箱格式）",
    "写入审计日志（批次 batch-20260921-a）… done",
  ];
  log.innerHTML=""; let i=0;
  const t = setInterval(()=>{
    if(i<lines.length){
      log.innerHTML += `<div>${esc(lines[i])}</div>`; log.scrollTop = log.scrollHeight;
      const p = Math.round((i+1)/lines.length*100); bar.style.width=p+"%"; pct.textContent=p+"%";
      i++;
    } else {
      clearInterval(t);
      setTimeout(()=>{ $("#execArea").style.display="none"; $("#reportArea").style.display=""; refreshIcons(); }, 500);
    }
  }, 420);
}
function renderPlan(){
  const T = { dept:["info","新建部门"], new:["ok","新建人员"], update:["warn","更新"], err:["dgr","错误"] };
  $("#planBody").innerHTML = PLAN.map(r=>`<tr style="${r.type==="err"?"background:var(--dgr-bg)":""}">
    <td class="mono">${r.row}</td>
    <td><span class="badge ${T[r.type][0]}">${T[r.type][1]}</span></td>
    <td>${r.name}</td><td style="color:var(--t2)">${r.dept}</td><td class="mono">${r.emp}</td>
    <td style="font-size:12px;${r.type==="err"?"color:var(--dgr)":"color:var(--t2)"}">${r.detail}</td></tr>`).join("");
}
$("#impPrev").addEventListener("click",()=>impGo(impStep-1,true));
$("#impNext").addEventListener("click",()=>{
  if(impStep===1) impGo(2);
  else if(impStep===2){ $("#fakeFile").style.display="flex"; setTimeout(()=>impGo(3),500); }
  else if(impStep===3) impGo(4);
  else { toast("导入会话已结束","ok","file-check-2"); impGo(1,true); }
});
$("#dropzone").addEventListener("click",()=>{ $("#fakeFile").style.display="flex"; toast("已选择模拟文件（原型不做真实上传）","ok","file-check-2"); });

/* ---------- 自助改密 ---------- */
const pwMeter = txt => {
  const v = $("#ssNew").value;
  let lv = 0; if(v.length>=8) lv++; if(/[a-zA-Z]/.test(v)&&/\d/.test(v)) lv++; if(/[^a-zA-Z0-9]/.test(v)&&v.length>=10) lv++;
  $$(".pw-meter i").forEach((el,i)=>{ el.className = i<lv ? "f"+lv : ""; });
  $("#pwMeterTxt").textContent = "密码强度：" + (["—","弱","中","强"][lv]||"—") + " · 至少 8 位，含字母和数字";
};
$("#ssNew").addEventListener("input", pwMeter);
function showPwErr(key){
  const [kind,title,txt] = PW_ERRS[key];
  $("#ssError").style.display="block";
  $("#ssError").innerHTML = `<div class="alert ${kind==="dgr"?"dgr":"warn"}">${ic(kind==="dgr"?"circle-alert":"triangle-alert")}<div><b>${title}。</b>${txt}</div></div>`;
  refreshIcons();
}
$$("[data-pw-err]").forEach(c=>c.addEventListener("click",()=>{
  const k = c.dataset.pwErr; showPwErr(k);
  if(k==="rate") toast("限流状态仅影响改密入口，管理员台不受影响（原型演示）","warn");
}));
$("#ssSubmit").addEventListener("click",()=>{
  const v = $("#ssNew").value;
  if(!v){ showPwErr("quality"); $("#ssNew").focus(); return; }
  $("#pwForm").style.display="none"; $("#pwDone").style.display="";
  refreshIcons();
});
$("#pwBack").addEventListener("click",()=>{ $("#pwDone").style.display="none"; $("#pwForm").style.display=""; $("#ssError").style.display="none"; });

/* ---------- 全局搜索 ---------- */
const sugBox = $("#searchSug");
$("#globalSearch").addEventListener("input", e=>{
  const q = e.target.value.trim().toLowerCase();
  if(!q){ sugBox.classList.remove("open"); return; }
  const ps = PEOPLE.filter(p=>[p.name,p.uid].join(" ").toLowerCase().includes(q)).slice(0,3);
  const gs = GROUPS.filter(g=>g.name.toLowerCase().includes(q)).slice(0,2);
  if(!ps.length && !gs.length){ sugBox.innerHTML = `<div class="cap">没有匹配的「${esc(q)}」</div>`; sugBox.classList.add("open"); return; }
  sugBox.innerHTML = (ps.length?`<div class="cap">人员</div>`+ps.map(p=>`<div class="it" data-uid="${p.uid}"><span class="avatar">${p.name[0]}</span><div><b>${p.name}</b> <span class="muted mono" style="font-size:11px">${p.uid} · ${p.ouName}</span></div></div>`).join(""):"")
    + (gs.length?`<div class="cap">用户组</div>`+gs.map(g=>`<div class="it" data-dn="${g.dn}">${ic("users-round")}<div><b>${g.name}</b> <span class="muted" style="font-size:11px">${g.desc}</span></div></div>`).join(""):"");
  sugBox.classList.add("open"); refreshIcons();
  $$("#searchSug .it").forEach(it=>it.addEventListener("click",()=>{
    sugBox.classList.remove("open"); $("#globalSearch").value="";
    if(it.dataset.uid){ const p=PEOPLE.find(x=>x.uid===it.dataset.uid); if(S.view!=="admin") showView("admin-people"); openPerson(p); }
    else { const g=GROUPS.find(x=>x.dn===it.dataset.dn); if(S.view!=="admin") showView("admin-groups"); openGroup(g); }
  }));
});
document.addEventListener("click",e=>{ if(!e.target.closest(".top-search")) sugBox.classList.remove("open"); });

/* ---------- 通用事件委托 ---------- */
document.addEventListener("click", e=>{
  const gt = e.target.closest("[data-goto]");
  if(gt){ showView(gt.dataset.goto); return; }
  const pg = e.target.closest("[data-page-go]");
  if(pg){ showPage(pg.dataset.pageGo); return; }
  const pw = e.target.closest("[data-toggle-pw]");
  if(pw){ const inp=$("#"+pw.dataset.togglePw); inp.type = inp.type==="password"?"text":"password";
    pw.innerHTML = ic(inp.type==="password"?"eye":"eye-off"); refreshIcons(); return; }
  const pe = e.target.closest("[data-pw-err]");
  const act = e.target.closest("[data-act]");
  if(act){
    const a = act.dataset.act;
    if(a==="new-person") openNewPerson();
    if(a==="export") toast("已开始导出 Excel（不含密码与哈希）","ok","download");
    if(a==="export-audit") toast("审计日志导出中…","ok","file-text");
    if(a==="refresh-audit"){ renderAudit(); toast("审计日志已刷新"); }
    if(a==="refresh-stats") toast("统计已刷新");
    if(a==="dl-tpl") toast("模板已开始下载：组织与人员导入模板.xlsx","ok","file-spreadsheet");
    if(a==="genpw"){ const inp = act.closest("div").querySelector("input"); if(inp){ inp.value = Math.random().toString(36).slice(2,8)+"!"+Math.random().toString(36).slice(2,5).toUpperCase(); toast("已生成随机强密码并复制到剪贴板"); } }
    if(a==="copy-pw") toast("密码已复制，请安全转交该员工");
    if(a==="save-person"){ closeDrawer(); toast("已保存对人员的修改 —— 属性级更新，未触碰其他字段（LAM 侧实时可见）"); }
    if(a==="save-new"){ closeDrawer(); toast("人员已创建于当前部门，首次登录须改密"); }
    if(a==="save-group"){ closeDrawer(); toast("用户组已保存"); }
    if(a==="disable"){ const p=S.expertPerson; closeDrawer(); toast(p?`${p.name} 账号状态已变更`:"账号状态已变更","warn"); }
    if(a==="to-expert"){ closeDrawer(); S.mode="expert"; showPage("expert"); renderExpert(); }
    if(a==="test-conn"){ toast("连接测试中…");
      setTimeout(()=>toast("连接正常 · LDAPS · 延迟 12ms","ok","search-check"),900); }
    if(a==="clear-pw") toast("本机保存的密码已清除，下次使用需重新输入","warn","eraser");
    if(a==="kill-session") toast("所有本机会话已注销","warn","log-out");
    if(a==="exp-new") toast("新建条目支持表单与粘贴 LDIF 两种方式（原型演示）","ok","plus");
    if(a==="exp-move") toast("移动 / 改名将执行 modrdn，执行前预览新 DN（原型演示）","ok","arrow-right-left");
    return;
  }
  const dc = e.target.closest("[data-demo-conflict]");
  if(dc){
    const box = $("#drawer .d-f");
    if(box && !$("#conflictAlert")){
      box.insertAdjacentHTML("beforebegin", `<div class="alert dgr" id="conflictAlert" style="margin:0 20px 12px">${ic("history")}<div><b>条目已被其他客户端修改。</b>uid=zhangwei 在你编辑期间被 LAM（10.8.0.22）修改过（entryCSN 不一致）。<a class="link" id="conflictRefresh">放弃我的修改并刷新</a></div></div>`);
      refreshIcons();
      $("#conflictRefresh").addEventListener("click",()=>{ $("#conflictAlert").remove(); toast("已加载最新版本，表单已刷新（你的其他修改不受影响）"); });
    }
    return;
  }
  if(e.target.closest("#drawerMask")) closeDrawer();
  if(e.target.closest("[data-demo=conflict]")){ showView("admin-people"); const p=PEOPLE[0]; setTimeout(()=>{ openPerson(p); const f=$("#drawer .d-f"); f && f.querySelector("[data-demo-conflict]").click(); },150); }
});
document.addEventListener("keydown",e=>{ if(e.key==="Escape"){ closeDrawer(); closeMenu(); sugBox.classList.remove("open"); } });

/* ---------- 侧栏 / 导航 / 模式 ---------- */
$("#sideToggle").addEventListener("click",()=>document.body.classList.toggle("side-collapsed"));
$("#treeRefresh").addEventListener("click",()=>{ renderTree(); toast("目录树已刷新（数据实时来自 LDAP）"); });
$("#treeNew").addEventListener("click",ev=>openMenu(ev.currentTarget,[
  {icon:"folder-plus",label:"新建部门（当前选中之下）",fn:()=>toast("新建部门 = 创建 organizationalUnit 条目（原型演示）")},
  {icon:"git-branch",label:"批量搭建子部门…",fn:()=>toast("推荐使用 Excel 导入一次性搭建（原型演示）","ok","file-spreadsheet")},
]));
$("#ouMenuBtn").addEventListener("click",ev=>ouMenu(ev.currentTarget));
$$("#mainNav .item").forEach(n=>n.addEventListener("click",()=>showPage(n.dataset.page)));
$$("#modeSeg button").forEach(b=>b.addEventListener("click",()=>{
  S.mode = b.dataset.mode;
  showPage(S.mode==="expert" ? "expert" : "people");
  if(S.mode==="expert") renderExpert();
}));
$("#helpBtn").addEventListener("click",ev=>openMenu(ev.currentTarget,[
  {icon:"circle-help",label:"DN 是什么？",fn:()=>toast("DN = 条目在目录树中的完整路径，像收快递的详细地址")},
  {icon:"circle-help",label:"OU 是什么？",fn:()=>toast("OU = 目录树里的“部门”节点，可多层嵌套")},
  {icon:"circle-help",label:"objectClass 是什么？",fn:()=>toast("objectClass = 条目的类型模板，决定它有哪些字段")},
  {icon:"book-open",label:"完整使用手册",fn:()=>toast("正式版将内置中文手册（原型占位）")},
]));
$("#userMenu").addEventListener("click",ev=>openMenu(ev.currentTarget,[
  {icon:"key-round",label:"修改我的密码",fn:()=>showView("password")},
  {icon:"settings",label:"连接设置",fn:()=>showPage("settings")},
  "-",
  {icon:"log-out",label:"退出登录",fn:()=>showView("login")},
]));
$("#gFilter").addEventListener("input",renderGroups);
$$("#gTypeSeg button").forEach(b=>b.addEventListener("click",()=>{ $$("#gTypeSeg button").forEach(x=>x.classList.remove("on")); b.classList.add("on"); renderGroups(); }));
$("#pFilter").addEventListener("input",e=>{ S.pFilter=e.target.value; renderPeople(); });
$("#pStatus").addEventListener("change",e=>{ S.pStatus=e.target.value; renderPeople(); });
$$("#pScope button").forEach(b=>b.addEventListener("click",()=>{ $$("#pScope button").forEach(x=>x.classList.remove("on")); b.classList.add("on"); S.scope=b.textContent.includes("含")?"sub":"one"; renderPeople(); }));
$("#aFilter").addEventListener("input",renderAudit);

/* ---------- 审计 ---------- */
function renderAudit(){
  const f = ($("#aFilter").value||"").toLowerCase();
  const list = AUDIT.filter(a=>!f||[a.who,a.dn,a.src,a.op].join(" ").toLowerCase().includes(f));
  const OP = {"修改属性":"pencil-line","重置密码":"key-round","登录":"log-out","导入批次":"file-spreadsheet","新建人员":"user-plus","删除部门":"trash-2"};
  $("#auditBody").innerHTML = list.map(a=>`<tr>
    <td class="mono">${a.ts}</td><td class="mono" style="font-size:11.5px">${a.who}</td><td class="mono" style="font-size:11.5px">${a.src}</td>
    <td><span class="audit-op">${ic(OP[a.op]||"circle-dot")}${a.op}</span></td>
    <td class="mono" style="font-size:11.5px;max-width:320px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap" title="${a.dn}">${a.dn}</td>
    <td>${a.res==="ok"?'<span class="badge ok">成功</span>':'<span class="badge dgr">失败</span>'}</td></tr>`).join("");
  refreshIcons();
}

/* ---------- 原型导航器 ---------- */
$("#pnFab").addEventListener("click",()=>document.body.classList.toggle("pn-closed"));
$("#ldifToggle").addEventListener("click",()=>toast("正式版支持在 LDIF 与表单间切换编辑（原型只读展示）","ok","braces"));

/* ---------- 初始化 ---------- */
(function init(){
  // 未验证图标名的安全替换
  $$('[data-lucide="log-in"]').forEach(el=>el.dataset.lucide="arrow-right");
  $$('[data-lucide="circle-alert"]').forEach(el=>el.dataset.lucide="triangle-alert");
  renderTree(); renderPeople(); renderGroups(); renderAudit(); renderPlan(); renderExpert();
  pwMeter();
  refreshIcons();
})();

/* ═══════════════ 帮助中心：新手引导 / LDAP 教学 / 对接指引 / 术语词典 ═══════════════ */
const HS = { tab:"onboard", tpl:"bastion", addr:"ldaps://ldap.example.cn:636",
  acct:null, pwHidden:false, test:null, lesson:"dn",
  glossTab:"attr", glossQ:"", tour:-1 };

/* ---- 路由增强：admin-help:<tab> 与 data-help-tab ---- */
const _showView0 = showView;
showView = function(v){
  if(typeof v==="string" && v.startsWith("admin-help")){
    const tab = v.split(":")[1];
    _showView0("admin-help");
    if(tab) helpTab(tab);
    return;
  }
  return _showView0(v);
};
document.addEventListener("click", e=>{
  const h = e.target.closest("[data-help-tab]");
  if(h && h.dataset.helpTab) helpTab(h.dataset.helpTab);
});

/* ---- 页签切换 ---- */
function helpTab(t){
  HS.tab = t;
  $$("#helpTabs > button").forEach(b=>b.classList.toggle("on", b.dataset.htab===t));
  $$(".htab-pane").forEach(p=>p.classList.toggle("on", p.id==="htab-"+t));
  refreshIcons();
}
$$("#helpTabs > button").forEach(b=>b.addEventListener("click",()=>helpTab(b.dataset.htab)));

/* ---- ① 新手引导：任务清单 ---- */
const OB = [
  { icon:"plug",            t:"连接目录并完成探测", d:"OpenLDAP 2.6.2 · LDAPS · 分页 / memberOf / ppolicy 均可用", done:true },
  { icon:"folder-tree",     t:"搭建组织结构",       d:"5 个一级分支 · 部门中文名与拖拽排序已生效", done:true },
  { icon:"users",           t:"导入部门与人员名单", d:"156 名人员 · Excel 批量导入完成", done:true },
  { icon:"users-round",     t:"创建用户组",         d:"3 个权限组 + 2 个登录组 · 权限组用于授权场景", done:true },
  { icon:"search",          t:"体验搜索与树定位",   d:"顶部搜索框输入姓名试试，点击结果直接打开人员详情", done:false },
  { icon:"plug-zap",        t:"对接第一个系统",     d:"堡垒机 / 零信任接入本目录该填什么，一键生成信息卡", done:false, go:"connect" },
];
function renderOb(){
  $("#obTasks").innerHTML = OB.map(o=>`
    <li class="${o.done?"done":""}">
      <span class="st ${o.done?"done":""}">${o.done?ic("check"):""}</span>
      <div><div class="tt">${o.t}</div><div class="desc">${o.d}</div>
      ${o.go?`<a class="link" data-help-tab="${o.go}" style="font-size:12px">去对接指引 →</a>`:""}</div>
    </li>`).join("");
  refreshIcons();
}

/* ---- ② LDAP 教学：课程内容 ---- */
function dnBreak(dn, labels){
  return `<div class="dn-breakdown">` + dn.split(",").map((s,i)=>{
    const kv=s.split("=");
    return `<span class="seg-dn" style="${i===0?"":(i===dn.split(",").length-1?"background:#EDF4FF":"")}">${esc(s)}<span class="lb">${esc(labels[i]||kv[0])}</span></span>${i<dn.split(",").length-1?'<span class="sep">,</span>':""}`;
  }).join("") + `</div>`;
}
const LESSONS = {
  dn: { icon:"list-tree", t:"目录、条目与 DN", sub:"用你自己目录里的一名真实员工来理解 LDAP 的“门牌地址”",
    html:()=>`
    <p>LDAP 目录是一棵<strong>树</strong>（叫 <span class="term" data-tip="DIT（Directory Information Tree）：目录信息树。LDAP 里所有数据组成的树形结构，像公司的组织架构图。">DIT</span>）。树上每个节点叫<strong>条目</strong>（entry），每个条目有一个全局唯一的 <span class="term" data-tip="DN（Distinguished Name）：条目在树中的完整路径，全局唯一，像完整的门牌地址。">DN</span>。拿本目录的员工「张伟」举例：</p>
    ${dnBreak("uid=zhangwei,ou=tech,dc=example,dc=cn", ["账号（RDN）","部门（组织单元）","域名第一段","域名第二段"])}
    <p>DN 从左往右读：<b>最左一段叫 <span class="term" data-tip="RDN（Relative Distinguished Name）：DN 的最左一段，只在同一父节点下需要唯一。像门牌号只在同一条街上唯一。">RDN</span></b>（这个人的账号），往右依次是它挂在哪（部门）、整棵树的根（公司域名）。所以“把员工调动到另一个部门”在 LDAP 里其实是<b>移动树节点</b>——DN 会整个变掉。</p>
    <h4>在哪里能看到</h4>
    <p>左侧目录树的每一行就是一个条目；点开任意人员的「属性」页签，第一行就是它的 DN。</p>
    <div class="where"><b>${ic("map-pin","ic ic-14")} 概念在 NewLDAP 里对应什么</b>目录树 = DIT 本身 · 人员详情页头部 = 该条目 DN · 「调动部门」= 移动条目（DN 改变，entryUUID 不变）</div>` },

  oc: { icon:"shapes", t:"objectClass 与 schema", sub:"条目的“模板”：决定了它有哪些字段、哪些必填",
    html:()=>`
    <p>每个条目都挂着一个或多个 <span class="term" data-tip="objectClass：条目的类型模板，由服务器 schema 定义。决定该条目拥有哪些属性、哪些必填（MUST）哪些可选（MAY）。">objectClass</span>（对象类），相当于“这个条目按哪个模板创建”。本目录的人员主类是 <span class="mono">inetOrgPerson</span>（互联网组织人员），它自带一套字段约定：</p>
    <table><tr><th>objectClass</th><th>角色</th><th>必填（MUST）</th><th>常用可选（MAY）</th></tr>
    <tr><td class="mono">inetOrgPerson</td><td>人员主类</td><td class="mono">cn、sn</td><td class="mono">uid、mail、mobile、title…</td></tr>
    <tr><td class="mono">posixAccount</td><td>辅助类 · 可登录 Linux</td><td class="mono">uid、uidNumber…</td><td class="mono">homeDirectory、loginShell</td></tr>
    <tr><td class="mono">organizationalUnit</td><td>部门（OU）</td><td class="mono">ou</td><td class="mono">description、ouOrder</td></tr>
    <tr><td class="mono">groupOfNames</td><td>权限组</td><td class="mono">cn、member</td><td class="mono">description</td></tr></table>
    <p><b>主类 vs 辅助类：</b>主类定“这是什么”（人是 inetOrgPerson）；辅助类是给它<b>追加能力</b>的——勾选「Linux 登录」就是给人员追加 posixAccount，自动分配 uidNumber。这就是为什么权限组删不掉最后一名成员：groupOfNames 的 member 是 MUST（必填）。</p>
    <h4>schema 是服务器定的</h4>
    <p>对象类清单不能随便发明，由 OpenLDAP 服务器的 schema（结构定义）决定。本管理台「对象类」页签里看到的都是这台服务器真实支持的。</p>
    <div class="where"><b>${ic("map-pin","ic ic-14")} 概念在 NewLDAP 里对应什么</b>人员编辑页「对象类」页签 = 该条目的主类 / 辅助类 · 新建人员表单 = 按 inetOrgPerson 的 MUST/MAY 自动生成</div>` },

  attr: { icon:"table-properties", t:"属性与多值", sub:"字段名是英文缩写，一个字段可以有多个值",
    html:()=>`
    <p>条目的数据就是一组<span class="term" data-tip="属性（Attribute）：条目上的一个字段，形如 mail=zhangwei@example.cn。属性名是固定英文，值可以有多个。">属性</span>（attribute）。属性名是 schema 定死的英文（改不了），值由你填。有两个新人最容易懵的点：</p>
    <h4>1. 一个属性可以有多个值</h4>
    <p>比如部门的 <span class="mono">ou</span> 属性：<b>第一个值</b>是英文标识（用在 DN 里，<span class="mono">ou=tech</span>），<b>第二个值</b>是中文显示名（技术研发部）。本管理台的树、部门选择器读的就是第二个值——这不是本工具发明的花样，是 LDAP 多值属性的标准用法。</p>
    <h4>2. 操作属性也是属性</h4>
    <p>服务器会自动维护一批“操作属性”：<span class="mono">createTimestamp</span>（创建时间）、<span class="mono">entryUUID</span>（永不变更的唯一 ID）、<span class="mono">entryCSN</span>（修改版本号）。它们不用你填，但对接系统时非常有用——entryUUID 是推荐给堡垒机 / 零信任当“唯一标识”的值，人员改名换部门它都不变。</p>
    <div class="where"><b>${ic("map-pin","ic ic-14")} 概念在 NewLDAP 里对应什么</b>人员详情「属性」页签 = 全部属性 · 「操作属性」独立页签 = 服务器维护的那批 · 术语词典 = 全部属性名中文对照</div>` },

  bind: { icon:"key-round", t:"Bind 与权限", sub:"LDAP 没有“登录后赋权”这回事：你是谁，就能做什么",
    html:()=>`
    <p><span class="term" data-tip="bind：用某个账号的 DN 和密码向 LDAP 服务器证明身份。之后的所有操作都以这个身份进行。">bind</span>（绑定）就是 LDAP 的“登录”：拿一个 DN + 密码向服务器证明身份。之后你能做什么，完全由服务器上的 <span class="term" data-tip="ACL（Access Control List）：服务器上的访问控制列表，规定哪个账号能读/写哪些子树。">ACL</span> 决定——<b>目录里没有额外的“管理员权限表”</b>。</p>
    <h4>这个设计带来两个本产品的关键行为</h4>
    <p>① 你在本管理台用 admin 登录，能做的 = admin 在 OpenLDAP 上的全部权限，本工具不额外设限也不额外放权；② 换一个只有读权限的账号登录，界面照常能用，只是保存会被服务器拒绝——错误信息会原样转译给你。</p>
    <h4>对接场景</h4>
    <p>堡垒机 / 零信任接入时填的「Bind DN」，就是给对方系统一个 LDAP 身份去查目录。所以绝不该把 admin 给出去——用「对接指引」里一键创建的<b>专用只读账号</b>。</p>
    <div class="where"><b>${ic("map-pin","ic ic-14")} 概念在 NewLDAP 里对应什么</b>登录页 = 真实 bind · 概览「目录状态」= 当前连接 · 对接指引第 3 步 = 创建低权限 bind 账号</div>` },

  filter: { icon:"funnel", t:"过滤器语法", sub:"对接表单里的「用户过滤器」就是它 —— 在下面的实验器里随便试",
    html:()=>`
    <p>LDAP 查询用 <span class="term" data-tip="过滤器（Search Filter）：形如 (objectClass=inetOrgPerson) 的查询条件表达式，括号必需。">过滤器</span>（filter）表达“要找哪些条目”。整个表达式<b>必须包在括号里</b>。语法一共就这么几个：</p>
    <table><tr><th>写法</th><th>含义</th><th>例子</th></tr>
    <tr><td class="mono">(属性=值)</td><td>等于</td><td class="mono">(uid=zhangwei)</td></tr>
    <tr><td class="mono">(属性=前*)</td><td>前缀匹配（* 是通配符）</td><td class="mono">(cn=张*)</td></tr>
    <tr><td class="mono">(属性=*)</td><td>有这个属性就行</td><td class="mono">(mail=*)</td></tr>
    <tr><td class="mono">(&#38;(a)(b))</td><td>并且 AND</td><td class="mono">(&#38;(objectClass=inetOrgPerson)(mail=*))</td></tr>
    <tr><td class="mono">(|(a)(b))</td><td>或者 OR</td><td class="mono">(|(objectClass=groupOfNames)(objectClass=posixGroup))</td></tr></table>
    <p>右边那个 OR 的例子，就是对接指引里「组过滤器」的原文。下面实验器跑的就是本目录 22 个条目的真实模拟（12 人 + 6 部门 + 5 组，含禁用账号）：</p>
    ${labHtml()}` },

  group: { icon:"users-round", t:"组的两种形态", sub:"权限组按 DN 授权，登录组按 uid 认账号",
    html:()=>`
    <p>本目录用两种组，服务不同的场景，别混：</p>
    <table><tr><th></th><th>权限组（groupOfNames）</th><th>登录组（posixGroup）</th></tr>
    <tr><td><b>用途</b></td><td>授权：VPN / 堡垒机 / 应用权限</td><td>Linux 主机登录资格</td></tr>
    <tr><td><b>成员字段</b></td><td class="mono">member = 完整 DN</td><td class="mono">memberUid = uid</td></tr>
    <tr><td><b>数字 ID</b></td><td>无</td><td class="mono">gidNumber（5000-5999 自动分配）</td></tr>
    <tr><td><b>成员能否为空</b></td><td>不能（schema 必填至少 1 名）</td><td>可以</td></tr></table>
    <p><b>为什么授权推荐用权限组：</b>member 存 DN，指向明确、不受重名影响；零信任 / 堡垒机读它做“谁有权用什么”。登录组是给 <span class="mono">sshd</span> 这类 Linux 组件看的，它们只认 uid 和 gidNumber。</p>
    <div class="where"><b>${ic("map-pin","ic ic-14")} 概念在 NewLDAP 里对应什么</b>用户组页 = 两类合并展示 · 新建组 = 先选场景（权限 / 登录）再填成员 · 对接指引「组映射」一节 = 对方系统该按哪个字段读成员</div>` },

  pitfall: { icon:"triangle-alert", t:"常见误区", sub:"四个新手几乎都会踩的坑",
    html:()=>`
    <div class="faq">
    <details open><summary>改 uid 不是“改个字段”，是移动条目</summary><div class="fa">uid 出现在 DN 里（<span class="mono">uid=zhangwei,ou=tech,…</span>）。改 uid = 改 RDN = LDAP 的 modrdn 操作，本工具会在保存时自动处理成移动。但如果对方系统把 uid 存了下来当外键，人员改账号后对方那边会“失联”——这也是对接时推荐用 <span class="mono">entryUUID</span> 当唯一标识的原因。</div></details>
    <details><summary>权限组删不掉最后一名成员，不是 bug</summary><div class="fa">groupOfNames 的 member 是 schema MUST（必填）。想清空一个权限组：先加新成员再移旧成员，或者直接删组。本工具保存时若被服务器拒绝，会把原因白话转译出来。</div></details>
    <details><summary>“禁用账号”≠ 删除条目</summary><div class="fa">禁用是把密码置为失效（无法再 bind），条目和所有属性原样保留——工号、邮箱、组员身份都在，随时可恢复启用。删除条目则是永久移出目录，两者不要混用。</div></details>
    <details><summary>删除部门前要先清空它</summary><div class="fa">LDAP 不允许删除有子节点的条目（树不能“断枝”）。部门下还有人或有子部门时删除会失败，先移走人员、删掉子部门。本工具的删除按钮会先检查并给出明细。</div></details>
    </div>` },
};
function labHtml(){
  return `
  <div class="lab">
    <div class="lab-h">
      <input class="input mono" id="labInput" value="(objectClass=inetOrgPerson)" placeholder="输入 LDAP 过滤器，如 (uid=zhangwei)">
      <button class="btn primary" id="labRun">${ic("play")}执行查询</button>
      <span class="muted" style="font-size:11.5px">模拟对 22 个条目执行子树搜索</span>
    </div>
    <div class="lab-chips">${LAB_CHIPS.map(([n,f])=>`<span class="lab-chip" data-f="${esc(f)}" title="${esc(f)}">${esc(n)}</span>`).join("")}</div>
    <div class="lab-out" id="labOut"></div>
    <div class="lab-err">
      <span class="muted" style="font-size:11px;align-self:center">${ic("flask-conical","ic ic-14")} 原型演示：</span>
      <span class="badge plain" data-lab-preset="bad1" style="cursor:pointer">括号不匹配</span>
      <span class="badge plain" data-lab-preset="bad2" style="cursor:pointer">缺等号</span>
      <span class="badge plain" data-lab-preset="ok1" style="cursor:pointer">组合条件</span>
    </div>
  </div>`;
}
const LAB_CHIPS = [
  ["所有人员","(objectClass=inetOrgPerson)"],
  ["姓张的","(cn=张*)"],
  ["有邮箱的人","(&(objectClass=inetOrgPerson)(mail=*))"],
  ["所有部门","(objectClass=organizationalUnit)"],
  ["所有组","(|(objectClass=groupOfNames)(objectClass=posixGroup))"],
  ["技术部的人","(ou=技术研发部)"],
];
/* 实验器数据：把原型模拟数据摊平成“条目 + 属性” */
const LAB_ENTRIES = (()=>{
  const out = [];
  const walk = n => {
    out.push({ dn:n.dn, kind:"部门", attrs:{ objectClass:["top","organizationalUnit"], ou:[n.dn.split(",")[0].slice(3), n.name], description:[n.name+"（组织单元）"] } });
    (n.children||[]).forEach(walk);
  };
  walk(TREE);
  PEOPLE.forEach(p=>out.push({ dn:`uid=${p.uid},${p.ou}`, kind:"人员", attrs:{
    objectClass:["top","inetOrgPerson"].concat(p.posix?["posixAccount"]:[]),
    cn:[p.name], sn:[p.name[0]], uid:[p.uid], mail:[p.mail], mobile:[p.phone],
    employeeNumber:[p.emp], title:[p.title], ou:[p.ou.split(",")[0].slice(3), p.ouName] } }));
  GROUPS.forEach(g=>out.push({ dn:g.dn, kind:g.oc==="posixGroup"?"登录组":"权限组", attrs:{
    objectClass:["top",g.oc], cn:[g.name], description:[g.desc],
    ...(g.oc==="posixGroup" ? { memberUid:g.members, gidNumber:[String(5000+GROUPS.indexOf(g))] } : { member:g.members.map(u=>`uid=${u},`+(PEOPLE.find(p=>p.uid===u)||{ou:""}).ou+","+ROOT) }) } }));
  return out;
})();
/* 极简 RFC2254 子集解析器：&/| 嵌套、=、通配 *、presence */
function parseFilter(s){
  let i = 0;
  const ws = ()=>{ while(i<s.length && s[i]===" ") i++; };
  function expr(){
    ws();
    if(s[i]!=="(") throw { why:"条件必须包在括号里" };
    i++; ws();
    if(s[i]==="&" || s[i]==="|"){
      const op=s[i]; i++;
      const kids=[]; ws();
      while(i<s.length && s[i]==="("){ kids.push(expr()); ws(); }
      if(s[i]!==")") throw { why:"括号不匹配" };
      i++;
      if(!kids.length) throw { why:"& 和 | 至少要包含一个子条件" };
      return { op, kids };
    }
    const a0=i;
    while(i<s.length && s[i]!=="=" && s[i]!==")" && s[i]!=="(") i++;
    if(s[i]!=="=") throw { why:"属性名后面要有等号（=）" };
    const attr=s.slice(a0,i).trim(); i++;
    const v0=i;
    while(i<s.length && s[i]!==")" && s[i]!=="(") i++;
    if(s[i]!==")") throw { why:"括号不匹配" };
    const val=s.slice(v0,i).trim(); i++;
    if(!attr) throw { why:"等号前要有属性名" };
    return { op:"=", attr, val };
  }
  const f = expr(); ws();
  if(i<s.length) throw { why:"过滤器末尾有多余字符" };
  return f;
}
function labMatch(e,f){
  if(f.op==="&") return f.kids.every(k=>labMatch(e,k));
  if(f.op==="|") return f.kids.some(k=>labMatch(e,k));
  const vals = e.attrs[f.attr] || [];
  if(f.val==="*") return vals.length>0;
  if(f.val.includes("*")){
    return vals.some(v=>{
      let s=v.toLowerCase();
      if(!f.val.startsWith("*") && !s.startsWith(f.val.split("*")[0].toLowerCase())) return false;
      if(!f.val.endsWith("*") && !s.endsWith(f.val.split("*").pop().toLowerCase())) return false;
      for(const p of f.val.split("*")){ if(!p) continue;
        const idx=s.indexOf(p.toLowerCase()); if(idx<0) return false; s=s.slice(idx+p.length); }
      return true;
    });
  }
  return vals.some(v=>v.toLowerCase()===f.val.toLowerCase());
}
function labRun(){
  const out=$("#labOut"); if(!out) return;
  const q=($("#labInput").value||"").trim();
  try{
    const f=parseFilter(q);
    const hits=LAB_ENTRIES.filter(e=>labMatch(e,f));
    out.innerHTML = hits.length
      ? `<div class="muted" style="margin-bottom:8px">${ic("circle-check","ic ic-14")} 语法正确 · 命中 <b>${hits.length}</b> / ${LAB_ENTRIES.length} 个条目${hits.length>10?"（仅显示前 10）":""}</div>
         <table class="tbl"><thead><tr><th>命中条目 DN</th><th style="width:90px">类型</th></tr></thead><tbody>
         ${hits.slice(0,10).map(h=>`<tr><td class="mono" style="font-size:11.5px">${esc(h.dn)}</td><td><span class="badge plain">${h.kind}</span></td></tr>`).join("")}</tbody></table>`
      : `<div class="alert warn">${ic("search-x","ic")}<div><b>语法正确，但没有命中条目。</b>真实 LDAP 也会这样返回空列表——对接系统“搜不到用户”多半就是过滤器或 Base DN 范围的问题，不是连不上。</div></div>`;
  }catch(err){
    out.innerHTML = `<div class="alert dgr">${ic("triangle-alert","ic")}<div><b>服务器将返回错误 21（Bad search filter）</b><br>白话解释：${esc(err.why)}。返回对方产品配置页改过滤器时，这类错误就长这样——把错误和白话对照着看，改起来不慌。</div></div>`;
  }
  refreshIcons();
}
function renderLesson(){
  const L=LESSONS[HS.lesson];
  $("#lessonBody").innerHTML = `<h3>${ic(L.icon,"ic ic-18")}${L.t}</h3><div class="lsub">${L.sub}</div>` + L.html();
  $$(".lesson-nav .lesson-item").forEach(el=>el.classList.toggle("on", el.dataset.lesson===HS.lesson));
  const inp=$("#labInput");
  if(inp){
    inp.addEventListener("keydown",e=>{ if(e.key==="Enter") labRun(); });
    $("#labRun").addEventListener("click",labRun);
    $$("#labChips .lab-chip").forEach(c=>c.addEventListener("click",()=>{ inp.value=c.dataset.f; labRun(); }));
    $$("[data-lab-preset]").forEach(b=>b.addEventListener("click",()=>{
      const m={bad1:"(&(objectClass=inetOrgPerson)",bad2:"(uid zhangwei)",ok1:"(&(objectClass=inetOrgPerson)(|(cn=李*)(cn=王*)))"}[b.dataset.labPreset];
      inp.value=m; labRun();
    }));
    labRun();
  }
  refreshIcons();
}
$$(".lesson-nav .lesson-item").forEach(el=>el.addEventListener("click",()=>{ HS.lesson=el.dataset.lesson; renderLesson(); }));

/* ---- ③ 对接指引 ---- */
const TPLS = {
  bastion: { icon:"server", name:"堡垒机", full:"JumpServer / 齐治等", desc:"运维审计系统，按用户与用户组授权资产", acct:"jumphub", org:true, grp:true,
    tip:"JumpServer「系统设置 → 认证 → LDAP」逐项对照：LDAP 服务器 / Base DN / 绑定 DN 用下方取值；用户属性映射——用户名=uid、姓名=cn、邮箱=mail；用户树与用户过滤器见「用户映射」一节。资产授权建议用「权限组」。" },
  zero:    { icon:"shield-check", name:"零信任 / SDP", full:"深信服 aTrust / SDP 网关", desc:"身份为准入依据，组映射权限策略", acct:"sdp-reader", org:true, grp:true,
    tip:"唯一标识强烈建议用 entryUUID 而非 uid：员工改名、换部门后身份不丢，授权关系保持连续。组授权用「权限组」（groupOfNames），其成员是完整 DN，与零信任的“身份组”模型天然对齐。" },
  vpn:     { icon:"network", name:"VPN / 网络设备", full:"防火墙 / 交换机 / SSL VPN", desc:"设备换 LDAP 认证，通常仅简单 bind + 单个过滤器", acct:"vpn-auth", org:false, grp:false,
    tip:"多数网络设备只支持「服务器 + Base DN + Bind DN + 一个用户过滤器」，不支持组织架构与组映射——只填「连接」和「用户映射」两节即可。部分老设备不支持 LDAPS，需在目录侧放开明文 389 端口（限内网可达）。" },
  sso:     { icon:"fingerprint", name:"单点登录", full:"OIDC / SAML 门户的 LDAP 认证源", desc:"门户登录的身份源", acct:"sso-bind", org:false, grp:true,
    tip:"两种接法：「LDAP 直连认证」每次登录实时向目录验证（推荐，密码改后即时生效）；「全量同步到本地」门户首启快，但从此两套账号要人工对齐。选直连时无需组织架构映射。" },
  generic: { icon:"plug", name:"通用 LDAP 客户端", full:"Nginx / Jenkins / GitLab / Confluence…", desc:"任何支持 LDAP 认证的软件", acct:"app-readonly", org:true, grp:true,
    tip:"这一档是全字段对照表：对方配置表单里出现的每个 LDAP 字段，下方信息卡都有对应取值。高级参数（超时、分页大小、检索深度）保持对方默认即可，无需修改。" },
};
function renderTpls(){
  $("#connTpls").innerHTML = Object.entries(TPLS).map(([k,t])=>`
    <div class="tpl5 ${HS.tpl===k?"on":""}" data-tpl="${k}">
      <i data-lucide="${t.icon}" class="ic ic-20"></i>
      <b>${t.name}</b><p>${t.desc}<br><span class="muted">${t.full}</span></p>
    </div>`).join("");
  $$("#connTpls .tpl5").forEach(el=>el.addEventListener("click",()=>{ HS.tpl=el.dataset.tpl; HS.acct=null; HS.test=null; renderTpls(); renderConn(); renderBind(); }));
  refreshIcons();
}
function connRows(){
  const t=TPLS[HS.tpl], addr=HS.addr||"—", bindDN=HS.acct?HS.acct.dn:null;
  const sec=(icon,title,note,rows)=>({icon,title,note,rows:rows.filter(Boolean)});
  return [
    sec("plug","连接 · 填在对方的「服务器 / 地址」处","地址是目录服务器，不是本管理台", [
      { k:"服务器地址", tip:"对方产品能访问到的 OpenLDAP 地址。ldaps:// 开头=加密 636，ldap://=明文 389。", v:addr, copy:addr!=="—" },
      { k:"Base DN / 搜索根", tip:"搜索从这里开始。填本目录的根，用户和组都能搜到。", v:ROOT, copy:true },
      { k:"加密方式", plain:true, v:addr.startsWith("ldaps")?"LDAPS（636，推荐）":"明文 389（建议改用 LDAPS）", tip:"加密方式取决于上面的地址前缀，两边要保持一致。" },
    ]),
    sec("key-round","认证 · 填在对方的「Bind DN / 绑定账号」处","用第 3 步创建的只读账号，不要用 admin", [
      { k:"Bind DN", tip:"对方系统用来登录目录查数据的账号。", v:bindDN||"（先在第 3 步创建只读账号）", copy:!!bindDN, ghost:!bindDN },
      { k:"密码", tip:"账号的密码。只在创建那一刻完整显示过，忘了就重置。", v:HS.acct?(HS.pwHidden?"••••••••（已隐藏）":HS.acct.pw):"（创建只读账号后生成）", copy:!!HS.acct&&!HS.pwHidden, ghost:!HS.acct },
    ]),
    sec("user","用户映射 · 告诉对方“人员长什么样”","「用户筛选器」+ 属性名对照", [
      { k:"用户筛选器", tip:"只把人员条目筛出来，部门和组不会混入。", v:"(objectClass=inetOrgPerson)", copy:true },
      { k:"用户名属性（登录用）", tip:"员工用什么登录对方系统。本目录是 uid（如 zhangwei），不是邮箱。", v:"uid", copy:true },
      { k:"唯一标识", tip:"建议 entryUUID：人员改名 / 换部门它都不变，对方不会重复建号。", v:"entryUUID", copy:true },
      { k:"姓名 / 姓 / 邮箱", tip:"姓名 cn、姓 sn、邮箱 mail，一般对方默认就是这三个。", v:"cn · sn · mail", copy:true },
      { k:"部门 / 工号 / 手机", v:"ou · employeeNumber · mobile", copy:true, tip:"ou 的第二个值是部门中文名（如 技术研发部）。" },
    ]),
    t.org ? sec("folder-tree","组织架构映射 · 告诉对方“部门树长什么样”", TPLS[HS.tpl].name+"支持组织同步，可按此填",[
      { k:"组织筛选器", tip:"把部门条目筛出来。", v:"(objectClass=organizationalUnit)", copy:true },
      { k:"部门名属性", tip:"第一个 ou 值是英文标识，第二个是中文名；对方若只有一个字段，建议用中文名。", v:"ou（多值：标识 + 中文名）", copy:true },
      { k:"部门说明", v:"description", copy:true, tip:"部门描述字段，本管理台「编辑部门信息」里维护。" },
    ]) : null,
    t.grp ? sec("users-round","用户组映射 · 告诉对方“组怎么读”","权限组与登录组的成员字段不同，注意区分",[
      { k:"组筛选器", tip:"同时筛出权限组和登录组；只想要权限组就去掉 posixGroup 那段。", v:"(|(objectClass=groupOfNames)(objectClass=posixGroup))", copy:true },
      { k:"组名属性", v:"cn", copy:true, tip:"组的名称字段。" },
      { k:"成员属性（权限组）", tip:"groupOfNames 的 member 存完整 DN，适合做授权判断。", v:"member（值为人员 DN）", copy:true },
      { k:"成员属性（登录组）", tip:"posixGroup 的 memberUid 存 uid，Linux 场景用。", v:"memberUid（值为 uid）", copy:true },
    ]) : null,
  ].filter(Boolean);
}
function renderConn(){
  const t=TPLS[HS.tpl];
  $("#connTplName").textContent=`当前模板：${t.name}（${t.full}）`;
  $("#connCard").innerHTML = `
    <div class="alert info" style="margin-bottom:14px">${ic("lightbulb","ic")}<div>${esc(t.tip)}</div></div>
    <div class="icard">${connRows().map(s=>`
      <div class="sec">
        <h5>${ic(s.icon,"ic ic-14")}${s.title}<span class="m">${s.note}</span></h5>
        ${s.rows.map(r=>`
          <div class="row">
            <span class="k"><span class="term" data-tip="${esc(r.tip||r.k)}">${r.k}</span></span>
            <span class="v ${r.plain?"plain":""} ${r.ghost?"ghost":""}">${esc(r.v)}</span>
            ${r.copy?`<button class="copy-btn" data-copy="${esc(r.copy===true?r.v:r.copy)}">${ic("copy","ic ic-14")}复制</button>`:""}
          </div>`).join("")}
      </div>`).join("")}
    </div>
    <div class="muted" style="font-size:11.5px;margin-top:12px">${ic("info","ic ic-14")} 以上取值实时来自当前目录与连接配置。目录里改了（如新建了账号），这里刷新即变——本工具不保存任何“对接配置副本”。</div>`;
  bindCopy($$("#connCard .copy-btn"));
  refreshIcons();
}
function bindCopy(btns){
  btns.forEach(b=>b.addEventListener("click",async ()=>{
    const txt=b.dataset.copy;
    try{ await navigator.clipboard.writeText(txt); }
    catch{ const ta=document.createElement("textarea"); ta.value=txt; document.body.appendChild(ta); ta.select(); document.execCommand("copy"); ta.remove(); }
    b.classList.add("ok"); b.innerHTML=ic("check","ic ic-14")+"已复制"; refreshIcons();
    toast("已复制：" + (txt.length>46 ? txt.slice(0,46)+"…" : txt));
    setTimeout(()=>{ b.classList.remove("ok"); b.innerHTML=ic("copy","ic ic-14")+"复制"; refreshIcons(); },1600);
  }));
}
$("#connCopyAll") && $("#connCopyAll").addEventListener("click",()=>{
  const lines=["# NewLDAP 对接信息（" + TPLS[HS.tpl].name + "）",""];
  connRows().forEach(s=>{ lines.push("── " + s.title + " ──"); s.rows.forEach(r=>lines.push(r.k + "：" + r.v)); });
  const txt=lines.join("\n");
  navigator.clipboard && navigator.clipboard.writeText(txt).catch(()=>{});
  toast("全部对接信息已复制，可直接粘贴到工单 / 发给对方管理员","ok","clipboard-check");
});
$("#connAddr") && $("#connAddr").addEventListener("change",e=>{ HS.addr=e.target.value.trim(); renderConn(); });

/* 只读账号 */
function randPw(){
  const s="abcdefghjkmnpqrstuvwxyz", S="ABCDEFGHJKMNPQRSTUVWXYZ", d="23456789";
  const pick=(set,n)=>Array.from({length:n},()=>set[Math.floor(Math.random()*set.length)]).join("");
  return pick(S,2)+pick(d,4)+pick(s,4)+"!"+pick(d,2);
}
const OU_OPTS = (()=>{ const out=[]; (function w(n,depth){ if(n.dn!==ROOT) out.push({dn:n.dn,name:n.name,depth}); (n.children||[]).forEach(c=>w(c,depth+1)); })(TREE,0); return out; })();
const SVC_OU = "ou=services,"+ROOT;
function acctPreviewDn(){
  const name=($("#acctName")?.value||"").trim()||TPLS[HS.tpl].acct;
  const parent=$("#acctParent")?.value || HS.parent || SVC_OU;
  return "cn="+name+","+parent;
}
function renderBind(){
  const t=TPLS[HS.tpl];
  if(!HS.acct){
    HS.parent = HS.parent || SVC_OU;
    $("#bindCard").innerHTML=`
      <div class="alert warn" style="margin-bottom:14px">${ic("triangle-alert","ic")}<div><b>不要把 admin 的密码填进对方系统。</b>对方会长期持有这组凭据——等于交出整本目录的读写权。创建一个专用只读账号：可随时改密停用、出问题能审计到是谁。</div></div>
      <div style="display:flex;gap:10px;align-items:flex-end;flex-wrap:wrap">
        <div><label class="muted" style="font-size:12px">账号名</label>
          <input class="input mono" id="acctName" value="${t.acct}" style="margin-top:6px;width:220px"></div>
        <div><label class="muted" style="font-size:12px">创建位置（目录结构由你定，不强制）</label>
          <select class="select" id="acctParent" style="margin-top:6px;width:300px">
            <option value="${SVC_OU}" ${HS.parent===SVC_OU?"selected":""}>ou=services（推荐 · 不存在时自动创建）</option>
            <option value="${ROOT}" ${HS.parent===ROOT?"selected":""}>根目录（${ROOT}）</option>
            ${OU_OPTS.map(o=>`<option value="${o.dn}" ${HS.parent===o.dn?"selected":""}>${"　".repeat(o.depth)}${o.name}（${o.dn.split(",")[0]}）</option>`).join("")}
          </select></div>
        <button class="btn primary" id="acctCreate">${ic("user-plus","ic ic-14")}创建只读账号</button>
      </div>
      <div class="muted mono" style="font-size:12px;margin-top:10px">${ic("eye","ic ic-14")} 将创建条目：<span id="acctDnPreview" style="color:var(--primary)"></span></div>
      <details class="acl-ldif"><summary>${ic("chevron-down","ic ic-14")}服务端“只读”如何落地？展开查看 ACL 样例（由运维在 OpenLDAP 上执行）</summary>
        <pre class="ldif" style="margin-top:10px" id="aclLdif"></pre></details>`;
    const paintDn=()=>{ $("#acctDnPreview").textContent=acctPreviewDn();
      $("#aclLdif").textContent=
`dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcAccess
olcAccess: to dn.subtree="${ROOT}"
  by dn.exact="${acctPreviewDn()}" read
  by * break
# 释义：允许该账号只读本目录子树；写权限只有 admin 拥有。
# 账号本身条目：${acctPreviewDn()}
#   objectClass: organizationalRole + simpleSecurityObject（标准 schema，无需扩展）`; };
    paintDn();
    $("#acctName").addEventListener("input",paintDn);
    $("#acctParent").addEventListener("change",e=>{ HS.parent=e.target.value; paintDn(); });
    $("#acctCreate").addEventListener("click",()=>{
      const name=($("#acctName").value||"").trim()||t.acct;
      const parent=$("#acctParent").value||HS.parent;
      const autoOu = parent===SVC_OU && !OU_OPTS.some(o=>o.dn===SVC_OU);
      HS.acct={ name, dn:`cn=${name},${parent}`, pw:randPw() };
      HS.pwHidden=false; HS.test=null;
      renderBind(); renderConn();
      toast(`只读账号已创建（标准 LDAP Add）${autoOu?"，ou=services 不存在已自动创建":""} · LAM 侧立即可见`,"ok","user-check");
    });
  } else {
    $("#bindCard").innerHTML=`
      <div class="alert ok">${ic("circle-check","ic")}<div><b>账号已创建。</b>条目 <span class="mono">${esc(HS.acct.dn)}</span>（objectClass: organizationalRole + simpleSecurityObject）。目录里存的是哈希，密码<b>只显示这一次</b>：</div></div>
      <div class="pw-once">
        <i data-lucide="key-round" class="ic ic-18" style="color:#A24E08"></i>
        <span class="pw">${HS.pwHidden?"••••••••••••":esc(HS.acct.pw)}</span>
        ${!HS.pwHidden?`<button class="copy-btn" data-copy="${esc(HS.acct.pw)}">${ic("copy","ic ic-14")}复制密码</button>
        <button class="btn sm" id="pwHide">${ic("eye-off","ic ic-14")}我已保存，隐藏</button>`:""}
      </div>
      <div style="display:flex;gap:10px;align-items:center;margin-top:16px;flex-wrap:wrap">
        <b style="font-size:13px">交付前自测：</b>
        <button class="btn primary" id="bindTest">${ic("plug-zap","ic ic-14")}验证这组 Bind 凭据</button>
        <span class="muted" style="font-size:11.5px">模拟一次真实 bind——把凭据交给对方之前先确认它能通过目录认证</span>
        <a class="link" id="bindTestBad" style="font-size:11.5px">演示：用错误密码再测一次</a>
      </div>
      <div id="bindTestOut" style="margin-top:12px"></div>
      <div class="muted" style="font-size:11.5px;margin-top:14px">${ic("info","ic ic-14")} 账号的“只读”由服务端 ACL 决定（见上方样例 LDIF）。本工具不越权代改服务器 ACL。</div>`;
    bindCopy($$("#bindCard .copy-btn"));
    $("#pwHide") && $("#pwHide").addEventListener("click",()=>{ HS.pwHidden=true; renderBind(); });
    $("#bindTest").addEventListener("click",()=>runBindTest(true));
    $("#bindTestBad").addEventListener("click",()=>runBindTest(false));
    if(HS.test) paintTest(HS.test);
  }
  refreshIcons();
}
function paintTest(ok){
  $("#bindTestOut").innerHTML = ok
    ? `<div class="bindtest-ok g">${ic("circle-check","ic")}<div><b>bind 验证通过（模拟）。</b>这组 DN + 密码能通过目录认证，可以放心填进对方系统。接下来在对方保存配置后，用审计日志确认它的首次访问。</div></div>`
    : `<div class="bindtest-ok r">${ic("circle-x","ic")}<div><b>bind 验证失败（49 invalidCredentials，模拟）。</b>白话：DN 或密码不对。常见原因：① 复制密码时带了空格；② 密码已重置过（旧的失效）；③ DN 抄错一段。回上面重新复制一次再试。</div></div>`;
  refreshIcons();
}
function runBindTest(ok){
  HS.test=ok;
  $("#bindTestOut").innerHTML=`<div class="muted" style="font-size:12px">${ic("loader-circle","ic ic-14")} 正在向目录发起 bind 验证…</div>`;
  refreshIcons();
  setTimeout(()=>paintTest(ok), 700);
}

/* ---- ④ 术语词典 ---- */
const GLOSS_ATTR = [
  ["cn","姓名 / 通用名","条目的显示名。人员的姓名、组的组名都用它。"],
  ["sn","姓","人员的姓氏。schema 必填，由姓名自动推导（含复姓）。"],
  ["givenName","名","名字部分，选填。"],
  ["uid","账号","登录用账号（如 zhangwei），全组织唯一。"],
  ["mail","邮箱","RFC 822 格式，可被对接系统用作用户邮箱字段。"],
  ["mobile","手机","移动电话。"],
  ["telephoneNumber","座机","办公电话。"],
  ["employeeNumber","工号","企业内部编号，导入时用于识别同一人。"],
  ["title","职位","职务头衔。"],
  ["ou","部门 / 组织单元","部门条目的标识；作为人员属性时表示所属部门。多值：第 1 个=英文标识，第 2 个=中文名。"],
  ["departmentNumber","部门编号","另一种记录部门编号的属性，本目录默认不启用。"],
  ["displayName","显示名","某些客户端优先显示的名称，本目录默认不启用。"],
  ["userPassword","密码","存储密码（目录内存哈希，界面永不回显）。"],
  ["objectClass","对象类","条目的类型模板，决定拥有哪些属性。"],
  ["member","组成员（DN）","权限组（groupOfNames）的成员，值为完整 DN。"],
  ["memberUid","组成员（uid）","登录组（posixGroup）的成员，值为账号名。"],
  ["member / isMemberOf","反向成员","开启 memberOf 后服务器自动生成，反查“某人在哪些组”。"],
  ["gidNumber","组 ID","登录组的数字编号，Linux 用；5000-5999 自动分配。"],
  ["uidNumber","用户 ID","Linux 登录账号的数字编号。"],
  ["homeDirectory","主目录","Linux 登录后的 home 路径。"],
  ["loginShell","登录 Shell","如 /bin/bash。"],
  ["entryUUID","唯一标识","服务器自动生成、永不改变。对接系统推荐用它识别人员。"],
  ["entryCSN","版本号","每次修改自动变化，用于并发冲突检测。"],
  ["creatorsName","创建者","创建该条目的操作者 DN，服务器自动记录。"],
  ["createTimestamp","创建时间","条目创建时间（UTC），服务器自动记录。"],
  ["modifiersName","修改者","最后一次修改者 DN。"],
  ["modifyTimestamp","修改时间","最后一次修改时间。"],
  ["description","说明","通用描述字段：部门说明、组说明都在这。"],
  ["ouOrder","部门排序","本部署扩展的部门显示顺序值，目录树按它排序。"],
];
const GLOSS_OC = [
  ["inetOrgPerson","人员（主类）","“互联网组织人员”，管理台新建人员的默认主类。"],
  ["organizationalUnit","部门（OU）","组织单元，树的枝干。第二 ou 值存中文名。"],
  ["groupOfNames","权限组","成员为 DN 的组，用于授权场景（VPN / 堡垒机）。member 必填至少一名。"],
  ["posixGroup","登录组","带 gidNumber 的组，memberUid 存账号名，Linux 主机用。"],
  ["posixAccount","Linux 登录账号","附加在人员上的辅助类，开启后可登录 Linux 主机。"],
  ["organizationalRole","角色 / 服务账号","角色类条目，对接账号（Bind DN）用它承载身份。"],
  ["simpleSecurityObject","简单密码载体","让 organizationalRole 这类条目能挂 userPassword。"],
  ["top","顶层抽象类","所有条目隐含的根类，无需手工关心。"],
  ["person","人","inetOrgPerson 的父类，只要 cn + sn。"],
  ["organizationalPerson","组织人员","person 的子类，inetOrgPerson 的直接父类。"],
];
function renderGloss(){
  const q=(HS.glossQ||"").toLowerCase();
  const src=HS.glossTab==="attr"?GLOSS_ATTR:GLOSS_OC;
  const list=src.filter(g=>!q||g.join(" ").toLowerCase().includes(q));
  $("#glossBody").innerHTML = list.length
    ? list.map(g=>`<tr><td class="mono">${esc(g[0])}</td><td><b>${esc(g[1])}</b></td><td style="color:var(--t2)">${esc(g[2])}</td></tr>`).join("")
    : `<tr><td colspan="3"><div class="empty">${ic("search-x","ic")}<div class="t">没有匹配「${esc(HS.glossQ)}」的词条</div></div></td></tr>`;
  refreshIcons();
}
$("#glossQ") && $("#glossQ").addEventListener("input",e=>{ HS.glossQ=e.target.value; renderGloss(); });
$$("#glossSeg button").forEach(b=>b.addEventListener("click",()=>{
  $$("#glossSeg button").forEach(x=>x.classList.remove("on")); b.classList.add("on");
  HS.glossTab=b.textContent.includes("属性")?"attr":"oc"; renderGloss();
}));

/* ---- 聚光灯导览（自研 CoachMark，无第三方库） ---- */
const TOUR = [
  { sel:"#tree",              side:"right", t:"这就是目录本身", d:"左侧的树是 LDAP 目录（DIT）：部门、人员、组都是树上的条目。你在任何 LDAP 客户端里看到的这棵树是同一棵——因为目录即真相。" },
  { sel:"#treeNew",           side:"right", t:"新建部门", d:"部门（OU）是树的枝干。这里新建的部门立刻出现在 phpLDAPadmin、LAM 等其他客户端里，反之亦然。" },
  { sel:'#mainNav .item[data-page="people"]', side:"right", t:"日常入口：组织与人员", d:"左树右表。支持行内加人、勾选批量调动、自定义列与排序；Excel 导入在「导入导出」页。" },
  { sel:"#globalSearch",      side:"bottom", t:"全局搜索", d:"输入姓名或账号，回车前就会出现建议，点结果直接打开人员详情——不用先找到部门。" },
  { sel:'#mainNav .item[data-page="help"]',  side:"right", t:"帮助中心", d:"对接堡垒机 / 零信任该填什么、LDAP 概念看不懂、术语对照——都在这里。导览完成！" },
];
function tourShow(){
  const step=TOUR[HS.tour], el=step.sel && $(step.sel);
  const mask=$("#coachMask");
  if(!mask||!el){ tourEnd(); return; }
  const r=el.getBoundingClientRect(), pad=6;
  const spot=$("#coachSpot"), bub=$("#coachBub");
  spot.style.cssText=`left:${r.left-pad}px;top:${r.top-pad}px;width:${r.width+pad*2}px;height:${r.height+pad*2}px`;
  const bw=300, bh=170;
  let left,top;
  if(step.side==="right" && r.right+bw+24<innerWidth){ left=r.right+24; top=Math.min(Math.max(r.top,12),innerHeight-bh-12); }
  else if(step.side==="bottom" && r.bottom+bh+20<innerHeight){ left=Math.min(Math.max(r.left,12),innerWidth-bw-12); top=r.bottom+16; }
  else { left=Math.min(Math.max(r.left,12),innerWidth-bw-12); top=Math.max(12,r.top-bh-16); }
  bub.style.cssText=`left:${left}px;top:${top}px`;
  bub.innerHTML=`
    <span class="cs-tag">${ic("spotlight","ic ic-14")}界面导览 ${HS.tour+1} / ${TOUR.length}</span>
    <h4>${step.t}</h4><p>${step.d}</p>
    <div class="cb-nav">
      <button class="btn sm" id="cbSkip">跳过</button><span class="grow"></span>
      ${HS.tour>0?`<button class="btn sm" id="cbPrev">上一步</button>`:""}
      <button class="btn sm primary" id="cbNext">${HS.tour===TOUR.length-1?"完成":"下一步"}</button>
    </div>
    <div class="cb-dots" style="margin-top:10px">${TOUR.map((_,i)=>`<i class="${i<=HS.tour?"on":""}"></i>`).join("")}</div>`;
  $("#cbSkip").onclick=tourEnd;
  const pv=$("#cbPrev"); if(pv) pv.onclick=()=>{ HS.tour--; tourShow(); };
  $("#cbNext").onclick=()=>{ HS.tour===TOUR.length-1 ? (tourEnd(), toast("导览完成。之后随时在帮助中心重看","ok","party-popper")) : (HS.tour++, tourShow()); };
  refreshIcons();
}
function tourStart(){
  if(!$("#coachMask")){
    document.body.insertAdjacentHTML("beforeend",`<div class="coach-mask" id="coachMask"><div class="coach-spot" id="coachSpot"></div><div class="coach-bub" id="coachBub"></div></div>`);
    $("#coachMask").addEventListener("click",e=>{ if(e.target.id==="coachMask") tourEnd(); });
  }
  $("#coachMask").style.display="";
  HS.tour=0; tourShow();
}
function tourEnd(){ const m=$("#coachMask"); if(m) m.style.display="none"; HS.tour=-1; }
document.addEventListener("keydown",e=>{ if(e.key==="Escape") tourEnd(); });
document.addEventListener("click",e=>{ if(e.target.closest('[data-demo="tour"]')){ if(!document.body.dataset.view.startsWith("admin")) showView("admin-help:onboard"); setTimeout(tourStart,120); } });
$("#tourReplay") && $("#tourReplay").addEventListener("click",tourStart);
$("#tourStart") && $("#tourStart").addEventListener("click",tourStart);

/* ---- 初始化 ---- */
renderOb(); renderLesson(); renderTpls(); renderConn(); renderBind(); renderGloss();
