<template>
  <el-drawer :model-value="open" size="620px" @update:model-value="$emit('close')">
    <template #header>
      <div class="eh">
        <span class="nl-avatar" :class="{ gray: !isPerson }">{{ headerChar }}</span>
        <div>
          <div class="et">{{ displayTitle }}<span class="kind-tag" v-if="view">{{ kindTag(view.structural) }}</span></div>
          <!-- DN 面包屑（LAM/phpLDAPadmin 风格） -->
          <div class="dn-crumb mono" v-if="detail">
            <template v-for="(seg, i) in dnSegments" :key="i">
              <span class="seg" :class="{ base: i === dnSegments.length - 1 }">{{ seg }}</span>
              <span v-if="i < dnSegments.length - 1" class="sep">›</span>
            </template>
          </div>
        </div>
      </div>
    </template>

    <!-- 简单模式（新手）：业务字段卡片 -->
    <SimplePersonPanel v-if="!app.expertMode" :dn="dn" @close="$emit('close')" @saved="$emit('saved', $event)" @enter-expert="app.expertMode = true" />
    <!-- 专家模式：完整 LDAP 视图 -->
    <div v-if="detail && view && app.expertMode" v-loading="loading">
      <el-alert v-if="conflict" type="error" show-icon :closable="false" style="margin-bottom: 12px"
        title="条目已被其他客户端修改（如 LAM），已刷新为最新版本" />

      <div class="expert-tabs">
        <el-tabs v-model="tab">
        <!-- 属性（必填/可选/其他，双语标签；操作属性独立页签） -->
        <el-tab-pane label="属性" name="attrs">
          <!-- 人员：所属组反查（member 在组条目上，人员条目本身看不到） -->
          <template v-if="isPerson && personGroups.length">
            <div class="sec-title">所属组 <span class="muted en">memberOf（反查）</span><span class="count">{{ personGroups.length }}</span></div>
            <div class="groups-wall">
              <el-tag v-for="g in personGroups" :key="g.dn" :type="g.type === '登录组' ? 'info' : 'primary'"
                effect="plain" style="margin: 0 6px 6px 0">{{ g.name }} · {{ g.type }}</el-tag>
            </div>
            <div class="muted" style="font-size: 11.5px; margin-bottom: 8px">组成员关系存储在组条目的 member 属性上；增删成员请到「用户组」页。</div>
          </template>
          <template v-else-if="isPerson">
            <div class="sec-title">所属组 <span class="muted en">memberOf（反查）</span></div>
            <div class="muted" style="font-size: 11.5px; margin-bottom: 10px">未加入任何组；到「用户组」页可将该成员加入。</div>
          </template>
          <template v-for="g in visibleGroups" :key="g">
            <div class="sec-title">
              {{ groupTitle(g).cn }}
              <span class="muted en">{{ groupTitle(g).en }}</span>
              <span class="count">{{ groupAttrs(g).length }}</span>
            </div>
            <div v-for="a in groupAttrs(g)" :key="a.name" class="attr-row">
              <div class="attr-label">
                <span>{{ dispLabel(a) }}</span>
                <span v-if="showEn(a)" class="mono en">{{ a.name }}</span>
                <el-tooltip v-if="attrDesc(a.name)" :content="attrDesc(a.name)" placement="top" :show-after="200">
                  <el-icon class="qmark"><CircleHelp /></el-icon>
                </el-tooltip>
                <span v-if="a.single" class="pill plain">单值</span>
              </div>
              <div class="attr-values">
                <div v-for="(v, i) in editValues[a.name] ?? []" :key="i" class="val-line">
                  <el-input v-model="editValues[a.name][i]" size="small" class="mono" />
                  <el-button v-if="editValues[a.name].length > 1 || !(mustSet.has(a.name))" size="small" text
                    type="danger" @click="removeValue(a.name, i)">删</el-button>
                </div>
                <el-button v-if="!a.single" size="small" text type="primary" @click="addValue(a.name)">+ 添加值</el-button>
              </div>
            </div>
            <!-- 新增中的属性（未保存前显示在这里，填值后随保存提交） -->
            <div v-for="name in addedList" :key="'new-' + name" class="attr-row new-attr">
              <div class="attr-label">
                <span>{{ attrCN(name) ?? name }}</span>
                <span v-if="!attrCN(name)" class="mono en">{{ name }}</span>
                <span class="pill info">新增</span>
                <el-button size="small" text type="danger" style="margin-left: auto" @click="removeAdded(name)">取消</el-button>
              </div>
              <div class="attr-values">
                <div v-for="(v, i) in editValues[name] ?? []" :key="i" class="val-line">
                  <el-input v-model="editValues[name][i]" size="small" class="mono" placeholder="填写属性值…" />
                </div>
              </div>
            </div>
            <!-- 添加属性（phpLDAPadmin 风格：只列类链 MAY 未占用） -->
            <div v-if="g === 'may' && availableFiltered.length" class="add-attr">
              <el-select v-model="pendingAttr" size="small" filterable placeholder="选择要添加的属性…"
                style="width: 300px">
                <el-option v-for="b in availableFiltered" :key="b.name" :value="b.name"
                  :label="attrLabel(b)">
                  <span>{{ attrLabel(b) }}</span>
                  <span v-if="!attrLabel(b).includes(b.name)" class="mono muted" style="float: right; font-size: 11px">{{ b.name }}</span>
                </el-option>
              </el-select>
              <el-button size="small" @click="addAttr" :disabled="!pendingAttr">添加属性</el-button>
            </div>
          </template>
        </el-tab-pane>

        <!-- 对象类：类型视图（对象类由服务器 schema 定义，此处只能选用） -->
        <el-tab-pane label="对象类" name="classes">
          <div class="nl-hint info" style="margin-bottom: 12px">
            <el-icon style="margin-top: 2px"><Info /></el-icon>
            <div>对象类决定这个条目"是什么、有哪些字段"，由 LDAP 服务器 schema 统一定义——这里只能<b>选用</b>，不能发明新类。人员一般无需改动；需要 Linux 登录时可追加 posixAccount。</div>
          </div>

          <!-- 主类卡片 -->
          <div class="sec-title">主类（结构类）<span class="muted en">Structural</span></div>
          <div class="oc-main">
            <template v-if="structuralClasses.length">
              <div v-for="c in structuralClasses" :key="c.name" class="oc-row structural">
                <div class="oc-main-name">
                  <span class="mono oc-name">{{ c.name }}</span>
                  <span v-if="classCN(c.name)" class="oc-cn">{{ classCN(c.name) }}</span>
                  <el-tooltip v-if="classDesc(c.name)" :content="classDesc(c.name)" placement="top" :show-after="200">
                    <el-icon class="qmark"><CircleHelp /></el-icon>
                  </el-tooltip>
                </div>
                <span v-if="c.name === view.structural" class="pill info">当前主类</span>
                <span v-else class="pill plain">继承自</span>
              </div>
            </template>
            <div v-else class="muted" style="padding: 4px 0">未识别到已知结构类（当前条目使用未知 schema 的类）</div>
          </div>

          <!-- 更换主类 -->
          <div v-if="pendingMain" class="nl-hint warn" style="margin: 6px 0">
            <el-icon style="margin-top: 2px"><TriangleAlert /></el-icon>
            <div>保存后将更换主类为 <b class="mono">{{ pendingMain }}</b>；如新类有必填属性，请在「属性」页补值。</div>
          </div>
          <div class="oc-change">
            <el-select v-model="pendingClass" filterable placeholder="更换主类（一般无需改动）…" style="flex: 1">
              <el-option v-for="c in structuralAvailable" :key="c.name" :value="c.name"
                :label="classCN(c.name) ? `${c.name} · ${classCN(c.name)}` : c.name">
                <span class="mono">{{ c.name }}</span>
                <span v-if="classCN(c.name)" class="oc-cn" style="margin-left: 6px">{{ classCN(c.name) }}</span>
              </el-option>
            </el-select>
            <el-button type="primary" plain :disabled="!pendingClass" @click="appendClass">更换主类</el-button>
          </div>

          <!-- 辅助类 -->
          <div class="sec-title">辅助类 <span class="muted en">Auxiliary · 可增删</span></div>
          <div class="oc-list">
            <div v-for="c in auxiliaryClasses" :key="c.name" class="oc-row">
              <div class="oc-main-name">
                <span class="mono oc-name">{{ c.name }}</span>
                <span v-if="classCN(c.name)" class="oc-cn">{{ classCN(c.name) }}</span>
                <el-tooltip v-if="classDesc(c.name)" :content="classDesc(c.name)" placement="top" :show-after="200">
                  <el-icon class="qmark"><CircleHelp /></el-icon>
                </el-tooltip>
              </div>
              <el-button size="small" text type="danger" @click="removeClass(c.name)">移除</el-button>
            </div>
            <div v-if="!auxiliaryClasses.length" class="muted" style="padding: 4px 0">无辅助类</div>
          </div>
          <div class="oc-change">
            <el-select v-model="pendingAux" filterable placeholder="添加辅助类（如 posixAccount 开通 Linux 登录）…" style="flex: 1">
              <el-option v-for="c in auxiliaryAvailable" :key="c.name" :value="c.name"
                :label="classCN(c.name) ? `${c.name} · ${classCN(c.name)}` : c.name">
                <span class="mono">{{ c.name }}</span>
                <span v-if="classCN(c.name)" class="oc-cn" style="margin-left: 6px">{{ classCN(c.name) }}</span>
              </el-option>
            </el-select>
            <el-button type="primary" plain :disabled="!pendingAux" @click="appendAux">添加辅助类</el-button>
          </div>

          <div class="nl-hint warn" style="margin-top: 12px">
            <el-icon style="margin-top: 2px"><TriangleAlert /></el-icon>
            <div>更换主类后，新类的必填属性需在「属性」页补值才能保存（服务端 schema 校验）；辅助类可随时追加或移除。</div>
          </div>
        </el-tab-pane>

        <!-- 操作属性（只读，独立页签） -->
        <el-tab-pane :label="`操作属性 (${groupAttrs('operational').length})`" name="operational">
          <div class="nl-hint info" style="margin-bottom: 10px">
            <el-icon style="margin-top: 2px"><Info /></el-icon>
            <div>由目录服务器自动维护的元数据（只读），不由用户填写。</div>
          </div>
          <div v-for="a in groupAttrs('operational')" :key="a.name" class="attr-row">
            <div class="attr-label">
              <span>{{ dispLabel(a) }}</span>
              <span v-if="showEn(a)" class="mono en">{{ a.name }}</span>
              <el-tooltip v-if="attrDesc(a.name)" :content="attrDesc(a.name)" placement="top" :show-after="200">
                <el-icon class="qmark"><CircleHelp /></el-icon>
              </el-tooltip>
              <span class="pill plain">只读</span>
            </div>
            <div class="attr-values readonly mono">
              <div v-for="v in a.values" :key="v" class="val-line-ro">{{ v }}</div>
            </div>
          </div>
          <div v-if="!groupAttrs('operational').length" class="muted" style="padding: 8px 0">无操作属性</div>
        </el-tab-pane>

        <!-- LDIF 原文（全部属性，只读） -->
        <el-tab-pane label="LDIF" name="ldif">
          <div class="nl-hint info" style="margin-bottom: 10px">
            <el-icon style="margin-top: 2px"><Info /></el-icon>
            <div>与 ldapsearch 输出等价的原文视图（未保存的修改不体现）。</div>
          </div>
          <pre class="ldif">{{ ldifText }}</pre>
        </el-tab-pane>
        </el-tabs>
        <!-- 与页签同一行右对齐，避免整行空白 -->
        <el-button class="back-simple" text size="small" @click="app.expertMode = false">‹ 切回简单模式</el-button>
      </div>
    </div>

    <!-- 抽屉底部操作条仅在专家模式渲染；简单模式由 SimplePersonPanel 自带页脚，
         避免两套按钮同时出现（截图反馈的双重按钮行问题） -->
    <template #footer v-if="app.expertMode">
      <div class="footer">
        <div class="left" v-if="isPerson">
          <el-button size="small" @click="doReset" :loading="busy">重置密码</el-button>
          <el-button size="small" v-if="!disabled" @click="doDisable">禁用</el-button>
          <el-button size="small" v-else @click="doEnable">启用</el-button>
        </div>
        <div>
          <el-button @click="$emit('close')">取消</el-button>
          <el-button type="primary" :loading="busy" @click="save">保存修改</el-button>
        </div>
      </div>
    </template>
  </el-drawer>

  <!-- 重置密码结果 -->
  <el-dialog v-model="pwResult" title="新密码（仅显示一次）" width="380px" append-to-body>
    <p class="mono pw">{{ pwShown }}</p>
    <p class="muted">请立即安全转交该员工；密码不会再次显示，也不写入审计日志。</p>
    <template #footer>
      <el-button @click="copyPw">复制</el-button>
      <el-button type="primary" @click="pwResult = false">我已转交</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleHelp, Info, TriangleAlert } from 'lucide-vue-next'
