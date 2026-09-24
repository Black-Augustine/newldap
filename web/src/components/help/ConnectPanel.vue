<template>
  <div>
    <!-- 对外通告地址 -->
    <div class="nl-card" style="margin-bottom: 14px">
      <div class="nl-card-head">
        <span class="t"><el-icon><Megaphone /></el-icon>对外通告地址</span>
        <span class="nl-pill warn">先确认这一项</span>
      </div>
      <div class="nl-card-body" style="display: flex; gap: 14px; flex-wrap: wrap">
        <div style="flex: 1; min-width: 320px">
          <div class="muted" style="margin-bottom: 6px">对方产品能访问到的<b>目录服务器</b>地址（不是本管理台地址）</div>
          <el-input v-model="addr" class="mono" placeholder="ldap://192.168.1.10:389 或 ldaps://…:636" @change="saveAddr" />
        </div>
        <div class="nl-hint info" style="flex: 1; min-width: 320px">
          <el-icon style="margin-top: 2px"><Lightbulb /></el-icon>
          <div>新手最易混：这里填 <b>OpenLDAP 目录服务器</b>，<b>不是</b> NewLDAP 管理台的网址。本管理台只是目录的"遥控器"，对方系统要直连目录本身。默认值按浏览器地址推断（{{ guessed }}），请按实际网络修正。</div>
        </div>
      </div>
    </div>

    <!-- 产品模板 -->
    <div class="tpl-grid">
      <div
        v-for="t in TPLS" :key="t.key" class="tpl" :class="{ on: tplKey === t.key }"
        @click="switchTpl(t.key)"
      >
        <el-icon :size="20" style="color: var(--el-color-primary)"><component :is="TICONS[t.icon] ?? Plug" /></el-icon>
        <b>{{ t.name }}</b>
        <p>{{ t.desc }}</p>
        <span class="muted">{{ t.full }}</span>
      </div>
    </div>

    <!-- 信息卡 -->
    <div class="nl-card" style="margin-top: 14px">
      <div class="nl-card-head">
        <span class="t"><el-icon><ClipboardCopy /></el-icon>对接信息卡</span>
        <div style="display: flex; gap: 8px; align-items: center">
          <span class="muted">当前模板：{{ tpl.name }}（{{ tpl.full }}）</span>
          <el-button @click="copyAll"><el-icon><Copy /></el-icon>&nbsp;一键复制全部</el-button>
        </div>
      </div>
      <div class="nl-card-body">
        <div class="nl-hint info" style="margin-bottom: 14px">
          <el-icon style="margin-top: 2px"><Lightbulb /></el-icon>
          <div>{{ tpl.tip }}</div>
        </div>
        <div class="icard">
          <div v-for="sec in sections" :key="sec.title" class="sec">
            <div class="sec-h">
              <el-icon :size="14"><component :is="SICONS[sec.icon] ?? Plug" /></el-icon>{{ sec.title }}
              <span class="m">{{ sec.note }}</span>
            </div>
            <div v-for="r in sec.rows" :key="r.k" class="row">
              <el-tooltip :content="r.tip ?? r.k" placement="top" :show-after="300">
                <span class="k"><span class="term-q">{{ r.k }}</span></span>
              </el-tooltip>
              <span class="v" :class="{ plain: r.plain, ghost: r.ghost }">{{ r.v }}</span>
              <button v-if="r.copy" class="copy-btn" :class="{ ok: copied === r.v }" @click="copy(r.v)">
                <el-icon :size="13"><component :is="copied === r.v ? Check : Copy" /></el-icon>{{ copied === r.v ? '已复制' : '复制' }}
              </button>
            </div>
          </div>
        </div>
        <div class="muted" style="margin-top: 12px; display: flex; gap: 6px; align-items: flex-start">
          <el-icon style="margin-top: 2px"><Info /></el-icon>
          <span>以上取值实时来自当前目录与连接配置（Base DN：{{ baseDN || '…' }}）。目录里改了（如新建了账号），这里刷新即变——本工具不保存任何"对接配置副本"。</span>
        </div>
      </div>
    </div>

    <!-- 只读账号 -->
    <div class="nl-card" style="margin-top: 14px">
      <div class="nl-card-head">
        <span class="t"><el-icon><UserCheck /></el-icon>第 3 步 · 对接专用只读账号</span>
        <span class="nl-pill plain">强烈建议</span>
      </div>
      <div class="nl-card-body">
        <template v-if="!acct">
          <div class="nl-hint warn" style="margin-bottom: 14px">
            <el-icon style="margin-top: 2px"><TriangleAlert /></el-icon>
            <div><b>不要把 admin 的密码填进对方系统。</b>对方会长期持有这组凭据——等于交出整本目录的读写权，且无法单独吊销。创建一个专用只读账号：权限最小（配合下方 ACL）、可随时改密停用、审计能定位到是谁在查。</div>
          </div>
          <div style="display: flex; gap: 12px; align-items: flex-end; flex-wrap: wrap">
            <div>
              <div class="muted" style="margin-bottom: 6px">账号名</div>
              <el-input v-model="acctName" class="mono" style="width: 220px" placeholder="如 jumphub" />
            </div>
            <div>
              <div class="muted" style="margin-bottom: 6px">创建位置（目录结构由你定，不强制）</div>
              <el-select v-model="acctParent" style="width: 330px">
                <el-option :value="svcOU" :label="`ou=services（推荐 · 不存在时自动创建）`" />
                <el-option :value="baseDN" :label="`根目录（${baseDN}）`" />
                <el-option v-for="o in ouOptions" :key="o.dn" :value="o.dn" :label="`${'　'.repeat(o.depth)}${o.name}（${o.dn.split(',')[0]}）`" />
              </el-select>
            </div>
            <el-button type="primary" :loading="creating" @click="create">
              <el-icon><UserPlus /></el-icon>&nbsp;创建只读账号
            </el-button>
          </div>
          <div class="muted mono" style="margin-top: 10px; display: flex; gap: 6px; align-items: center">
            <el-icon :size="13"><Eye /></el-icon>将创建条目：<span style="color: var(--el-color-primary)">{{ previewDN }}</span>
          </div>
          <el-collapse style="margin-top: 12px">
            <el-collapse-item name="acl">
              <template #title><span class="acl-t">服务端"只读"如何落地？展开查看 ACL 样例（由运维在 OpenLDAP 上执行）</span></template>
              <pre class="ldif">{{ aclLDIF }}</pre>
            </el-collapse-item>
          </el-collapse>
        </template>
        <template v-else>
          <div class="nl-hint ok">
            <el-icon style="margin-top: 2px"><CircleCheck /></el-icon>
            <div>
              <b>账号已创建。</b>条目 <span class="mono">{{ acct.dn }}</span>（objectClass: organizationalRole + simpleSecurityObject），LAM / phpLDAPadmin 立即可见。
              目录里存的是哈希，密码<b>只显示这一次</b>：
            </div>
          </div>
          <div class="pw-once">
            <el-icon :size="18" color="#a24e08"><KeyRound /></el-icon>
            <span class="pw">{{ pwHidden ? '••••••••••••' : acct.password }}</span>
            <template v-if="!pwHidden">
              <button class="copy-btn" @click="copy(acct.password)"><el-icon :size="13"><Copy /></el-icon>复制密码</button>
              <el-button size="small" @click="pwHidden = true"><el-icon><EyeOff /></el-icon>&nbsp;我已保存，隐藏</el-button>
            </template>
          </div>
          <div style="display: flex; gap: 10px; align-items: center; margin-top: 14px">
            <el-button size="small" @click="openEntry(acct.dn)"><el-icon><PencilLine /></el-icon>&nbsp;编辑该账号（改名 / 改密）</el-button>
          </div>
          <div style="display: flex; gap: 10px; align-items: center; margin-top: 16px; flex-wrap: wrap">
            <b style="font-size: 13px">交付前自测：</b>
            <el-button type="primary" :loading="testing" @click="test(true)">
              <el-icon><PlugZap /></el-icon>&nbsp;验证这组 Bind 凭据
            </el-button>
            <span class="muted">真实 bind 一次——把凭据交给对方之前先确认它能通过目录认证</span>
            <a class="bad-demo" @click="test(false)">演示：用错误密码再测一次</a>
          </div>
          <div v-if="testOut !== null" class="nl-hint" :class="testOut.ok ? 'ok' : 'dgr'" style="margin-top: 12px">
            <el-icon style="margin-top: 2px"><component :is="testOut.ok ? CircleCheck : CircleX" /></el-icon>
            <div>{{ testOut.message }}</div>
          </div>
          <div class="muted" style="margin-top: 14px; display: flex; gap: 6px">
            <el-icon style="margin-top: 2px"><Info /></el-icon>
            <span>账号的"只读"由服务端 ACL 决定（展开上方样例 LDIF，按实际 DN 生成）。本工具不越权代改服务器 ACL。</span>
          </div>
        </template>
      </div>
    </div>

    <!-- 验证与 FAQ -->
    <div class="nl-card" style="margin-top: 14px">
      <div class="nl-card-head"><span class="t"><el-icon><MessageCircleQuestion /></el-icon>对方接入后 · 验证与排错</span></div>
      <div class="nl-card-body">
        <div class="nl-hint ok">
          <el-icon style="margin-top: 2px"><CircleCheck /></el-icon>
          <div><b>接入后 3 步验证：</b>① 用一名测试账号在对方系统登录一次 → ② 回本系统「审计日志」看这次 bind 的记录 → ③ 让对方触发一次用户同步，对比两边人员数。</div>
        </div>
        <el-collapse class="faq">
          <el-collapse-item v-for="(f, i) in FAQS" :key="i" :title="f.q" :name="i">
            <div class="fa" v-html="f.a"></div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Megaphone, Lightbulb, ClipboardCopy, Copy, Check, Info, UserCheck, TriangleAlert,
  UserPlus, Eye, EyeOff, KeyRound, PlugZap, CircleCheck, CircleX, MessageCircleQuestion, PencilLine,
  Server, ShieldCheck, Network, Fingerprint, Plug, User, FolderTree, UsersRound,
} from 'lucide-vue-next'
import { createBindAccount, bindTest, errText, me as apiMe, ouTree, type TreeNode } from '../../api'
import { TPLS, USER_FILTER, OU_FILTER, GROUP_FILTER, type Tpl } from '../../utils/integrations'
import { bumpTree, openEntry } from '../../store/app'

