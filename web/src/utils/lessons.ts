// LDAP 教学课件（帮助中心）：9 节课程内容。
// 原则：不用虚构例子——课程 HTML 由 ctx（登录者 DN / BaseDN / 真实样例人员）注入真实值，
// 管理员学到的概念直接对得上自己屏幕上看到的东西。
// filter 一课的实验器由 LearnPanel 在课件之后单独挂载（不走 v-html）。

export interface LessonCtx {
  baseDN: string
  meDN: string
  person?: { dn: string; cn: string; uid: string; ou: string; mail?: string; title?: string } | null
}

export interface Lesson {
  key: string
  icon: string
  title: string
  sub: string
  lab?: boolean // 第 5 课：渲染过滤器实验器插槽
  html: (ctx: LessonCtx) => string
}

const esc = (s: string) =>
  s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

function dnBreak(dn: string, labels: string[]): string {
  const parts = dn.split(',')
  return (
    '<div class="dn-breakdown">' +
    parts
      .map(
        (s, i) =>
          `<span class="seg-dn"><b>${esc(s)}</b><i>${esc(labels[i] || s.split('=')[0])}</i></span>` +
          (i < parts.length - 1 ? '<span class="sep">,</span>' : '')
      )
      .join('') +
    '</div>'
  )
}

function where(html: string): string {
  return `<div class="where"><b>↪ 概念在 NewLDAP 里对应什么</b>${html}</div>`
}