import SimplePersonPanel from './SimplePersonPanel.vue'
import { app } from '../store/app'
import { attrCN, attrDesc, classCN, classDesc } from '../utils/ldapDict'
import {
  disablePerson, enablePerson, errText, entryView, personDetail, resetPassword, updateEntry,
  type AttrView, type EntryViewResult,
} from '../api'

const props = defineProps<{ open: boolean; dn: string }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'saved', dn: string): void }>()

const loading = ref(false)
const busy = ref(false)
const detail = ref<EntryViewResult | null>(null)
const tab = ref('attrs')
const conflict = ref(false)
const disabled = ref(false)
const editValues = reactive<Record<string, string[]>>({})
const addedAttrs = ref(new Set<string>())
const removedAttrs = ref(new Set<string>())
const pendingAttr = ref('')
const pendingClass = ref('')

const view = computed(() => detail.value?.view)
const isPerson = computed(() => (view.value?.objectClasses ?? []).some(c => c.name === 'inetOrgPerson'))
const headerChar = computed(() => {
  const cn = detail.value?.attrs?.cn?.[0] ?? detail.value?.attrs?.ou?.[0] ?? detail.value?.dn ?? '?'
  return cn[0] ?? '?'
})
const displayTitle = computed(() =>
  detail.value?.attrs?.cn?.[0] ?? detail.value?.attrs?.ou?.[0] ?? detail.value?.dn.split(',')[0] ?? '')