// integrations.ts / sections 里的 icon 是字符串名 → 映射到组件（字符串无法被 :is 解析）
const TICONS: Record<string, unknown> = { Server, ShieldCheck, Network, Fingerprint, Plug }
const SICONS: Record<string, unknown> = { Plug, KeyRound, User, FolderTree, UsersRound }

const LS_ADDR = 'nl.connAddr'

const addr = ref('')
const guessed = ref('')
const baseDN = ref('')
const tplKey = ref(TPLS[0].key)
const copied = ref('')

const acct = ref<{ dn: string; password: string } | null>(null)
const acctName = ref('')
const acctParent = ref('')
const pwHidden = ref(false)
const creating = ref(false)
const testing = ref(false)
const testOut = ref<{ ok: boolean; message: string } | null>(null)
const ouOptions = ref<Array<{ dn: string; name: string; depth: number }>>([])

const svcOU = computed(() => 'ou=services,' + baseDN.value)
const tpl = computed<Tpl>(() => TPLS.find((t) => t.key === tplKey.value) ?? TPLS[0])
const previewDN = computed(() => `cn=${acctName.value || tpl.value.acct},${acctParent.value || svcOU.value}`)
const aclLDIF = computed(() => `dn: olcDatabase={1}mdb,cn=config
changetype: modify
add: olcAccess
olcAccess: to dn.subtree="${baseDN || 'dc=…'}"
  by dn.exact="${previewDN}" read
  by * break
# 释义：允许该账号只读 Base DN 子树；写权限只有 admin 拥有。
# 账号本身条目：${previewDN}
#   objectClass: organizationalRole + simpleSecurityObject（标准 schema，无需扩展）
# 注意：add 若提示已存在请改用 replace: olcAccess 并合并现有规则——改 ACL 前先备份。`)

