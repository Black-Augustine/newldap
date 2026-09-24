<template>
  <div class="admin">
    <header class="topbar">
      <AppLogo :size="30" />
      <div class="brand">NewLDAP<span class="sub">目录管理台</span></div>
      <div class="top-search">
        <el-input v-model="kw" placeholder="搜索人员、部门、用户组…" clearable @keyup.enter="doSearch" @clear="doSearch">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
      </div>
      <div class="grow" />
      <div class="seg">
        <button :class="{ on: !app.expertMode }" @click="app.expertMode = false">
          <el-icon><Users /></el-icon>简单模式
        </button>
        <button :class="{ on: app.expertMode }" @click="app.expertMode = true">
          <el-icon><Braces /></el-icon>专家模式
        </button>
      </div>
      <!-- 帮助入口收敛到侧栏「帮助中心」页（原顶栏 ? 下拉与其重复，已移除） -->
      <el-dropdown trigger="click">
        <div class="user" :title="me?.bindDN ?? ''">
          <span class="nl-avatar">管</span>
          <div class="dn"><b>管理员</b><span class="mono">{{ shortBindDN }}</span></div>
          <el-icon color="#8F959E"><ChevronDown /></el-icon>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item @click="$router.push('/password')">修改我的密码</el-dropdown-item>
            <el-dropdown-item @click="$router.push('/settings')">连接设置</el-dropdown-item>
            <el-dropdown-item divided @click="doLogout">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </header>

    <div class="shell">
      <aside class="sidebar" :class="{ collapsed }">
        <nav class="nav">
          <router-link v-for="n in navItems" :key="n.to" :to="n.to" class="item" :class="{ on: isActive(n.to) }">
            <el-icon :size="18"><component :is="n.icon" /></el-icon>{{ n.label }}
          </router-link>
        </nav>
        <div class="tree-cap">
          <span>目录树（DIT）</span>
          <span class="ops">
            <el-button id="tour-new-ou" circle size="small" @click="openCreateOU" title="新建部门（在当前选中节点下；未选中时建在根下）"><el-icon><FolderPlus /></el-icon></el-button>
            <el-button circle size="small" @click="bumpTree()" title="刷新"><el-icon><RefreshCw /></el-icon></el-button>
          </span>
        </div>
        <div class="tree-panel">
          <el-tree :key="app.treeVersion" :props="treeProps" node-key="dn" lazy :load="loadNode"
            highlight-current :expand-on-click-node="false" default-expand-all
            draggable :allow-drag="allowDrag" :allow-drop="allowDrop" @node-drop="onDrop"
            @node-click="onNode">
            <template #default="{ data }">
              <span class="tnode" :title="data.dn">
                <el-icon :size="13" :color="iconColor(data)" style="flex: none">
                  <component :is="nodeIcon(data)" />
                </el-icon>
                <span v-if="data.isRoot" class="rdn root-node">全部（{{ data.dn }}）</span>
                <template v-else>
                  <span class="rdn mono" :class="{ person: isPerson(data) }">{{ data.rdn }}</span>
                  <span v-if="data.displayName" class="rdn-cn">{{ data.displayName }}</span>
                </template>
                <span class="count" v-if="data.childCount > 0">{{ data.childCount }}</span>
                <!-- OU 行内新建人员：预选该部门 -->
                <el-button v-if="data.rdn?.startsWith('ou=')" class="row-add" text size="small"
                  title="在此部门新建人员" @click.stop="openPersonCreate(data.dn)">
                  <el-icon :size="13"><UserPlus /></el-icon>
                </el-button>
              </span>
            </template>
          </el-tree>
        </div>
        <div class="side-foot">
          <div class="st"><span class="dot" />已连接 · {{ profile?.profile?.url || '…' }}</div>
          <div>{{ connSecLabel }}</div>
          <!-- 桥接页自动代填管理员账密登录 phpLDAPadmin（GET /api/v1/plda/bridge）；档案未存密码时桥接页会给出手动入口 -->
          <a v-if="pldaURL" class="plda-btn" :class="{ down: pldaDown }" href="/api/v1/plda/bridge"
            target="_blank" rel="noopener"
            :title="pldaDown ? '探测不到 phpLDAPadmin（可能未启动）；点击仍会尝试自动登录' : '新标签页打开 phpLDAPadmin（自动代填管理员账密），交叉验证目录内容'">
            <el-icon :size="13" style="margin-right: 5px"><ExternalLink /></el-icon>
            phpLDAPadmin 交叉验证
            <span v-if="pldaDown" class="plda-warn">未启动</span>
          </a>
        </div>
      </aside>
      <div class="collapser" @click="collapsed = !collapsed" :title="collapsed ? '展开侧栏' : '收起侧栏'">
        <el-icon><ChevronsLeft v-if="!collapsed" /><ChevronsRight v-else /></el-icon>
      </div>
      <main class="content">
        <router-view />
      </main>
      <EntryDrawer :open="!!entryDrawerDN" :dn="entryDrawerDN" @close="entryDrawerDN = ''" @saved="onDrawerSaved" />
    </div>

    <!-- 全局新建人员（树行按钮与人员页共用；默认部门取打开时传入的 OU） -->
    <NewPersonDialog :open="app.personCreateOpen" :default-ou="app.personCreateOU"
      @close="app.personCreateOpen = false" @created="onPersonCreated" />

    <!-- 新手聚光灯导览（帮助中心触发 / 首次登录自动出现一次） -->
    <CoachMark v-model="tourOpen" />

    <!-- 新建部门（树标题栏入口）：上级可选——选「根（顶层）」即创建顶级 OU -->
    <el-dialog v-model="ouDlg" title="新建部门" width="460px">
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="上级部门 *" required>
          <el-tree-select v-model="ouParentSel" :data="ouParentOptions" node-key="dn" check-strictly
            :props="{ label: 'name' }" default-expand-all :render-after-expand="false"
            style="width: 100%" placeholder="选择上级（根（顶层）= 顶级部门）" />
        </el-form-item>
        <el-form-item label="部门名称 *" required>
          <el-input v-model="ouForm.name" placeholder="如：tech（对应 ou=tech）" @keyup.enter="createOU" />
        </el-form-item>
        <el-form-item label="中文名（选填）">
          <el-input v-model="ouForm.cn" placeholder="如：技术部（目录树中显示在英文名旁）" />
        </el-form-item>
        <el-form-item label="说明（选填）">
          <el-input v-model="ouForm.desc" placeholder="部门职责一句话" />
        </el-form-item>
        <div class="nl-hint info">
          <el-icon style="margin-top: 2px"><CircleHelp /></el-icon>
          <div>部门是目录树中的 OU 节点，可多层嵌套；选「根（顶层）」可创建顶级部门。</div>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="ouDlg = false">取消</el-button>
        <el-button type="primary" :loading="creatingOU" :disabled="!ouForm.name.trim() || !ouParentSel" @click="createOU">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  ArrowUpDown, Braces, Building2, ChevronDown, ChevronsLeft, ChevronsRight, CircleHelp,
  ClipboardList, ExternalLink, FileText, Folder, FolderPlus, Home, RefreshCw, Search, Settings,
  User, UserPlus, Users, UsersRound,
} from 'lucide-vue-next'
import AppLogo from '../components/AppLogo.vue'
import NewPersonDialog from '../components/NewPersonDialog.vue'
import CoachMark from '../components/CoachMark.vue'
import { createEntry, domainFromBaseDN, errText, getPolicy, healthz, logout, me as apiMe, ouTree, setupStatus, tree, updateEntry, type Me, type SetupStatus, type TreeNode } from '../api'
import EntryDrawer from '../components/EntryDrawer.vue'
import { app, baseDomain, bumpPeople, bumpTree, entryDrawerDN, loadPolicyOnce, openEntry, openPersonCreate, searchQuery, setSelected, tourOpen } from '../store/app'