const dnSegments = computed(() => {
  if (!detail.value) return []
  return detail.value.dn.split(',').map(s => s.trim()).reverse()
})

const kindOf = (name: string) => view.value?.objectClasses.find(c => c.name === name)?.kind ?? 'unknown'
const kindTag = (structural: string) => structural ? structural : ''

// 对象类页视图数据
const structuralClasses = computed(() =>
  (view.value?.objectClasses ?? []).filter(c => kindOf(c.name) === 'STRUCTURAL'))
const auxiliaryClasses = computed(() =>
  (view.value?.objectClasses ?? []).filter(c => kindOf(c.name) === 'AUXILIARY'))
const structuralAvailable = computed(() =>
  (view.value?.availableClasses ?? []).filter(c => c.kind !== 'AUXILIARY'))
const auxiliaryAvailable = computed(() =>
  (view.value?.availableClasses ?? []).filter(c => c.kind === 'AUXILIARY'))
const pendingAux = ref('')
const pendingMain = ref('')
const personGroups = ref<Array<{ dn: string; name: string; type: string }>>([])

// 更换主类 / 添加辅助类：写入待保存集合（保存时统一提交 objectClass 替换）
function appendClass() {
  if (!pendingClass.value) return
  pendingAux.value = ''
  pendingMain.value = pendingClass.value
  pendingClass.value = ''
  ElMessage.info('已选择新主类，点击底部「保存修改」生效')
}
function appendAux() {
  if (!pendingAux.value || !view.value) return
  const name = pendingAux.value
  if (!view.value.objectClasses.some(c => c.name === name)) {
    view.value.objectClasses.push({ name, kind: 'AUXILIARY' })
  }
  pendingAux.value = ''
  ElMessage.info('已加入辅助类，点击底部「保存修改」生效')
}