function switchTpl(k: string) {
  tplKey.value = k
  acct.value = null
  testOut.value = null
  acctName.value = TPLS.find((t) => t.key === k)?.acct ?? ''
  acctParent.value = svcOU.value
}

function saveAddr() { localStorage.setItem(LS_ADDR, addr.value) }

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = text
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    ta.remove()
  }
  copied.value = text
  ElMessage.success('已复制：' + (text.length > 46 ? text.slice(0, 46) + '…' : text))
  setTimeout(() => { if (copied.value === text) copied.value = '' }, 1600)
}

function copyAll() {
  const lines = ['# NewLDAP 对接信息（' + tpl.value.name + '）', '']
  for (const s of sections.value) {
    lines.push('── ' + s.title + ' ──')
    for (const r of s.rows) lines.push(r.k + '：' + r.v)
  }
  copy(lines.join('\n'))
}

interface Row { k: string; v: string; tip?: string; plain?: boolean; ghost?: boolean; copy?: boolean }
const sections = computed<Array<{ icon: string; title: string; note: string; rows: Row[] }>>(() => {
  const t = tpl.value
  const bindDN = acct.value?.dn
  const secs = [
    {
      icon: 'Plug', title: '连接 · 填在对方的「服务器 / 地址」处', note: '地址是目录服务器，不是本管理台',
      rows: [
        { k: '服务器地址', v: addr.value || '（先在上方确认通告地址）', tip: '对方产品能访问到的 OpenLDAP 地址。ldaps:// 开头 = 加密 636，ldap:// = 明文 389。', copy: !!addr.value, ghost: !addr.value },
        { k: 'Base DN / 搜索根', v: baseDN.value, tip: '搜索从这里开始。填本目录的根，用户和组都能搜到。', copy: true },
        { k: '加密方式', v: addr.value.startsWith('ldaps') ? 'LDAPS（636，推荐）' : '明文 389（建议改用 LDAPS）', tip: '加密方式取决于地址前缀，两边保持一致。', plain: true },
      ],
    },
    {
      icon: 'KeyRound', title: '认证 · 填在对方的「Bind DN / 绑定账号」处', note: '用下方创建的只读账号，不要用 admin',
      rows: [
        { k: 'Bind DN', v: bindDN ?? '（先在第 3 步创建只读账号）', tip: '对方系统用来登录目录查数据的账号。', copy: !!bindDN, ghost: !bindDN },
        { k: '密码', v: acct.value ? (pwHidden.value ? '••••••••（已隐藏）' : acct.value.password) : '（创建只读账号后生成）', tip: '账号的密码，只在创建那一刻完整显示过。', copy: !!acct.value && !pwHidden.value, ghost: !acct.value },
      ],
    },
    {
      icon: 'User', title: '用户映射 · 告诉对方"人员长什么样"', note: '「用户筛选器」+ 属性名对照',
      rows: [
        { k: '用户筛选器', v: USER_FILTER, tip: '只把人员条目筛出来，部门和组不会混入。', copy: true },
        { k: '用户名属性（登录用）', v: 'uid', tip: '员工用什么登录对方系统。本目录是 uid（如 zhangwei），不是邮箱、不是 AD 的 sAMAccountName。', copy: true },
        { k: '唯一标识', v: 'entryUUID', tip: '推荐：人员改名 / 换部门它都不变，对方不会重复建号。', copy: true },
        { k: '姓名 / 姓 / 邮箱', v: 'cn · sn · mail', tip: '姓名 cn、姓 sn、邮箱 mail，一般对方默认就是这三个。', copy: true },
        { k: '部门 / 工号 / 手机', v: 'ou · employeeNumber · mobile', tip: 'ou 的第二个值是部门中文名。', copy: true },
      ],
    },
  ]
  if (t.org) {
    secs.push({
      icon: 'FolderTree', title: '组织架构映射 · 告诉对方"部门树长什么样"', note: t.name + '支持组织同步时填写',
      rows: [
        { k: '组织筛选器', v: OU_FILTER, tip: '把部门条目筛出来。', copy: true },
        { k: '部门名属性', v: 'ou（多值：英文标识 + 中文名）', tip: '第一个 ou 值是英文标识（DN 用），第二个是中文名；对方若只有一个字段，建议用中文名。', copy: true },
        { k: '部门说明', v: 'description', tip: '部门描述字段，本管理台「编辑部门信息」里维护。', copy: true },
      ],
    })
  }
  if (t.grp) {
    secs.push({
      icon: 'UsersRound', title: '用户组映射 · 告诉对方"组怎么读"', note: '权限组与登录组的成员字段不同，注意区分',
      rows: [
        { k: '组筛选器', v: GROUP_FILTER, tip: '同时筛出权限组和登录组；只想要权限组就去掉 posixGroup 那段。', copy: true },
        { k: '组名属性', v: 'cn', tip: '组的名称字段。', copy: true },
        { k: '成员属性（权限组）', v: 'member（值为人员 DN）', tip: 'groupOfNames 的 member 存完整 DN，适合做授权判断。', copy: true },
        { k: '成员属性（登录组）', v: 'memberUid（值为 uid）', tip: 'posixGroup 的 memberUid 存 uid，Linux 场景用。', copy: true },
      ],
    })
  }
  return secs
})