const router = useRouter()
const route = useRoute()
const me = ref<Me | null>(null)
const profile = ref<SetupStatus | null>(null)
const health = ref<{ ldap?: string; mock?: boolean; pldaURL?: string } | null>(null)
const collapsed = ref(false)
const kw = ref('')
const pldaURL = computed(() => health.value?.pldaURL ?? '')
// 连接安全文案跟随实际地址/档案（此前写死"LDAPS/加密连接"，与 ldap:// 矛盾）
const connSecLabel = computed(() => {
  if (health.value?.mock) return '内置演示目录（--mock）'
  const url = (profile.value?.profile?.url ?? '').toLowerCase()
  if (url.startsWith('ldaps://')) return 'LDAPS 加密连接'
  if (url.startsWith('ldap://')) return '明文连接（建议改用 LDAPS）'
  return '外部目录'
})
const pldaDown = ref(false)

// 顶栏只显示 RDN 值（admin），完整 DN 悬停提示
const shortBindDN = computed(() => {
  const dn = me.value?.bindDN ?? ''
  const rdn = dn.split(',')[0] ?? ''
  const i = rdn.indexOf('=')
  return i > 0 ? rdn.slice(i + 1) : (rdn || dn)
})

function onPersonCreated(dn: string) {
  bumpTree()
  bumpPeople() // 人员页若在挂载则刷新列表
}

