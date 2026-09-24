<template>
  <div style="max-width: 1280px; margin: 0 auto">
    <!-- 搜索结果模式 -->
    <template v-if="searchResult">
      <div class="nl-page-head">
        <div>
          <h2>搜索“{{ searchedKw }}”</h2>
          <div class="crumb">共 {{ searchResult.total }} 条 · 关键字全子树搜索</div>
        </div>
        <div class="ops"><el-button @click="clearSearch()">返回</el-button></div>
      </div>
      <div class="nl-card nl-table-wrap">
        <el-table :data="searchRows" v-loading="searching" stripe>
          <el-table-column label="名称" width="180">
            <template #default="{ row }">
              <div style="display: flex; align-items: center; gap: 10px">
                <span class="nl-avatar" v-if="row.isPerson">{{ (row.name || '?')[0] }}</span>
                <span class="nl-avatar gray" v-else><el-icon :size="13"><Folder /></el-icon></span>
                <span>{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="类型" width="100">
            <template #default="{ row }">
              <span class="nl-pill" :class="row.isPerson ? 'info' : 'plain'">{{ row.isPerson ? '人员' : '部门' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="DN">
            <template #default="{ row }"><span class="mono" style="font-size: 11.5px">{{ row.dn }}</span></template>
          </el-table-column>
          <el-table-column label="操作" width="110" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.isPerson" size="small" link type="primary" @click="openPerson(row.dn)">编辑</el-button>
              <el-button size="small" link type="primary" @click="locate(row.dn)">在树中定位</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </template>

    <!-- 正常浏览模式 -->
    <template v-else>
      <div class="nl-page-head">
        <div>
          <h2>{{ app.selectedOU?.name ?? '根' }}</h2>
          <div class="crumb"><span class="mono">{{ currentDN || me?.baseDN }}</span> · {{ people.length }} 名人员<template v-if="scopeSub">（含子部门）</template></div>
        </div>
        <div class="ops">
          <el-button type="primary" @click="openPersonCreate(currentDN)"><el-icon><UserPlus /></el-icon>&nbsp;新建人员</el-button>
          <el-button @click="openCreateOU"><el-icon><FolderPlus /></el-icon>&nbsp;新建子部门</el-button>
          <el-dropdown>
            <el-button><el-icon><Ellipsis /></el-icon>&nbsp;更多<el-icon style="margin-left:4px"><ChevronDown /></el-icon></el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item :disabled="!app.selectedOU" @click="openEditOU"><el-icon><PencilLine /></el-icon>&nbsp;编辑部门信息</el-dropdown-item>
                <el-dropdown-item :disabled="!app.selectedOU" @click="openRename"><el-icon><PencilLine /></el-icon>&nbsp;重命名部门</el-dropdown-item>
                <el-dropdown-item :disabled="!app.selectedOU" @click="moveDlg = true"><el-icon><ArrowRightLeft /></el-icon>&nbsp;移动部门</el-dropdown-item>
                <el-dropdown-item @click="exportCurrent"><el-icon><Download /></el-icon>&nbsp;导出本部门人员</el-dropdown-item>
                <el-dropdown-item divided :disabled="!app.selectedOU" @click="openDelete">
                  <span style="color: var(--nl-dgr)"><el-icon><Trash2 /></el-icon>&nbsp;删除部门</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      <el-alert v-if="conflictHit" type="error" show-icon :closable="false" style="margin-bottom: 12px"
        title="检测到编辑冲突（entryCSN），已被拦截——他人（如 LAM）可能刚修改过该条目，请刷新后重试" />
      <div class="nl-card nl-table-wrap">
        <div class="nl-toolbar">
          <el-input v-model="filter" placeholder="在本部门内筛选姓名 / 账号 / 工号 / 邮箱" clearable style="width: 240px">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-radio-group v-model="scope">
            <el-radio-button value="sub">含子部门</el-radio-button>
            <el-radio-button value="one">仅本部门</el-radio-button>
          </el-radio-group>
          <el-button type="primary" plain :disabled="!selectedPeople.length" @click="openMovePeople()">
            <el-icon><ArrowRightLeft /></el-icon>&nbsp;移动到部门{{ selectedPeople.length ? `（${selectedPeople.length} 人）` : '' }}
          </el-button>
          <span class="spacer" />
          <!-- 自定义列 -->
          <el-popover width="220" trigger="click">
            <template #reference>
              <el-button text><el-icon><Columns3 /></el-icon>&nbsp;自定义列</el-button>
            </template>
            <el-checkbox-group v-model="visibleKeys">
            <div class="col-order-list">
              <div v-for="(c, i) in orderedCols" :key="c.key" class="col-order-row">
                <el-checkbox :value="c.key" :label="c.key">
                  {{ c.label }} <span class="muted" style="font-size: 11px">{{ c.en }}</span>
                </el-checkbox>
                <span class="ord-ops">
                  <el-button text size="small" :disabled="i === 0" @click="moveCol(i, -1)">▲</el-button>
                  <el-button text size="small" :disabled="i === orderedCols.length - 1" @click="moveCol(i, 1)">▼</el-button>
                </span>
              </div>
            </div>
            </el-checkbox-group>
          </el-popover>
          <span class="muted">共 {{ filtered.length }} 人</span>
        </div>
        <el-table :data="filtered" v-loading="loading" stripe border @selection-change="onSelChange">
          <el-table-column type="selection" width="42" />
          <el-table-column label="姓名 (cn)" min-width="150">
            <template #default="{ row }">
              <div style="display: flex; align-items: center; gap: 10px; cursor: pointer" @click="openPerson(row.dn)">
                <span class="nl-avatar">{{ (row.cn || '?')[0] }}</span>
                <div>
                  <div>{{ row.cn }}</div>
                  <div class="muted" style="font-size: 11.5px">{{ row.title || '' }}</div>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column v-for="c in visibleCols" :key="c.key" :width="c.width">
            <template #header><span class="th-cn">{{ c.label }} <span class="th-en">{{ c.en }}</span></span></template>
            <template #default="{ row }"><span class="mono" style="font-size: 12px">{{ row[c.key] || '—' }}</span></template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <span class="nl-pill" :class="row.disabled ? 'plain' : 'ok'"><span class="dot" />{{ row.disabled ? '已禁用' : '在职' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button size="small" link type="primary" @click="openPerson(row.dn)">编辑</el-button>
              <el-button size="small" link type="danger" @click="delPerson(row)">删除</el-button>
            </template>
          </el-table-column>
          <template #empty>
            <div class="nl-empty">
              <el-empty :image-size="72" description="该部门暂无人员">
                <el-button type="primary" @click="openPersonCreate(currentDN)">新建人员</el-button>
                <el-button @click="$router.push('/import')">用 Excel 批量导入</el-button>
              </el-empty>
            </div>
          </template>
        </el-table>
      </div>
    </template>

    <!-- 编辑部门信息（中文名/说明） -->
    <el-dialog v-model="editOUDlg" title="编辑部门信息" width="480px">
      <el-form label-position="top">
        <el-form-item label="部门名称（ou）">
          <el-input :model-value="app.selectedOU?.name" readonly class="mono" />
          <div class="muted" style="font-size: 11px">英文名是目录标识，修改请用「重命名部门」</div>
        </el-form-item>
        <el-form-item label="中文名（目录树中显示）">
          <el-input v-model="editOUForm.cn" placeholder="如：技术部（留空清除）" />
        </el-form-item>
        <el-form-item label="说明 (description)">
          <el-input v-model="editOUForm.desc" type="textarea" :rows="2" placeholder="部门职责一句话" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editOUDlg = false">取消</el-button>
        <el-button type="primary" :loading="editOUBusy" @click="saveEditOU">保存</el-button>
      </template>
    </el-dialog>

    <!-- 批量移动人员到其他部门 -->
    <el-dialog v-model="movePeopleDlg" title="移动人员到其他部门" width="480px">
      <p style="margin: 0 0 12px">已选 <b>{{ selectedPeople.length }}</b> 名人员：
        <span class="mono" style="font-size: 11.5px">{{ selectedPeople.map(p => p.uid || p.cn).slice(0, 5).join('、') }}{{ selectedPeople.length > 5 ? ' 等' : '' }}</span>
      </p>
      <el-form label-position="top">
        <el-form-item label="目标部门 *" required>
          <el-tree-select v-model="movePeopleTarget" :data="ouOptions" node-key="dn" check-strictly
            :props="{ label: 'name' }" default-expand-all :render-after-expand="false"
            style="width: 100%" placeholder="选择目标部门" />
        </el-form-item>
      </el-form>
      <div class="nl-hint info">
        <el-icon style="margin-top: 2px"><Info /></el-icon>
        <div>移动 = 标准 LDAP modrdn 操作，账号、密码、组成员关系保持不变，仅 DN 中的部门部分更新。</div>
      </div>
      <template #footer>
        <el-button @click="movePeopleDlg = false">取消</el-button>
        <el-button type="primary" :loading="movingPeople" :disabled="!movePeopleTarget" @click="doMovePeople">
          移动 {{ selectedPeople.length }} 人
        </el-button>
      </template>
    </el-dialog>

    <!-- OU 对话框们（新建/重命名/移动/删除）沿用 M1 逻辑 -->
    <el-dialog v-model="ouDlg" title="新建子部门（OU）" width="460px">
      <el-form label-position="top">
        <el-form-item label="部门名称" required>
          <el-input v-model="ouForm.name" placeholder="例如：研发中心" @keyup.enter="createOU" />
        </el-form-item>
        <el-form-item label="描述（可选）">
          <el-input v-model="ouForm.desc" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <p class="mono muted" style="font-size: 12px">将创建：ou=&lt;名称&gt;,{{ currentDN || me?.baseDN }}</p>
      <template #footer>
        <el-button @click="ouDlg = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createOU">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="renameDlg" title="重命名部门" width="460px">
      <el-form label-position="top">
        <el-form-item label="新名称" required><el-input v-model="renameForm.name" /></el-form-item>
      </el-form>
      <p class="mono muted" style="font-size: 12px">预览：{{ renamePreview?.newDN ?? '—' }}</p>
      <template #footer>
        <el-button @click="renameDlg = false">取消</el-button>
        <el-button type="primary" :loading="renaming" @click="doRename">确认改名</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="moveDlg" title="移动 / 改名" width="520px">
      <el-form label-position="top">
        <el-form-item label="新 RDN（改名时修改，如 ou=newname）"><el-input v-model="moveForm.newRDN" class="mono" /></el-form-item>
        <el-form-item label="新父 DN（移动时修改）"><el-input v-model="moveForm.newParent" class="mono" /></el-form-item>
      </el-form>
      <el-button size="small" :loading="previewing" @click="previewMove">预览变更</el-button>
      <el-alert v-if="mvPreview" type="info" :closable="false" show-icon style="margin-top: 12px"
        :title="`新 DN：${mvPreview.newDN}`"
        :description="`将随之迁移的子条目：${mvPreview.moved} 条${mvPreview.rename ? '（同时改名）' : ''}`" />
      <template #footer>
        <el-button @click="moveDlg = false">取消</el-button>
        <el-button type="primary" :disabled="!mvPreview" :loading="moving" @click="doMove">执行移动</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="delDlg" title="删除条目" width="540px">
      <template v-if="delPrev">
        <p>目标：<span class="mono">{{ delPrev.dn }}</span></p>
        <el-alert v-if="delPrev.empty" type="info" :closable="false" show-icon title="该条目没有子条目，可直接删除。" />
        <template v-else>
          <el-alert type="warning" :closable="false" show-icon
            :title="`该条目下有 ${delPrev.children} 个直接子条目，全部子孙共 ${delPrev.subtree} 条`"
            description="必须显式勾选“级联删除”才能执行；LDAP 无回收站，删除不可撤销。" />
          <div class="sample-list">
            <div v-for="d in delPrev.sample" :key="d" class="mono" style="font-size: 11.5px">{{ d }}</div>
            <div v-if="delPrev.subtree > delPrev.sample.length" class="muted" style="font-size: 11.5px">… 其余 {{ delPrev.subtree - delPrev.sample.length }} 条略</div>
          </div>
          <el-checkbox v-model="delCascade">我已知晓影响，确认级联删除全部 {{ delPrev.subtree }} 条子孙条目</el-checkbox>
        </template>
      </template>
      <p v-else class="muted">正在评估删除影响…</p>
      <template #footer>
        <el-button @click="delDlg = false">取消</el-button>
        <el-button type="danger" :loading="deleting" :disabled="!delPrev || (!delPrev.empty && !delCascade)" @click="doDelete">
          {{ delPrev && !delPrev.empty ? '级联删除' : '删除' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowRightLeft, ChevronDown, Columns3, Download, Ellipsis, Folder, FolderPlus, Info, PencilLine,
  Search, Trash2, UserPlus,
} from 'lucide-vue-next'
import {
  createEntry, deleteEntry, deletePreview, dnParts, errText, exportURL, getEntry,
  me as apiMe, moveEntry, movePreview, ouTree, peopleList, updateEntry, rdnAttr, rdnValue, searchEntries,
  type DeletePreview, type Me, type MovePreview, type SearchResult, type TreeNode,
} from '../api'
import { app, bumpPeople, bumpTree, openEntry, openPersonCreate, searchQuery, setSelected } from '../store/app'

const me = ref<Me | null>(null)
const people = ref<any[]>([])
const loading = ref(false)
const conflictHit = ref(false)
const filter = ref('')
const scope = ref('sub')

const personOpen = ref(false)
const personDN = ref('')

// ---------- 自定义列（localStorage 持久化） ----------
const OPTIONAL_COLS = [
  { key: 'uid', label: '账号', en: 'uid', width: 120 },
  { key: 'employeeNumber', label: '工号', en: 'employeeNumber', width: 110 },
  { key: 'mail', label: '邮箱', en: 'mail', width: 180 },
  { key: 'mobile', label: '手机', en: 'mobile', width: 130 },
  { key: 'title', label: '职务', en: 'title', width: 120 },
  { key: 'sn', label: '姓氏', en: 'sn', width: 90 },
  { key: 'department', label: '部门', en: 'ou', width: 110 },
  { key: 'description', label: '描述', en: 'description', width: 160 },
]
const colOrder = ref<string[]>(
  (() => {
    try {
      const saved = JSON.parse(localStorage.getItem('nl.peopleColsOrder') ?? 'null')
      if (Array.isArray(saved) && saved.length === OPTIONAL_COLS.length) return saved
    } catch { /* 忽略 */ }
    return OPTIONAL_COLS.map(c => c.key)
  })(),
)
watch(colOrder, (v) => localStorage.setItem('nl.peopleColsOrder', JSON.stringify(v)), { deep: true })
const orderedCols = computed(() => {
  const m = new Map(OPTIONAL_COLS.map(c => [c.key, c]))
  return colOrder.value.map(k => m.get(k)!).filter(Boolean)
})
function moveCol(i: number, dir: number) {
  const arr = [...colOrder.value]
  const j = i + dir
  if (j < 0 || j >= arr.length) return
  ;[arr[i], arr[j]] = [arr[j], arr[i]]
  colOrder.value = arr
}
const visibleKeys = ref<string[]>(
  (() => {
    try {
      const saved = JSON.parse(localStorage.getItem('nl.peopleCols2') ?? 'null')
      if (Array.isArray(saved) && saved.length !== undefined) return saved
    } catch { /* 忽略 */ }
    return ['uid', 'employeeNumber', 'mail', 'mobile', 'department']
  })(),
)
watch(visibleKeys, (v) => localStorage.setItem('nl.peopleCols2', JSON.stringify(v)), { deep: true })
const visibleCols = computed(() => orderedCols.value.filter(c => visibleKeys.value.includes(c.key)))

// ---------- 批量移动人员（勾选 → 选目标部门 → modrdn） ----------
const selectedPeople = ref<any[]>([])
const movePeopleDlg = ref(false)
const movePeopleTarget = ref('')
const movingPeople = ref(false)
const ouOptions = ref<TreeNode[]>([])

async function openMovePeople() {
  movePeopleTarget.value = ''
  movePeopleDlg.value = true
  if (!ouOptions.value.length) {
    try { ouOptions.value = await ouTree() } catch { ouOptions.value = [] }
  }
}

function onSelChange(rows: any[]) { selectedPeople.value = rows }

async function doMovePeople() {
  if (!movePeopleTarget.value || !selectedPeople.value.length) return
  movingPeople.value = true
  const failed: string[] = []
  try {
    for (const p of selectedPeople.value) {
      try {
        await moveEntry(p.dn, undefined, movePeopleTarget.value)
      } catch {
        failed.push(p.uid || p.cn)
      }
    }
    if (failed.length) {
      ElMessage.warning(`已移动 ${selectedPeople.value.length - failed.length} 人；失败 ${failed.length} 人：${failed.join('、')}`)
    } else {
      ElMessage.success(`已移动 ${selectedPeople.value.length} 人到目标部门`)
    }
    movePeopleDlg.value = false
    movePeopleTarget.value = ''
    refresh()
    bumpTree()
  } finally {
    movingPeople.value = false
  }
}

const currentDN = computed(() => app.selectedOU?.dn ?? '')
const filtered = computed(() =>
  people.value.filter((p) => !filter.value ||
    [p.cn, p.uid, p.employeeNumber, p.mail].join(' ').toLowerCase().includes(filter.value.toLowerCase())))
const scopeSub = computed(() => scope.value === 'sub')

function openPerson(dn: string) { openEntry(dn) }

async function refresh() {
  loading.value = true
  try {
    const res = await peopleList(currentDN.value || me.value?.baseDN || '')
    people.value = (res.people ?? []).map((p) => ({
      dn: p.dn,
      cn: p.attrs.cn?.[0] ?? '',
      uid: p.attrs.uid?.[0] ?? '',
      employeeNumber: p.attrs.employeeNumber?.[0] ?? '',
      mail: p.attrs.mail?.[0] ?? '',
      mobile: p.attrs.mobile?.[0] ?? '',
      title: p.attrs.title?.[0] ?? '',
      sn: p.attrs.sn?.[0] ?? '',
      description: p.attrs.description?.[0] ?? '',
      disabled: p.disabled,
      // 部门 = DN 中第一级 ou（如 uid=x,ou=tech,… → tech）
      department: (p.dn.split(',')[1] ?? '').replace(/^ou=/i, ''),
    }))
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    loading.value = false
  }
}

async function delPerson(row: any) {
  try {
    await ElMessageBox.confirm(`删除人员 ${row.cn}（${row.uid}）？此操作不可撤销。`, '删除人员', { type: 'warning' })
  } catch { return }
  try {
    await deleteEntry(row.dn, false)
    ElMessage.success('已删除')
    refresh()
    bumpTree() // 侧栏树的子节点数/叶子状态同步
  } catch (e: any) {
    if (e?.response?.status === 409) ElMessage.error('该条目有子条目，请在树中处理')
    else ElMessage.error(errText(e))
  }
}

function exportCurrent() {
  window.open(exportURL(currentDN.value || undefined), '_blank')
}

// ---------- 全局搜索（顶栏触发） ----------
const searching = ref(false)
const searchResult = ref<SearchResult | null>(null)
const searchedKw = ref('')
const searchRows = computed(() =>
  (searchResult.value?.entries ?? []).map((e) => ({
    dn: e.dn,
    name: e.attrs.cn?.[0] ?? e.attrs.ou?.[0] ?? rdnValue(dnParts(e.dn)[0] ?? ''),
    isPerson: (e.attrs.objectClass ?? []).includes('inetOrgPerson'),
  })),
)

async function runSearch(q: string) {
  if (!q) { clearSearch(); return }
  searching.value = true
  try {
    searchResult.value = await searchEntries({ q, attr: ['cn', 'uid', 'ou', 'employeeNumber', 'objectClass'], limit: 50 })
    searchedKw.value = q
  } catch (e) { ElMessage.error(errText(e)) } finally { searching.value = false }
}

function clearSearch() { searchResult.value = null; searchQuery.value = '' }

watch(() => searchQuery.value, (q) => {
  if (q) runSearch(q)
  else if (searchResult.value) clearSearch() // 顶栏清空 → 退出搜索模式回到浏览
})
watch(() => app.selectedOU, () => { clearSearch(); refresh() })
watch(scope, () => refresh())
watch(() => app.peopleVersion, () => refresh()) // 树上行内新建等跨组件来源的人员变更

// ---------- 树定位（搜索结果 → 树选中） ----------
async function locate(dn: string) {
  const baseParts = dnParts(me.value?.baseDN ?? '')
  const parts = dnParts(dn)
  if (parts.length <= baseParts.length) return
  const names = parts.slice(0, parts.length - baseParts.length).reverse().map((p) => rdnValue(p))
  const treeEl = () => document.querySelector('.el-tree')
  for (let t = 0; t < 100 && !treeEl(); t++) await new Promise((r) => setTimeout(r, 30))
  let container: HTMLElement | null = treeEl() as HTMLElement | null
  if (!container) return
  const findChild = (root: HTMLElement | null, name: string): HTMLElement | null => {
    if (!root) return null
    for (const node of Array.from(root.querySelectorAll<HTMLElement>(':scope > .el-tree-node'))) {
      const label = node.querySelector<HTMLElement>(':scope > .el-tree-node__content .el-tree-node__label')
        ?? node.querySelector<HTMLElement>(':scope > .el-tree-node__content')
      const text = (label?.textContent ?? '').trim()
      if (text === name || text.startsWith(name)) return node
    }
    return null
  }
  for (let i = 0; i < names.length; i++) {
    const name = names[i]
    let node = findChild(container, name)
    for (let t = 0; t < 150 && !node; t++) { await new Promise((r) => setTimeout(r, 30)); node = findChild(container, name) }
    if (!node) { ElMessage.info('已在树中定位到最近一级节点'); return }
    if (i === names.length - 1) {
      node.querySelector<HTMLElement>(':scope > .el-tree-node__content')
        ?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      node.scrollIntoView({ block: 'nearest' })
      return
    }
    if (!node.classList.contains('is-expanded')) {
      node.querySelector<HTMLElement>(':scope > .el-tree-node__content .el-tree-node__expand-icon')
        ?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    }
    const kids = () => node.querySelector<HTMLElement>(':scope > .el-tree-node__children')
    for (let t = 0; t < 100 && !kids(); t++) await new Promise((r) => setTimeout(r, 30))
    if (!kids()) { ElMessage.info('已在树中定位到最近一级节点'); return }
    container = kids()
  }
}

// ---------- 编辑部门信息（中文名=第二个 ou 值；说明=description） ----------
const editOUDlg = ref(false)
const editOUBusy = ref(false)
const editOUForm = reactive({ cn: '', desc: '' })

async function openEditOU() {
  if (!app.selectedOU) return
  try {
    const e = await getEntry(app.selectedOU.dn)
    const ous = e.attrs.ou ?? []
    editOUForm.cn = ous.length > 1 ? ous[1] : ''
    editOUForm.desc = e.attrs.description?.[0] ?? ''
    editOUDlg.value = true
  } catch (e) {
    ElMessage.error(errText(e))
  }
}

async function saveEditOU() {
  if (!app.selectedOU) return
  editOUBusy.value = true
  try {
    const e = await getEntry(app.selectedOU.dn)
    const first = (e.attrs.ou ?? [app.selectedOU.name])[0]
    const cn = editOUForm.cn.trim()
    const newOU = cn && cn !== first ? [first, cn] : [first]
    const changes: Array<{ op: string; attr: string; vals: string[] }> = [
      { op: 'replace', attr: 'ou', vals: newOU },
    ]
    const curDesc = e.attrs.description?.[0] ?? ''
    if (editOUForm.desc.trim() !== curDesc) {
      changes.push({ op: 'replace', attr: 'description', vals: editOUForm.desc.trim() ? [editOUForm.desc.trim()] : [] })
    }
    await updateEntry(app.selectedOU.dn, e.entryCSN, changes)
    ElMessage.success('部门信息已保存')
    editOUDlg.value = false
    bumpTree()
    refresh()
  } catch (e: any) {
    if (e?.response?.status === 409) ElMessage.error('部门刚被他人修改，请重试')
    else ElMessage.error(errText(e))
  } finally {
    editOUBusy.value = false
  }
}

// ---------- OU 操作（M1 逻辑保留） ----------
const ouDlg = ref(false)
const creating = ref(false)
const ouForm = reactive({ name: '', desc: '' })

function openCreateOU() { ouForm.name = ''; ouForm.desc = ''; ouDlg.value = true }

async function createOU() {
  const name = ouForm.name.trim()
  if (!name || name.includes(',') || name.includes('=')) { ElMessage.warning('部门名称不能为空，且不能包含 , 或 ='); return }
  creating.value = true
  try {
    const parent = currentDN.value || me.value?.baseDN || ''
    const attrs: Record<string, string[]> = { ou: [name] }
    if (ouForm.desc.trim()) attrs.description = [ouForm.desc.trim()]
    await createEntry(`ou=${name},${parent}`, ['organizationalUnit'], attrs)
    ElMessage.success(`已创建部门：${name}`)
    ouDlg.value = false
    bumpTree()
    setTimeout(() => locate(`ou=${name},${parent}`), 400)
  } catch (e) { ElMessage.error(errText(e)) } finally { creating.value = false }
}

const renameDlg = ref(false)
const renaming = ref(false)
const renameForm = reactive({ name: '' })
const renamePreview = ref<MovePreview | null>(null)

function openRename() {
  if (!app.selectedOU) return
  renameForm.name = app.selectedOU.name
  renamePreview.value = null
  renameDlg.value = true
}

watch([renameDlg, () => renameForm.name], async () => {
  if (renameDlg.value && app.selectedOU && renameForm.name.trim() && !renameForm.name.includes(',')) {
    const rdn = dnParts(app.selectedOU.dn)[0] ?? ''
    try { renamePreview.value = await movePreview(app.selectedOU.dn, `${rdnAttr(rdn)}=${renameForm.name.trim()}`) }
    catch { renamePreview.value = null }
  } else renamePreview.value = null
})

async function doRename() {
  const name = renameForm.name.trim()
  if (!name || name.includes(',') || name.includes('=')) { ElMessage.warning('名称不能为空，且不能包含 , 或 ='); return }
  if (!app.selectedOU) return
  const rdn = dnParts(app.selectedOU.dn)[0] ?? ''
  renaming.value = true
  try {
    const newDN = await moveEntry(app.selectedOU.dn, `${rdnAttr(rdn)}=${name}`)
    ElMessage.success('已重命名')
    renameDlg.value = false
    bumpTree()
    setTimeout(() => locate(newDN), 400)
  } catch (e) { ElMessage.error(errText(e)) } finally { renaming.value = false }
}

const moveDlg = ref(false)
const moving = ref(false)
const previewing = ref(false)
const mvPreview = ref<MovePreview | null>(null)
const moveForm = reactive({ newRDN: '', newParent: '' })

watch(moveDlg, (open) => {
  if (open && app.selectedOU) {
    const parts = dnParts(app.selectedOU.dn)
    moveForm.newRDN = parts[0] ?? ''
    moveForm.newParent = parts.slice(1).join(',')
    mvPreview.value = null
  }
})

watch([() => moveForm.newRDN, () => moveForm.newParent], async () => {
  if (moveDlg.value && app.selectedOU && moveForm.newRDN.trim() && moveForm.newParent.trim()) {
    try { mvPreview.value = await movePreview(app.selectedOU.dn, moveForm.newRDN, moveForm.newParent) }
    catch { mvPreview.value = null }
  } else mvPreview.value = null
})

async function previewMove() {
  if (!app.selectedOU) return
  previewing.value = true
  try { mvPreview.value = await movePreview(app.selectedOU.dn, moveForm.newRDN, moveForm.newParent) }
  catch (e) { ElMessage.error(errText(e)) } finally { previewing.value = false }
}

async function doMove() {
  if (!app.selectedOU || !mvPreview.value) return
  moving.value = true
  try {
    const newDN = await moveEntry(app.selectedOU.dn, moveForm.newRDN, moveForm.newParent)
    ElMessage.success(`已移动到 ${newDN}`)
    moveDlg.value = false
    bumpTree()
    setTimeout(() => locate(newDN), 400)
  } catch (e) { ElMessage.error(errText(e)) } finally { moving.value = false }
}

const delDlg = ref(false)
const deleting = ref(false)
const delPrev = ref<DeletePreview | null>(null)
const delCascade = ref(false)

async function openDelete() {
  if (!app.selectedOU) return
  delCascade.value = false
  delPrev.value = null
  delDlg.value = true
  try { delPrev.value = await deletePreview(app.selectedOU.dn) }
  catch (e) { ElMessage.error(errText(e)); delDlg.value = false }
}

async function doDelete() {
  if (!app.selectedOU || !delPrev.value) return
  if (!delPrev.value.empty && !delCascade.value) { ElMessage.warning('请先勾选级联删除确认'); return }
  deleting.value = true
  try {
    await deleteEntry(app.selectedOU.dn, delCascade.value)
    ElMessage.success('已删除')
    delDlg.value = false
    setSelected(null)
    bumpTree()
    refresh()
  } catch (e: any) {
    if (e?.response?.status === 409 && e?.response?.data?.code === 'not-empty') {
      ElMessage.error('条目下有子条目，需勾选级联删除')
      if (app.selectedOU) delPrev.value = await deletePreview(app.selectedOU.dn)
    } else ElMessage.error(errText(e))
  } finally { deleting.value = false }
}



onMounted(async () => {
  me.value = await apiMe()
  refresh()
  if (searchQuery.value) runSearch(searchQuery.value)
})
</script>

<style scoped>
.sample-list { max-height: 180px; overflow: auto; border: 1px solid var(--el-border-color); border-radius: 8px; padding: 8px 12px; margin: 10px 0; }
/* 表头：中文主标签 + 小号英文同行显示，禁止换行（避免行高被长属性名撑宽） */
.th-cn { white-space: nowrap; font-size: 12.5px; }
.th-en { font-size: 10.5px; color: #8f959e; font-family: ui-monospace, Consolas, monospace; }
.col-order-list { display: flex; flex-direction: column; }
.col-order-row { display: flex; align-items: center; padding: 2px 0; }
.col-order-row .ord-ops { margin-left: auto; display: inline-flex; }
.col-order-row .ord-ops .el-button { padding: 2px 6px; }
:deep(.el-table th.el-table__cell) { white-space: nowrap; }
</style>