const groupDefs: Record<string, { cn: string; en: string }> = {
  must: { cn: '必填属性', en: 'MUST' },
  may: { cn: '可选属性', en: 'MAY' },
  operational: { cn: '操作属性', en: 'Operational（只读）' },
  unknown: { cn: '其他属性', en: 'Not in schema' },
}
const groupTitle = (g: string) => groupDefs[g] ?? { cn: g, en: '' }
const visibleGroups = computed(() => {
  if (!view.value) return []
  const groups = view.value.attrs.map(a => a.group)
  // 属性页只含 must/may/unknown；操作属性独立页签展示
  return ['must', 'may', 'unknown'].filter(g => groups.includes(g as AttrView['group']))
})
const groupAttrs = (g: any) => {
  const all = view.value?.attrs ?? []
  if (g === 'must') {
    // uid 是登录身份，产品层面视为必填：置于必填组首位（即便 schema 里是 MAY）
    const uid = all.find((a: any) => a.name === 'uid')
    const must = all.filter((a: any) => a.group === 'must' && a.name !== 'uid')
    return uid ? [uid, ...must] : must
  }
  if (g === 'may') {
    return all.filter((a: any) => a.group === 'may' && a.name !== 'uid')
  }
  return all.filter((a: any) => a.group === g)
}
// 添加属性下拉的显示标签：有中文用中文+英文名；后端无中文且与 name 相同时只显示一次
const attrLabel = (b: any) => {
  const cn = attrCN(b.name)
  if (cn) return `${cn} (${b.name})`
  if ((b.label ?? '').toLowerCase() === b.name.toLowerCase()) return b.name
  return `${b.label} (${b.name})`
}
// 中文标签已含英文属性名（如「账号 (uid)」）时不再重复显示属性名；
// 字典有中文短标签的（多为操作属性）优先使用字典
const dispLabel = (a: any) => attrCN(a.name) ?? a.label
const showEn = (a: any) => !dispLabel(a).toLowerCase().includes(a.name.toLowerCase())
const mustSet = computed(() => new Set(groupAttrs('must').map(a => a.name)))