async function create() {
  creating.value = true
  try {
    acct.value = await createBindAccount(acctName.value.trim() || tpl.value.acct, acctParent.value || svcOU.value)
    pwHidden.value = false
    testOut.value = null
    ElMessage.success('只读账号已创建（标准 LDAP Add），LAM 侧立即可见')
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    creating.value = false
  }
}

async function test(okMode: boolean) {
  if (!acct.value) return
  testing.value = true
  testOut.value = null
  try {
    if (okMode) {
      testOut.value = await bindTest(acct.value.dn, acct.value.password)
    } else {
      testOut.value = await bindTest(acct.value.dn, 'wrong-password-demo')
    }
  } catch (e) {
    testOut.value = { ok: false, message: errText(e) }
  } finally {
    testing.value = false
  }
}

const FAQS = [
  { q: '能连上，但对方"搜不到任何用户"', a: '<b>最常见原因：</b>Base DN 范围太窄（只填了某个部门），或对方默认用了 AD 的过滤器（含 sAMAccountName）。把信息卡里的 <span class="mono">Base DN</span> 与<b>用户筛选器</b>原样复制过去。可先在「LDAP 教学 → 过滤器实验器」跑一遍确认有结果。' },
  { q: '密码正确，但对方系统登录失败', a: '多半是对方「用户名属性」填错：本目录用 <span class="mono">uid</span> 登录（如 zhangwei），不是 AD 的 sAMAccountName，也不是邮箱前缀。核对信息卡「用户映射」一节。' },
  { q: '同步成功，但部门层级 / 中文名丢了', a: '对方未启用「组织架构同步」。在对方的组织（机构）映射里填：过滤器 <span class="mono">(objectClass=organizationalUnit)</span>、部门名 <span class="mono">ou</span>；本目录部门的中文名存在第二个 ou 值里。' },
  { q: '员工改了密码 / 换了部门，对方系统没生效', a: '对方多半有账号缓存，属正常现象：把对方缓存 TTL 调短（如 5 分钟），或在其后台手动触发同步。本目录侧的修改即时生效——目录即真相，没有中间库。' },
  { q: '为什么不要把 admin 的密码填给对方？', a: '对方系统会长期持有这组凭据，等于把整本目录的读写权交出去。用本页创建的<b>专用只读账号</b>：权限最小（配合 ACL 样例仅可读）、可随时改密/停用、出问题能精确审计到是谁在查。' },
]