export const LESSONS: Lesson[] = [
  {
    key: 'dn', icon: 'ListTree', title: '目录、条目与 DN',
    sub: '用你自己目录里的真实数据理解 LDAP 的"门牌地址"',
    html(c) {
      const p = c.person
      return `
<p>LDAP 目录是一棵<strong>树</strong>（叫 <span class="term" title="DIT（Directory Information Tree）：目录信息树。LDAP 里所有数据组成的树形结构，像公司的组织架构图。">DIT</span>），树上每个节点叫<strong>条目</strong>（entry），每个条目有一个全局唯一的 <span class="term" title="DN（Distinguished Name）：条目在树中的完整路径，全局唯一，像完整的门牌地址。">DN</span>。你现在登录用的账号 DN 是：</p>
${dnBreak(c.meDN, ['账号名', '……中间路径', '域名第一段', '域名第二段'])}
${p ? `<p>再看你目录里的一名真实员工「${esc(p.cn)}」——DN 从左往右读：<b>最左一段叫 <span class="term" title="RDN（Relative Distinguished Name）：DN 的最左一段，只在同一父节点下需要唯一。">RDN</span></b>（这个人的账号），往右依次是它挂在哪里、树的根：</p>${dnBreak(p.dn, ['账号（RDN）', '所在部门', '域名第一段', '域名第二段'])}` : ''}
<h4>为什么"调动部门"在 LDAP 里是移动节点</h4>
<p>DN 含路径，员工换部门 = 挂到另一个父节点 = <b>整个 DN 变了</b>（本工具自动用 modrdn 完成）。但条目的 <span class="mono">entryUUID</span> 永远不变——所以对接外部系统时推荐用它当唯一标识（见对接指引）。</p>
<h4>DC / OU / CN / UID：路径里的段叫什么</h4>
<table>
<tr><th>前缀</th><th>含义</th><th>常见于</th></tr>
<tr><td class="mono">dc=</td><td>域名组件（domain component）</td><td>树根：${esc(c.baseDN)}</td></tr>
<tr><td class="mono">ou=</td><td>组织单元（部门/分支）</td><td>ou=tech,ou=…</td></tr>
<tr><td class="mono">cn=</td><td>通用名（组名、角色名）</td><td>cn=vpn-access,ou=groups,…</td></tr>
<tr><td class="mono">uid=</td><td>账号（人员）</td><td>uid=zhangwei,ou=tech,…</td></tr>
</table>
${where('左侧目录树 = DIT 本身 · 人员详情页头部 = 该条目 DN · 「调动部门」= 移动条目（DN 改变，entryUUID 不变）')}`
    },
  },
  {
    key: 'oc', icon: 'Shapes', title: 'objectClass 与 schema',
    sub: '条目的"模板"：决定它有哪些字段、哪些必填',
    html() {
      return `
<p>每个条目都挂着一个或多个 <span class="term" title="objectClass：条目的类型模板，由服务器 schema 定义。决定该条目拥有哪些属性、哪些必填（MUST）哪些可选（MAY）。">objectClass</span>（对象类），相当于"这个条目按哪个模板创建"。本目录里你会遇到这几类：</p>
<table>
<tr><th>objectClass</th><th>角色</th><th>必填（MUST）</th><th>常用可选（MAY）</th></tr>
<tr><td class="mono">inetOrgPerson</td><td>人员主类</td><td class="mono">cn、sn</td><td class="mono">uid、mail、mobile、title、employeeNumber…</td></tr>
<tr><td class="mono">posixAccount</td><td>辅助类 · 可登录 Linux</td><td class="mono">uid、uidNumber、gidNumber、homeDirectory</td><td class="mono">loginShell</td></tr>
<tr><td class="mono">organizationalUnit</td><td>部门（OU）</td><td class="mono">ou</td><td class="mono">description、ouOrder</td></tr>
<tr><td class="mono">groupOfNames</td><td>权限组</td><td class="mono">cn、member</td><td class="mono">description</td></tr>
<tr><td class="mono">posixGroup</td><td>登录组</td><td class="mono">cn、gidNumber</td><td class="mono">memberUid、description</td></tr>
<tr><td class="mono">organizationalRole</td><td>角色 / 服务账号</td><td class="mono">cn</td><td class="mono">userPassword（配合 simpleSecurityObject）</td></tr>
</table>
<h4>主类 vs 辅助类</h4>
<p>主类定"这是什么"（人是 inetOrgPerson）；辅助类是给它<b>追加能力</b>——人员勾选「Linux 登录」就是追加 posixAccount，自动分配 uidNumber；对接只读账号用 organizationalRole + simpleSecurityObject 携带密码。</p>
<h4>schema 是服务器定的，不能随便发明</h4>
<p>对象类与属性清单由 OpenLDAP 服务器的 schema（结构定义）决定，所有客户端（本工具 / LAM / phpLDAPadmin）看到的都是同一份。想新增自定义属性（如本部署的 ouOrder）需要运维在服务器上注入 schema——界面上能做的是"在已有 schema 内填表"。</p>
${where('人员详情「对象类」页签 = 主类 / 辅助类卡片 · 新建人员表单 = 按 inetOrgPerson 的 MUST/MAY 自动生成 · 术语词典 = 本服务器支持的对象类对照')}`
    },
  },
  {
    key: 'attr', icon: 'TableProperties', title: '属性与多值',
    sub: '字段名是英文缩写，一个字段可以有多个值',
    html(c) {
      const p = c.person
      return `
<p>条目的数据就是一组<span class="term" title="属性（Attribute）：条目上的一个字段，形如 mail=${esc((c.person && c.person.mail) || 'user@域名')}。属性名由 schema 定义，值可以有多个。">属性</span>（attribute）。属性名是 schema 定死的英文缩写，值由你填。两个新人最容易懵的点：</p>
<h4>1. 一个属性可以有多个值</h4>
<p>比如部门的 <span class="mono">ou</span> 属性：<b>第一个值</b>是英文标识（用在 DN 里，如 <span class="mono">ou=tech</span>），<b>第二个值</b>是中文显示名（如"技术研发部"）。本工具的目录树、部门选择器读的就是第二个值——这不是本产品发明的花样，是 LDAP 多值属性的标准用法。人员的 ou 属性同理${p ? `（「${esc(p.cn)}」的 ou = ${esc(p.ou.split(',')[0])}）` : ''}。</p>
<h4>2. 操作属性：服务器自动维护的"元数据"</h4>
<table>
<tr><th>属性</th><th>白话</th><th>为什么重要</th></tr>
<tr><td class="mono">entryUUID</td><td>永不改变的全局唯一 ID</td><td>对接堡垒机/零信任的推荐唯一标识：改名换部门都不丢身份</td></tr>
<tr><td class="mono">entryCSN</td><td>每次修改都会变的版本号</td><td>本工具保存前比对它，防止覆盖 LAM 等其他客户端的并发修改</td></tr>
<tr><td class="mono">createTimestamp / modifyTimestamp</td><td>创建 / 最后修改时间</td><td>审计"这账号什么时候建的"</td></tr>
<tr><td class="mono">creatorsName / modifiersName</td><td>谁建 / 谁改的</td><td>操作者 DN，可直接定位到人</td></tr>
</table>
<h4>3. 密码是属性，但你永远看不到它</h4>
<p><span class="mono">userPassword</span> 存的是哈希，任何正规客户端（包括本工具）都只写不读。界面上的"重置密码 / 禁用账号"本质上都是在改这个属性。</p>
${where('人员详情「属性」页签 = 全部业务属性 · 「操作属性」独立页签 = 上表那批 · 悬停任意属性的 ? 图标 = 中文说明')}`
    },
  },
  {
    key: 'bind', icon: 'KeyRound', title: 'Bind 与权限',
    sub: 'LDAP 没有"登录后赋权"这回事：你是谁，就能做什么',
    html(c) {
      return `
<p><span class="term" title="bind：用某个账号的 DN 和密码向 LDAP 服务器证明身份，之后所有操作都以这个身份进行。">bind</span>（绑定）就是 LDAP 的"登录"：拿一个 DN + 密码向服务器证明身份。你刚才登录本工具（<span class="mono">${esc(c.meDN)}</span>）就是一次真实 bind。之后能做什么，完全由服务器上的 <span class="term" title="ACL（Access Control List）：服务器上的访问控制列表，规定哪个账号能读/写哪些子树。">ACL</span> 决定——<b>目录里没有额外的"管理员权限表"</b>。</p>
<h4>这个设计带来两个本产品的关键行为</h4>
<p>① 你在本工具做的每件事 = 这个账号在 OpenLDAP 上的权限，工具不额外设限也不额外放权；② 换一个只有读权限的账号登录，界面照常能用，保存会被服务器拒绝——错误会原样白话转译（"当前账号无权……（服务端 ACL 拒绝）"）。</p>
<h4>对接场景：Bind DN 是什么</h4>
<p>堡垒机 / 零信任接入时填的「Bind DN」，就是给对方系统一个 LDAP 身份去查目录。三种常见身份的取舍：</p>
<table>
<tr><th>身份</th><th>权限</th><th>评价</th></tr>
<tr><td class="mono">cn=admin</td><td>整本目录读写</td><td><b>绝对不要</b>——等于把目录交给对方系统</td></tr>
<tr><td>匿名（anonymous）</td><td>取决于 ACL，默认读不到</td><td>不推荐：不可审计、常被禁</td></tr>
<tr><td>专用只读账号</td><td>仅读（配合 ACL）</td><td><b>推荐</b>：可随时改密停用、审计能定位是谁在查</td></tr>
</table>
${where('登录页 = 真实 bind · 概览「目录状态」= 当前连接 · 对接指引第 3 步 = 一键创建低权限 bind 账号')}`
    },
  },
  {
    key: 'filter', icon: 'Filter', title: '过滤器语法',
    sub: '对接表单里的「用户过滤器」就是它——在实验器里随便试',
    lab: true,
    html() {
      return `
<p>LDAP 查询用 <span class="term" title="过滤器（Search Filter）：形如 (objectClass=inetOrgPerson) 的查询条件表达式，括号必需。">过滤器</span>（filter）表达"要找哪些条目"。整个表达式<b>必须包在括号里</b>。常用语法一共就这几个：</p>
<table>
<tr><th>写法</th><th>含义</th><th>例子</th></tr>
<tr><td class="mono">(属性=值)</td><td>等于</td><td class="mono">(uid=zhangwei)</td></tr>
<tr><td class="mono">(属性=前*)</td><td>前缀匹配（* 是通配符，也可放中间/结尾）</td><td class="mono">(cn=张*)</td></tr>
<tr><td class="mono">(属性=*)</td><td>有这个属性就行（存在性）</td><td class="mono">(mail=*)</td></tr>
<tr><td class="mono">(&amp;(a)(b))</td><td>并且 AND</td><td class="mono">(&amp;(objectClass=inetOrgPerson)(mail=*))</td></tr>
<tr><td class="mono">(|(a)(b))</td><td>或者 OR</td><td class="mono">(|(objectClass=groupOfNames)(objectClass=posixGroup))</td></tr>
<tr><td class="mono">(!(a))</td><td>非 NOT</td><td class="mono">(!(mail=*)) 没有邮箱的</td></tr>
</table>
<h4>三个实战要点</h4>
<p>① <b>过滤器按"属性值"匹配，不按类型</b>——想只匹配人员必须带 <span class="mono">objectClass=inetOrgPerson</span>，否则部门、组都会混进来；② 通配符放<b>开头</b>（<span class="mono">*=x</span>）性能差，大目录慎用；③ 语法错误（少括号、缺等号）服务器直接返回错误 21（Bad search filter），不会"尽量执行"。</p>
<p>下面实验器执行的是<b>真实查询</b>（走本目录的搜索接口，只读、最多 50 条）：</p>`
    },
  },
  {
    key: 'group', icon: 'UsersRound', title: '组的两种形态',
    sub: '权限组按 DN 授权，登录组按 uid 认账号',
    html() {
      return `
<p>本目录用两种组服务不同场景，字段完全不同，别混：</p>
<table>
<tr><th></th><th>权限组（groupOfNames）</th><th>登录组（posixGroup）</th></tr>
<tr><td><b>用途</b></td><td>授权：VPN / 堡垒机 / 应用权限</td><td>Linux 主机登录资格</td></tr>
<tr><td><b>成员字段</b></td><td class="mono">member = 完整 DN</td><td class="mono">memberUid = uid</td></tr>
<tr><td><b>数字 ID</b></td><td>无</td><td class="mono">gidNumber（5000-5999 自动分配）</td></tr>
<tr><td><b>成员能否为空</b></td><td>不能（schema 必填至少 1 名）</td><td>可以</td></tr>
<tr><td><b>放哪</b></td><td>ou=groups 或任意约定位置</td><td>同左</td></tr>
</table>
<h4>为什么授权推荐用权限组</h4>
<p>member 存 DN，指向明确、不受重名影响；零信任 / 堡垒机读它做"谁有权用什么"。登录组是给 <span class="mono">sshd</span>、<span class="mono">su</span> 这类 Linux 组件看的，它们只认 uid 和 gidNumber。</p>
<h4>成员关系存在组上，不在人上</h4>
<p>人员条目上<b>没有</b>"所属组"属性（除非服务器开了 memberOf 反向属性）。想知道"张伟在哪些组"，实际是搜"哪些组的 member 包含张伟的 DN"。本工具的人员详情「所属组」区块就是这样做反查的。</p>
${where('用户组页 = 两类合并展示 · 新建组 = 先选场景（权限 / 登录）· 人员详情专家模式顶部 = 所属组反查 · 对接指引「组映射」= 告诉对方系统按哪个字段读成员')}`
    },
  },
  {
    key: 'ldif', icon: 'FileCode', title: 'LDIF：目录的通用语言',
    sub: '所有 LDAP 工具都认的文本格式——看得懂它，排错就通了一半',
    html(c) {
      return `
<p>LDIF（LDAP Data Interchange Format）是描述目录内容的纯文本格式。命令行工具（ldapadd / ldapmodify）、LAM 的导入、本工具的专家模式用的都是它。四种最常见的形式：</p>
<h4>① 描述一个条目（长什么样）</h4>
<pre>dn: uid=zhangwei,ou=tech,${esc(c.baseDN)}
objectClass: inetOrgPerson
cn: 张伟
sn: 张
uid: zhangwei
mail: ${esc((c.person && c.person.mail) || 'user@域名')}</pre>
<p>第一行永远是 <span class="mono">dn:</span>，其后每行一个属性。<b>同名属性出现多行 = 多值</b>（部门的中文名就是这么存的：第二行 ou:）。</p>
<h4>② 新增（changetype: add）</h4>
<pre>dn: ou=services,${esc(c.baseDN)}
changetype: add
objectClass: organizationalUnit
ou: services</pre>
<h4>③ 修改（changetype: modify，分增/删/替换三种操作）</h4>
<pre>dn: uid=zhangwei,ou=tech,${esc(c.baseDN)}
changetype: modify
replace: mobile
mobile: 139-0101-2233
-
add: title
title: 高级工程师</pre>
<p><span class="mono">-</span> 单独一行是操作之间的分隔符。本工具保存时按<b>属性级</b>提交——等价于只 modify 真正变过的属性，其他客户端的并发修改不受影响。</p>
<h4>④ 删除</h4>
<pre>dn: uid=zhangwei,ou=tech,${esc(c.baseDN)}
changetype: delete</pre>
${where('条目详情「LDIF」页签 = 当前条目的只读 LDIF · 专家模式新建条目支持直接粘贴 LDIF')}`
    },
  },
  {
    key: 'design', icon: 'Network', title: '目录树设计建议',
    sub: '没有标准答案，但有被反复验证的约定俗成',
    html(c) {
      return `
<p>LDAP 不会强制你怎么建树，但树一旦定下来，所有对接系统都要按它配置——<b>先想清楚再动手</b>。以下是本工具视角的实用建议：</p>
<h4>1. 根用域名，部门按业务建</h4>
<p>根 <span class="mono">${esc(c.baseDN)}</span>（dc= 域名）是惯例；部门（ou=）按你们真实的组织架构建。本工具支持任意层级嵌套与拖拽排序、中文名。</p>
<h4>2. 层级宁浅勿深</h4>
<p>2-4 层足够（公司 → 部门 → 组）。每深一层，对接系统的"用户 Base DN"就更难写对，调部门时也更痛。跨部门的虚拟团队用<b>组</b>表达，不要建 OU。</p>
<h4>3. 人员、组、服务账号分区放</h4>
<table>
<tr><th>内容</th><th>建议位置</th><th>为什么</th></tr>
<tr><td>人员</td><td class="mono">按部门 ou= 散放</td><td>组织架构天然分层；对接系统配全树搜索即可命中</td></tr>
<tr><td>组</td><td class="mono">ou=groups,&lt;base&gt;</td><td>集中管理，组过滤器好写</td></tr>
<tr><td>对接/服务账号</td><td class="mono">ou=services,&lt;base&gt;（或你们已有的约定）</td><td>与人员隔离，不进人员过滤器；换对接方时好清理</td></tr>
</table>
<p>很多老教程让你把人员全部塞进 <span class="mono">ou=people</span>——那是影子数据库产品的习惯（先同步进库再管理）。本工具目录即真相，人员放真实部门下即可；对接系统只要把「用户 Base DN」写成根或具体部门都能搜到。</p>
<h4>4. 命名用稳定的标识</h4>
<p>OU 用英文标识（DN 里的 ou=tech 不动）+ 第二个 ou 值放中文名——改名只改中文值，DN 永不变，对接系统零影响。人员同理：uid 是账号标识，姓名写在 cn。</p>
<h4>5. 预留"对接产物"的位置</h4>
<p>开始对接前先建好 ou=services（帮助中心可代建），避免第一个对接账号建到奇怪的位置。同理约定组命名（如 <span class="mono">vpn-access</span>、<span class="mono">bastion-ops</span>），一看名字就知道是哪个系统的权限组。</p>
${where('目录树拖拽 = 调 ouOrder 显示顺序 · 新建部门 = 英文标识 + 中文名 · 对接指引第 3 步 = 在选定位置创建服务账号')}`
    },
  },
  {
    key: 'pitfall', icon: 'TriangleAlert', title: '常见误区',
    sub: '五个新手几乎都会踩的坑',
    html() {
      return `
<div class="faq-list">
<details open><summary>改 uid 不是"改个字段"，是移动条目</summary><div>uid 出现在 DN 里（uid=zhangwei,ou=tech,…）。改 uid = 改 RDN = LDAP 的 modrdn 操作，本工具会在保存时自动处理成移动。但如果对方系统把 uid 存下来当了外键，人员改账号后对方那边会"失联"——这就是对接时推荐用 <span class="mono">entryUUID</span> 当唯一标识的原因。</div></details>
<details><summary>权限组删不掉最后一名成员，不是 bug</summary><div>groupOfNames 的 member 是 schema MUST（必填）。想清空一个权限组：先加新成员再移旧成员，或者直接删组。本工具保存被服务器拒绝时会把原因白话转译出来。</div></details>
<details><summary>"禁用账号" ≠ 删除条目</summary><div>禁用是把密码置为失效（无法再 bind），条目和所有属性原样保留——工号、邮箱、组员身份都在，随时可恢复。删除条目则永久移出目录。离职处理建议：先禁用观察一段时间，确认无系统依赖后再删。</div></details>
<details><summary>删除部门前要先清空它</summary><div>LDAP 不允许删除有子节点的条目（树不能"断枝"）。部门下还有人或有子部门时删除会失败，先把人员移走、子部门删掉。本工具删除前会先做预览检查。</div></details>
<details><summary>不要用 admin 对接任何外部系统</summary><div>对方系统会长期持有这组凭据——等于交出整本目录的读写权，且无法单独吊销。永远用「对接指引」创建的专用只读账号：权限最小、可独立改密停用、审计能定位到是谁在查。</div></details>
</div>`
    },
  },
]