const addedList = computed(() => Array.from(addedAttrs.value))
function removeAdded(name: string) {
  addedAttrs.value.delete(name)
  delete editValues[name]
}
const availableFiltered = computed(() => (view.value?.availableAttrs ?? []).filter(b => !addedAttrs.value.has(b.name)))

const ldifText = computed(() => {
  if (!detail.value) return ''
  const lines = [`dn: ${detail.value.dn}`]
  for (const [k, vs] of Object.entries(detail.value.attrs)) {
    for (const v of vs) lines.push(`${k}: ${v}`)
  }
  return lines.join('\n')
})

async function load() {
  if (!props.dn) return
  loading.value = true
  try {
    const ev = await entryView(props.dn)
    detail.value = ev
    for (const k of Object.keys(editValues)) delete editValues[k]
    addedAttrs.value = new Set()
    removedAttrs.value = new Set()
    pendingAttr.value = ''
    pendingClass.value = ''
    // 用已加载的值初始化编辑区：否则 v-for="editValues[a.name] ?? []" 对已有属性
    // 渲染不出任何输入框（专家模式不可编辑的根因）
    for (const a of detail.value.view.attrs) {
      if (!a.readOnly && a.group !== 'operational') {
        editValues[a.name] = [...a.values]
      }
    }
    // 人员条目补取禁用态（entryView 剥离了 userPassword，读不到 {crypt}! 前缀），
    // 否则已禁用账号在专家模式下永远显示「禁用」而无法启用
    if ((ev.attrs?.objectClass ?? []).some(c => c === 'inetOrgPerson')) {
      try {
        const p = await personDetail(props.dn)
        disabled.value = p.disabled
        personGroups.value = p.groups ?? []
      } catch { /* 保持现状 */ }
    } else {
      disabled.value = false
      personGroups.value = []
    }
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    loading.value = false
  }
}

// 冲突标记的复位只在（重新）打开抽屉时做：load() 内复位会吞掉
// 409 处理里"刷新后仍提示冲突"的 el-alert（闪现即逝）
watch(() => [props.open, props.dn], ([open]) => { if (open) { conflict.value = false; load() } }, { immediate: true })

function addValue(name: string) {
  if (!editValues[name]) editValues[name] = ['']
  else editValues[name].push('')
}
function removeValue(name: string, i: number) {
  editValues[name]?.splice(i, 1)
  // MUST 属性至少留一个输入框
  if (mustSet.value.has(name) && (editValues[name] ?? []).length === 0) editValues[name] = ['']
  // 用户主动清空的可选属性 → 删除属性
  if (!mustSet.value.has(name) && (editValues[name] ?? []).length === 0 && detail.value?.attrs[name]) {
    removedAttrs.value.add(name)
  }
}
function addAttr() {
  const name = pendingAttr.value
  if (!name) return
  editValues[name] = ['']
  addedAttrs.value.add(name)
  pendingAttr.value = ''
}
function removeClass(name: string) {
  if (!view.value || !detail.value) return
  detail.value.attrs.objectClass = detail.value.attrs.objectClass.filter(c => c !== name)
  view.value.objectClasses = view.value.objectClasses.filter(c => c.name !== name)
}