// 新建部门（树标题栏）：上级可选（根 = 顶级 OU），默认定位到当前选中的 OU
const ouDlg = ref(false)
const creatingOU = ref(false)
const ouForm = reactive({ name: '', cn: '', desc: '' })
const ouParentSel = ref('')
const ouParentOptions = ref<TreeNode[]>([])

async function openCreateOU() {
  ouForm.name = ''
  ouForm.cn = ''
  ouForm.desc = ''
  // 先备好选项再开窗设值：选项未就绪时 tree-select 会清空找不到节点的预设值
  try {
    const ous = await ouTree()
    ouParentOptions.value = me.value?.baseDN
      ? [{ dn: me.value.baseDN, rdn: me.value.baseDN.split(',')[0], name: '根（顶层）', objectClass: [], hasChildren: true, childCount: 0, children: ous }]
      : ous
  } catch { ouParentOptions.value = [] }
  ouParentSel.value = app.selectedOU?.rdn?.startsWith('ou=') ? app.selectedOU.dn : (me.value?.baseDN ?? '')
  ouDlg.value = true
}

async function createOU() {
  const name = ouForm.name.trim()
  if (!name || name.includes(',') || name.includes('=')) {
    ElMessage.warning('部门名称不能为空，且不能包含 , 或 =')
    return
  }
  if (!ouParentSel.value) {
    ElMessage.warning('请选择上级部门（根（顶层）= 顶级部门）')
    return
  }
  creatingOU.value = true
  try {
    const attrs: Record<string, string[]> = { ou: [name] }
    if (ouForm.cn.trim() && ouForm.cn.trim() !== name) attrs.ou = [name, ouForm.cn.trim()]
    if (ouForm.desc.trim()) attrs.description = [ouForm.desc.trim()]
    await createEntry(`ou=${name},${ouParentSel.value}`, ['organizationalUnit'], attrs)
    ElMessage.success(`已创建部门：ou=${name}`)
    ouDlg.value = false
    bumpTree()
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    creatingOU.value = false
  }
}

const navItems = [
  { to: '/overview', label: '概览', icon: Home },
  { to: '/people', label: '组织与人员', icon: Users },
  { to: '/groups', label: '用户组', icon: UsersRound },
  { to: '/import', label: '导入导出', icon: ArrowUpDown },
  { to: '/audit', label: '审计日志', icon: ClipboardList },
  { to: '/settings', label: '系统设置', icon: Settings },
  { to: '/help', label: '帮助中心', icon: CircleHelp },
]

function isActive(to: string) { return route.path === to }

const treeProps = { label: 'name', children: 'children', isLeaf: (d: any) => !d.hasChildren }
async function loadNode(node: any, resolve: (d: any[]) => void) {
  if (node.level === 0) {
    // 根节点（Base DN）：点它 = 查看全部员工
    const base = me.value?.baseDN ?? ''
    resolve([{ dn: base, rdn: base.split(',')[0] ?? base, name: '全部', isRoot: true,
      objectClass: [], hasChildren: true, childCount: 0 }])
    return
  }
  try {
    // containers 模式：树只呈现部门/组等容器（人员行不出现），
    // hasChildren 按"有无下级容器"统计 → 没有下级的 OU 不显示展开箭头
    resolve(await tree(node.data.dn, { containers: true }))
  } catch { resolve([]) }
}

// ---- 拖拽排序：同层 OU 之间拖动，顺序写入 ouOrder 属性（目录即真相，LAM 可见） ----
function allowDrag(node: any) {
  return node.data?.rdn?.startsWith('ou=')
}
function allowDrop(drag: any, drop: any, type: string) {
  return type !== 'inner' && drop.data?.rdn?.startsWith('ou=') && drag.data.dn !== drop.data.dn
}
async function onDrop(drag: any, drop: any) {
  // 拖动节点的新兄弟序列 = 父节点展开后的子节点顺序
  const siblings: any[] = drop.parent?.childNodes?.map((n: any) => n.data) ?? []
  if (!siblings.length) return
  try {
    await Promise.all(siblings.map((d, i) =>
      updateEntry(d.dn, '', [{ op: 'replace', attr: 'ouOrder', vals: [String((i + 1) * 10)] }])))
    ElMessage.success('部门顺序已保存（写入 ouOrder 属性）')
    bumpTree()
  } catch (e) {
    ElMessage.error('保存顺序失败：' + errText(e) + '（真实 OpenLDAP 需在 schema 中允许 ouOrder 属性）')
  }
}