function flatten(nodes: TreeNode[], depth = 0): Array<{ dn: string; name: string; depth: number }> {
  const out: Array<{ dn: string; name: string; depth: number }> = []
  for (const n of nodes) {
    if (n.dn.toLowerCase().startsWith('ou=')) {
      const disp = (n as { displayName?: string }).displayName
      out.push({ dn: n.dn, name: disp || n.name, depth })
      if (n.children?.length) out.push(...flatten(n.children, depth + 1))
    }
  }
  return out
}

onMounted(async () => {
  guessed.value = `ldap://${window.location.hostname}:389`
  const saved = localStorage.getItem(LS_ADDR)
  addr.value = saved || guessed.value
  try {
    const m = await apiMe()
    baseDN.value = m.baseDN
    acctParent.value = 'ou=services,' + m.baseDN
  } catch { /* 忽略 */ }
  acctName.value = tpl.value.acct
  try {
    ouOptions.value = flatten(await ouTree())
  } catch { /* 忽略：位置下拉只剩 services/根 */ }
})

// 创建账号后刷新左侧目录树（ou=services 可能是新建的）
watch(acct, (n, o) => { if (n && n.dn !== o?.dn) bumpTree() })
</script>

<style scoped>
.tpl-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px; }
@media (max-width: 1250px) { .tpl-grid { grid-template-columns: repeat(3, 1fr); } }
.tpl { border: 1.5px solid var(--el-border-color); border-radius: 8px; padding: 13px 14px; cursor: pointer; transition: 0.15s; display: flex; flex-direction: column; gap: 6px; background: #fff; }
.tpl:hover { border-color: #adc6ff; }
.tpl.on { border-color: var(--el-color-primary); background: #f0f5ff; box-shadow: 0 0 0 3px #f0f5ff; }
.tpl b { font-size: 13px; }
.tpl p { font-size: 11.5px; color: #646a73; margin: 0; line-height: 1.55; }
.tpl .muted { font-size: 10.5px; }
.icard { display: flex; flex-direction: column; gap: 16px; }
.icard .sec { border: 1px solid var(--el-border-color); border-radius: 8px; overflow: hidden; }
.icard .sec-h { font-size: 12.5px; font-weight: 600; margin: 0; padding: 10px 14px; background: #fafbfc; border-bottom: 1px solid var(--el-border-color-lighter); display: flex; align-items: center; gap: 8px; color: #1f2329; }
.icard .sec-h .el-icon { color: var(--el-color-primary); }
.icard .sec-h .m { font-weight: 400; color: #8f959e; font-size: 11px; margin-left: auto; }
.icard .row { display: flex; align-items: center; gap: 12px; padding: 8px 14px; border-bottom: 1px solid var(--el-border-color-extra-light); font-size: 12.5px; }
.icard .row:last-child { border-bottom: 0; }
.icard .row:hover { background: #fafbfc; }
.icard .k { width: 176px; flex: none; color: #646a73; }
.term-q { border-bottom: 1px dashed #a8abb2; cursor: help; }
.icard .v { flex: 1; font-family: Consolas, monospace; font-size: 12px; word-break: break-all; }
.icard .v.plain { font-family: inherit; font-size: 12.5px; color: #646a73; }
.icard .v.ghost { font-family: inherit; font-size: 12px; color: #8f959e; font-style: italic; }
.copy-btn { flex: none; display: inline-flex; align-items: center; gap: 4px; border: 1px solid var(--el-border-color); background: #fff; border-radius: 6px; height: 24px; padding: 0 8px; font-size: 11px; color: #646a73; cursor: pointer; transition: 0.12s; white-space: nowrap; }
.copy-btn:hover { border-color: var(--el-color-primary); color: var(--el-color-primary); }
.copy-btn.ok { border-color: #67c23a; color: #67c23a; }
.pw-once { display: flex; align-items: center; gap: 12px; background: var(--nl-warn-bg, #fdf6ec); border: 1px solid #f5d5b0; border-radius: 8px; padding: 12px 14px; flex-wrap: wrap; margin-top: 12px; }
.pw-once .pw { font-family: Consolas, monospace; font-size: 15px; font-weight: 700; letter-spacing: 1px; color: #a24e08; }
.bad-demo { color: var(--el-color-primary); cursor: pointer; font-size: 11.5px; text-decoration: underline dotted; }
.acl-t { color: var(--el-color-primary); font-size: 12.5px; }
.ldif { background: #101a3e; color: #d6defa; border-radius: 8px; padding: 14px 16px; font-family: Consolas, monospace; font-size: 12px; line-height: 1.8; overflow: auto; white-space: pre; margin: 0; }
:deep(.faq .el-collapse-item__header) { font-size: 13px; font-weight: 500; }
:deep(.faq .fa) { font-size: 12.5px; color: #646a73; line-height: 1.8; }
:deep(.faq .mono) { font-family: Consolas, monospace; font-size: 11.5px; background: #f5f6f8; border-radius: 4px; padding: 1px 5px; }
</style>