async function save() {
  if (!detail.value || !view.value) return
  busy.value = true
  try {
    const changes: Array<{ op: string; attr: string; vals: string[] }> = []
    // 删除的属性
    for (const name of removedAttrs.value) {
      if (detail.value.attrs[name]) changes.push({ op: 'delete', attr: name, vals: detail.value.attrs[name] })
    }
    // 新增属性（值为空的跳过）
    for (const name of addedAttrs.value) {
      const vals = (editValues[name] ?? []).filter(v => v.trim() !== '')
      if (vals.length > 0) changes.push({ op: 'add', attr: name, vals })
    }
    // 已有属性的 diff
    for (const a of view.value.attrs) {
      if (a.readOnly || a.group === 'operational' || removedAttrs.value.has(a.name)) continue
      const cur = (editValues[a.name] ?? a.values).filter(v => v !== undefined)
      const next = cur.filter(v => v.trim() !== '')
      if (JSON.stringify(a.values) !== JSON.stringify(next)) {
        changes.push({ op: 'replace', attr: a.name, vals: next })
      }
    }
    // objectClass 变更
    const origOC = detail.value.attrs.objectClass
    const curOC = view.value.objectClasses.map(c => c.name)
    if (JSON.stringify(origOC) !== JSON.stringify(curOC)) {
      changes.push({ op: 'replace', attr: 'objectClass', vals: curOC })
    }
    // 更换主类（替换主类场景）；pendingMain 来自对象类页「更换主类」按钮的暂存
    if (pendingClass.value || pendingMain.value) {
      const target = pendingClass.value || pendingMain.value
      const kind = view.value.availableClasses.find(c => c.name === target)?.kind ?? 'STRUCTURAL'
      const final = kind === 'AUXILIARY'
        ? [...curOC, target]
        : [...curOC.filter(c => !isKnownStructural(c)), target]
      const existing = changes.find(c => c.attr === 'objectClass')
      if (existing) existing.vals = final
      else changes.push({ op: 'replace', attr: 'objectClass', vals: final })
      pendingClass.value = ''
      pendingMain.value = ''
    }
    if (!changes.length) {
      ElMessage.info('没有修改需要保存')
      return
    }
    await updateEntry(detail.value.dn, detail.value.entryCSN, changes)
    ElMessage.success(`已保存（${changes.length} 处属性级修改，LAM 侧实时可见）`)
    emit('saved', detail.value.dn)
    load()
  } catch (e: any) {
    if (e?.response?.status === 409) {
      conflict.value = true
      ElMessage.error('编辑冲突：条目已被其他客户端修改，已刷新为最新版本')
      load()
    } else {
      ElMessage.error(errText(e))
    }
  } finally {
    busy.value = false
  }
}

const isKnownStructural = (name: string) =>
  (view.value?.objectClasses ?? []).some(c => c.name === name && c.kind === 'STRUCTURAL')

async function doReset() {
  busy.value = true
  try {
    const { password } = await resetPassword(props.dn)
    pwShown.value = password
    pwResult.value = true
  } catch (e) { ElMessage.error(errText(e)) } finally { busy.value = false }
}
const pwResult = ref(false)
const pwShown = ref('')
function copyPw() {
  navigator.clipboard?.writeText(pwShown.value)
  ElMessage.success('已复制')
}
async function doDisable() {
  busy.value = true
  try {
    await disablePerson(props.dn)
    disabled.value = true
    ElMessage.success('已禁用（{crypt}! 前缀约定，可逆）')
    emit('saved', props.dn)
  } catch (e) { ElMessage.error(errText(e)) } finally { busy.value = false }
}
async function doEnable() {
  busy.value = true
  try {
    await enablePerson(props.dn)
    disabled.value = false
    ElMessage.success('已启用')
    emit('saved', props.dn)
  } catch (e) { ElMessage.error(errText(e)) } finally { busy.value = false }
}
</script>