function onNode(data: TreeNode) {
  // 根节点 → 查看全部员工
  if ((data as any).isRoot) {
    setSelected(null)
    if (route.path !== '/people') router.push('/people')
    return
  }
  const cls = data.objectClass ?? []
  // 只有部门（organizationalUnit）走"选中部门→人员列表"；
  // 组、角色/服务账号等其余条目一律打开条目抽屉（可查看/编辑）
  if (!cls.includes('organizationalUnit')) {
    openEntry(data.dn)
    return
  }
  setSelected(data)
  if (route.path !== '/people') router.push('/people')
}

function isPerson(d: TreeNode) { return (d.objectClass ?? []).includes('inetOrgPerson') }
function nodeIcon(d: TreeNode) {
  const cls = d.objectClass ?? []
  if (cls.includes('inetOrgPerson')) return User
  if (cls.includes('groupOfNames') || cls.includes('posixGroup')) return UsersRound
  if (d.rdn?.startsWith('ou=')) return Folder
  if (d.rdn?.startsWith('dc=')) return Building2
  return FileText
}
function iconColor(d: TreeNode) {
  const cls = d.objectClass ?? []
  if (cls.includes('inetOrgPerson')) return '#2F54EB'
  if (cls.includes('groupOfNames') || cls.includes('posixGroup')) return '#D46B08'
  return '#8F959E'
}

function doSearch() {
  searchQuery.value = kw.value.trim() // 空值 = 清空搜索结果
  if (route.path !== '/people') router.push('/people')
}

// 抽屉保存回调：刷新树与人员列表（禁用/启用等操作后表格状态列同步）；
// 若简单模式改了账号（重命名导致 DN 变化），抽屉指向新 DN
function onDrawerSaved(dn: string) {
  bumpTree()
  bumpPeople()
  if (dn && dn !== entryDrawerDN.value) entryDrawerDN.value = dn
}

function term(what: string) {
  const map: Record<string, string> = {
    DN: 'DN = 条目在目录树中的完整路径，像收快递的详细地址，全局唯一。',
    OU: 'OU = 目录树里的"部门"节点，可以多层嵌套，像公司的组织架构。',
    objectClass: 'objectClass = 条目的类型模板，决定它有哪些字段（如人员 = inetOrgPerson）。',
  }
  ElMessage.info(map[what] ?? '')
}

async function doLogout() {
  try { await logout() } catch { /* 忽略 */ }
  router.push('/')
}

onMounted(async () => {
  // 首次登录自动导览（本地记忆，打扰一次即可；开关是全局 store 的 tourOpen）
  if (!localStorage.getItem('nl.tour.done')) {
    localStorage.setItem('nl.tour.done', '1')
    setTimeout(() => { tourOpen.value = true }, 600)
  }
  try {
    me.value = await apiMe()
    if (me.value?.baseDN) {
      baseDomain.value = domainFromBaseDN(me.value.baseDN)
    }
    loadPolicyOnce()
    bumpTree() // 根节点依赖 baseDN；me 就绪后重挂树（首载时树可能先于 me 渲染）
    profile.value = await setupStatus()
    health.value = await healthz()
  } catch (e) {
    ElMessage.error(errText(e))
    router.push('/')
  }
  // phpLDAPadmin 可达性探测（只影响按钮上的「未启动」提示，不拦截点击）：
  // 走同源反代 /plda/（与用户点击后实际走的路径一致），避免跨域 fetch 被浏览器拦截
  if (pldaURL.value) {
    try {
      const ctl = new AbortController()
      setTimeout(() => ctl.abort(), 3000)
      await fetch('/plda/index.php', { signal: ctl.signal })
      pldaDown.value = false
    } catch {
      pldaDown.value = true
    }
  }
})

watch(() => app.treeVersion, () => { /* 树由 :key 重挂载 */ })
</script>