<style scoped>
.eh { display: flex; align-items: center; gap: 12px; }
.et { font-size: 16px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
.kind-tag { font-size: 11px; color: var(--el-color-primary); background: #f0f5ff; border-radius: 10px; padding: 1px 8px; font-weight: 400; }
.dn-crumb { font-size: 11px; color: #8f959e; display: flex; flex-wrap: wrap; gap: 4px; margin-top: 3px; }
.dn-crumb .seg { background: #f2f3f5; border-radius: 4px; padding: 0 6px; }
.dn-crumb .seg.base { background: #f0f5ff; color: var(--el-color-primary); }
.dn-crumb .sep { color: #c9cdd4; }
.sec-title { font-weight: 600; font-size: 13px; margin: 18px 0 8px; display: flex; align-items: center; gap: 7px; }
.sec-title::after { content: ''; flex: 1; height: 1px; background: var(--el-border-color-lighter); }
.sec-title .count { font-size: 11px; color: #8f959e; background: #f2f3f8; border-radius: 8px; padding: 0 7px; }
.muted { color: #8f959e; }
.en { font-size: 11px; font-weight: 400; }
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 12px; }
.pill { display: inline-flex; align-items: center; height: 18px; padding: 0 6px; border-radius: 9px; font-size: 10.5px; }
.pill.plain { background: #f2f3f5; color: #646a73; }
.pill.info { background: #f0f5ff; color: var(--el-color-primary); }
.attr-row { padding: 8px 0; border-bottom: 1px dashed var(--el-border-color-lighter); }
.groups-wall { display: flex; flex-wrap: wrap; }
.attr-label { display: flex; align-items: center; gap: 6px; font-size: 12.5px; margin-bottom: 6px; flex-wrap: wrap; }
.attr-label > span:first-child { font-weight: 500; }
.attr-values .val-line { display: flex; gap: 6px; margin-bottom: 4px; align-items: center; }
.attr-values.readonly .val-line-ro { padding: 4px 8px; background: #fafbfc; border-radius: 4px; margin-bottom: 4px; font-size: 11.5px; word-break: break-all; }
.add-attr { display: flex; gap: 8px; margin-top: 10px; padding: 10px; background: #fafbfc; border-radius: 8px; }
.attr-row.new-attr { background: #f7fbff; border: 1px dashed #adc6ff; border-radius: 6px; padding: 8px; }
.oc-list { display: flex; flex-direction: column; gap: 4px; margin-bottom: 8px; }
.oc-row { display: flex; align-items: center; gap: 8px; padding: 8px 10px; border: 1px solid var(--el-border-color-lighter); border-radius: 6px; }
.oc-main { display: flex; flex-direction: column; gap: 4px; margin-bottom: 10px; }
.oc-row.structural { background: #f0f5ff; border-color: #adc6ff; }
.oc-main-name { display: flex; align-items: center; gap: 8px; flex: 1; min-width: 0; }
.oc-name { font-size: 12.5px; }
.oc-cn { font-size: 12px; color: #646a73; }
.oc-change { display: flex; gap: 8px; margin: 6px 0 14px; }
.oc-row .pill { margin-left: auto; flex: none; }
.oc-row .pill + .muted { margin-left: 8px; }
.qmark { font-size: 13px; color: #8f959e; cursor: help; flex: none; }
.qmark:hover { color: var(--el-color-primary); }
.oc-row.structural { background: #f0f5ff; border-color: #adc6ff; }
.ldif { background: #101a3e; color: #d6defa; padding: 14px; border-radius: 8px; font-size: 11.5px; line-height: 1.8; overflow: auto; max-height: 520px; font-family: ui-monospace, Consolas, monospace; white-space: pre-wrap; word-break: break-all; }
.footer { display: flex; justify-content: space-between; width: 100%; }
.footer .left { display: flex; gap: 8px; }
.expert-tabs { position: relative; }
.expert-tabs :deep(.el-tabs__header) { margin-bottom: 4px; }
.expert-tabs .back-simple { position: absolute; right: 0; top: 2px; z-index: 2; padding: 4px 6px; }
.expert-tabs :deep(.sec-title) { margin-top: 12px; }
.pw { font-size: 18px; letter-spacing: 1px; text-align: center; margin: 8px 0; }
</style>