<style scoped>
.admin { height: 100vh; display: flex; flex-direction: column; }
.topbar { height: 56px; flex: none; background: #fff; border-bottom: 1px solid var(--el-border-color); display: flex; align-items: center; gap: 14px; padding: 0 16px; z-index: 20; }
.brand { color: #1f2329; font-weight: 600; font-size: 14.5px; display: flex; align-items: baseline; gap: 8px; }
.brand .sub { font-size: 11px; color: #8f959e; font-weight: 400; }
.top-search { width: 320px; }
.top-search :deep(.el-input__wrapper) { border-radius: 17px; background: #f2f3f5; box-shadow: none; }
.top-search :deep(.el-input__wrapper.is-focus) { background: #fff; box-shadow: 0 0 0 1px var(--el-color-primary) inset; }
.grow { flex: 1; }
.seg { display: inline-flex; background: #f2f3f5; border-radius: 8px; padding: 2px; gap: 2px; }
.seg button { display: inline-flex; align-items: center; gap: 5px; border: 0; background: transparent; height: 26px; padding: 0 12px; border-radius: 6px; font-size: 12px; color: #646a73; cursor: pointer; }
.seg button.on { background: #fff; color: var(--el-color-primary); box-shadow: 0 1px 3px rgba(31, 35, 41, 0.12); font-weight: 500; }
.icon-btn { border: 0; }
.user { display: flex; align-items: center; gap: 8px; padding: 4px 8px; border-radius: 8px; cursor: pointer; }
.user:hover { background: #f2f3f5; }
.user .dn { font-size: 12px; color: #646a73; text-align: right; line-height: 1.3; }
.user .dn b { display: block; font-size: 12.5px; color: #1f2329; font-weight: 600; }
.user .dn .mono { font-size: 10.5px; max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; display: block; }

.shell { display: flex; flex: 1; min-height: 0; }
.sidebar { width: 248px; flex: none; background: #fff; border-right: 1px solid var(--el-border-color); display: flex; flex-direction: column; overflow: hidden; transition: margin-left 0.2s; }
.sidebar.collapsed { margin-left: -248px; }
.nav { padding: 10px 10px 4px; display: flex; flex-direction: column; gap: 2px; }
.nav .item { display: flex; align-items: center; gap: 10px; height: 36px; padding: 0 10px; border-radius: 8px; color: #646a73; font-size: 13px; text-decoration: none; transition: 0.12s; }
.nav .item:hover { background: #f2f3f5; color: #1f2329; }
.nav .item.on { background: #f0f5ff; color: var(--el-color-primary); font-weight: 500; }
.tree-cap { font-size: 11px; color: #8f959e; padding: 12px 12px 6px; display: flex; align-items: center; justify-content: space-between; }
.tree-cap .ops { display: flex; }
.tree-panel { flex: 1; overflow: auto; padding: 0 10px 12px; }
.side-foot { flex: none; border-top: 1px solid var(--el-border-color-lighter); padding: 10px 14px; font-size: 11.5px; color: #8f959e; line-height: 1.7; }
.side-foot .st { display: flex; align-items: center; gap: 6px; color: #646a73; }
.side-foot .dot { width: 7px; height: 7px; border-radius: 50%; background: var(--nl-ok); }
.collapser { width: 14px; flex: none; display: flex; align-items: center; justify-content: center; cursor: pointer; color: #8f959e; background: #fff; border-right: 1px solid var(--el-border-color-lighter); }
.collapser:hover { color: var(--el-color-primary); }
.content { flex: 1; min-width: 0; overflow: auto; padding: 18px 20px 26px; }
.tnode { display: inline-flex; align-items: center; gap: 6px; max-width: 100%; }
.tnode .rdn { font-family: ui-monospace, Consolas, monospace; font-size: 12px; color: #1f2329; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tnode .rdn.person { color: var(--el-color-primary); }
.tnode .rdn.root-node { color: #1f2329; font-weight: 500; font-family: inherit; }
.tnode .rdn-cn { font-size: 11.5px; color: #8f959e; }
.tnode .row-add { padding: 2px; height: 20px; margin-left: 2px; color: #8f959e; flex: none; }
.tnode .row-add:hover { color: var(--el-color-primary); background: #f0f5ff; }
.count { font-size: 11px; color: #8f959e; background: #f2f3f8; border-radius: 8px; padding: 0 6px; }
.plda-btn { display: inline-flex; align-items: center; justify-content: center; width: calc(100% - 2px); height: 24px; margin-top: 8px; border: 1px solid var(--el-border-color); border-radius: var(--el-border-radius-base); font-size: 12px; color: #646a73; text-decoration: none; background: var(--el-fill-color-blank); }
.plda-btn:hover { color: var(--el-color-primary); border-color: var(--el-color-primary-light-5); }
.plda-btn.down { color: #8f959e; border-style: dashed; }
.plda-warn { font-size: 10.5px; color: #d46b08; background: #fff7e6; border-radius: 8px; padding: 0 5px; margin-left: 6px; }
</style>
