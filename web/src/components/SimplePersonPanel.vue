<template>
  <!-- 简单模式：业务字段卡片（新手友好，贴合"人"的视角） -->
  <div class="simple">
    <div class="s-head">
      <span class="nl-avatar lg">{{ (title || '?')[0] }}</span>
      <div>
        <div class="s-name">{{ title }}</div>
        <div class="s-sub mono">{{ detail?.dn }}</div>
      </div>
      <span v-if="isPerson" class="nl-pill" :class="disabled ? 'plain' : 'ok'">
        <span class="dot" />{{ disabled ? '已禁用' : '在职' }}
      </span>
    </div>

    <el-alert v-if="conflict" type="error" show-icon :closable="false" style="margin-bottom: 12px"
      title="该条目刚被其他人修改过，已自动刷新为最新版本，请重新调整后再保存" />

    <!-- 非人员条目（组/角色等）：不渲染人员字段，避免把 uid/工号写进组 -->
    <template v-if="!isPerson">
      <div class="sec-title">条目信息 <span class="muted" style="font-weight: 400">（{{ kindLabel }}）</span></div>
      <!-- 对象类徽章：中文名 + 英文名，悬停显示说明 -->
      <div class="oc-badges" v-if="!isPerson && detail?.attrs?.objectClass">
        <el-tooltip v-for="oc in (detail.attrs.objectClass ?? []).filter(c => c !== 'top')" :key="oc"
          :content="classDesc(oc) ?? oc" placement="top" :show-after="300">
          <span class="nl-pill" :class="{ info: oc === (detail.attrs.objectClass ?? []).find(c => c !== 'top') }">
            {{ classCN(oc) ?? oc }}
            <span class="mono" style="font-size: 10px; margin-left: 3px; opacity: 0.7">{{ oc }}</span>
          </span>
        </el-tooltip>
      </div>
      <el-form label-position="top">
        <el-form-item label="名称 (cn)">
          <!-- 角色/服务账号（organizationalRole，如 admin、对接只读账号）支持改名：改 cn = 改 RDN -->
          <el-input v-if="isRole" v-model="roleCN" class="mono" placeholder="条目名称" />
          <el-input v-else :model-value="title" readonly class="mono" />
          <div v-if="isRole" class="muted" style="font-size: 11px; margin-top: 3px">
            名称即登录标识的一部分；修改将同时更新条目 DN（如 cn=admin → cn=新名）
          </div>
          <div v-else class="muted" style="font-size: 11px; margin-top: 3px">组的名称在「用户组」页管理；重命名请使用专家模式</div>
        </el-form-item>
        <el-form-item label="说明 (description)">
          <el-input v-model="otherDesc" placeholder="一句话说明（选填）" />
        </el-form-item>
        <el-form-item v-if="memberCount > 0" :label="isLoginGroup ? '成员账号 (memberUid)' : '成员 (member)'">
          <div class="groups">
            <el-tag v-for="m in memberList" :key="m" effect="plain" style="margin: 0 6px 6px 0">{{ m }}</el-tag>
            <span v-if="memberCount > memberList.length" class="muted">等共 {{ memberCount }} 名成员</span>
          </div>
        </el-form-item>
      </el-form>
      <div class="nl-hint info" style="margin: 10px 0">
        <el-icon style="margin-top: 2px"><Info /></el-icon>
        <div>{{ kindHint }}</div>
      </div>
    </template>

    <!-- 人员：业务字段卡片 -->
    <template v-else>
    <div class="sec-title">基本信息</div>
    <el-form label-position="top">
      <el-form-item v-for="f in simpleFields" :key="f.name" :required="f.required">
        <template #label>{{ f.label }}<span v-if="f.required" class="req">*</span></template>
        <el-input v-model="f.edit" :placeholder="f.hint" :class="{ mono: f.name === 'uid' }" />
        <div v-if="f.name === 'uid'" class="muted" style="font-size: 11px; margin-top: 3px">
          登录账号（必填）；修改后条目 DN 会随之更新
        </div>
      </el-form-item>
      <el-form-item v-if="isPerson" label="所属组">        <div class="groups">
          <el-tag v-for="g in groups" :key="g.dn" :type="g.type === '登录组' ? 'info' : 'primary'"
            effect="plain" style="margin: 0 6px 6px 0">{{ g.name }} · {{ g.type }}</el-tag>
          <span v-if="!groups.length" class="muted">未加入任何组</span>
        </div>
      </el-form-item>
    </el-form>
    </template>

    <div v-if="extraCount > 0" style="text-align: center; margin: 8px 0 4px">
      <el-button text type="primary" @click="$emit('enter-expert')">
        更多属性（{{ extraCount }} 项）· 切换专家模式 {{ '›' }}
      </el-button>
    </div>

    <div class="s-footer">
      <el-button v-if="isGroup" size="small" @click="goGroups"><el-icon><UsersRound /></el-icon>&nbsp;去用户组页管理</el-button>
      <template v-if="isPerson">
        <el-button size="small" @click="doReset" :loading="busy">重置密码</el-button>
        <el-button size="small" v-if="!disabled" @click="doDisable">禁用账号</el-button>
        <el-button size="small" v-else @click="doEnable">启用账号</el-button>
      </template>
      <!-- 角色/服务账号（admin、对接只读账号等）：与人员同样支持改密（userPassword 属性写） -->
      <el-button v-if="isRole" size="small" @click="doReset" :loading="busy">修改密码</el-button>
      <span style="flex: 1"></span>
      <el-button @click="$emit('close')">关闭</el-button>
      <el-button type="primary" :loading="busy" @click="saveSimple">保存修改</el-button>
    </div>
  </div>

  </template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Info, UsersRound } from 'lucide-vue-next'
import {
  disablePerson, enablePerson, errText, getEntry, moveEntry, personDetail,
  resetPassword, searchEntries, updateEntry, type PersonDetail,
} from '../api'
import { baseDomain, policy } from '../store/app'
import { classCN, classDesc } from '../utils/ldapDict'

const props = defineProps<{ dn: string }>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', dn: string): void
  (e: 'enter-expert'): void
}>()



const detail = ref<PersonDetail | null>(null)
const busy = ref(false)
const loading = ref(false)
const conflict = ref(false)
const disabled = ref(false)

interface SimpleField { name: string; label: string; edit: string; multi: boolean; hint: string; required: boolean }
const fields = reactive<SimpleField[]>([])

// 简单模式字段集（业务视角，中英文对照；LDAP 原因字段在专家模式可见）。
// uid 位于姓名与工号之间：账号是登录身份，重要性仅次于姓名。
const SIMPLE_DEFS = [
  { name: 'cn', label: '姓名 (cn)', hint: '支持中文', required: true },
  { name: 'uid', label: '账号 (uid)', hint: '登录账号，如 zhangwei', required: true },
  { name: 'employeeNumber', label: '工号 (employeeNumber)', hint: '选填', required: false },
  { name: 'mail', label: '邮箱 (mail)', hint: '选填', required: false },
  { name: 'mobile', label: '手机 (mobile)', hint: '选填', required: false },
  { name: 'title', label: '职务 (title)', hint: '选填', required: false },
]

const title = computed(() => fields.find(f => f.name === 'cn')?.edit || detail.value?.attrs?.cn?.[0] || '')
// 必须按 objectClass 判定：组等非人员条目也会打开本面板，不能显示
// 「在职」徽章与重置密码/禁用（否则会把 userPassword 写到组条目上）
const isPerson = computed(() => (detail.value?.attrs?.objectClass ?? []).some(c => c === 'inetOrgPerson'))
const isLoginGroup = computed(() => (detail.value?.attrs?.objectClass ?? []).some(c => c === 'posixGroup'))
const isGroup = computed(() => (detail.value?.attrs?.objectClass ?? []).some(c => c === 'groupOfNames' || c === 'posixGroup'))
// 角色/服务账号（organizationalRole）：admin、对接只读账号等——支持改名与改密
const isRole = computed(() => !isPerson.value && (detail.value?.attrs?.objectClass ?? []).some(c => c === 'organizationalRole'))
const roleCN = ref('')
const otherDesc = ref('')
// 成员很多时只展示前 8 个，其余折叠为计数（避免标签铺满抽屉）
const memberList = computed(() =>
  ((isLoginGroup.value ? detail.value?.attrs?.memberUid : detail.value?.attrs?.member) ?? []).slice(0, 8))
const memberCount = computed(() => memberList.value.length)
const kindLabel = computed(() => {
  if (isGroup.value) return isLoginGroup.value ? '登录组 posixGroup' : '权限组 groupOfNames'
  if (isRole.value) return '角色 / 服务账号'
  const oc = (detail.value?.attrs?.objectClass ?? []).slice(-1)[0] ?? ''
  return '条目 ' + (classCN(oc) ? `${classCN(oc)}` : oc)
})
const kindHint = computed(() => isGroup.value
  ? '这是用户组，不是人员：成员维护请在「用户组」页进行；此处仅可修改说明。'
  : '这是目录条目，不是人员：人员专属字段已隐藏；完整编辑请切换专家模式。')
function goGroups() { window.location.hash = ''; window.location.assign('/#/groups') }
const groups = computed(() => detail.value?.groups ?? [])
const extraCount = computed(() => {
  if (!detail.value) return 0
  const simple = new Set([...SIMPLE_DEFS.map(d => d.name), 'objectClass', 'sn', 'uid', 'userPassword'])
  return Object.keys(detail.value.attrs).filter(k => !simple.has(k)).length
})
const simpleFields = computed(() => fields)

async function load() {
  if (!props.dn) return
  loading.value = true
  try {
    detail.value = await personDetail(props.dn)
    disabled.value = detail.value.disabled
    otherDesc.value = detail.value.attrs.description?.[0] ?? ''
    roleCN.value = detail.value.attrs.cn?.[0] ?? ''
    fields.length = 0
    for (const d of SIMPLE_DEFS) {
      fields.push({
        name: d.name, label: d.label, hint: d.hint, required: d.required,
        edit: detail.value.attrs[d.name]?.[0] ?? '',
        multi: d.name === 'mail',
      })
    }
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    loading.value = false
  }
}

// 冲突标记只在切换条目时复位（load 内复位会吞掉 409 后的冲突提示条）
watch(() => props.dn, () => { if (props.dn) { conflict.value = false; load() } }, { immediate: true })

// 邮箱自动拼接（系统设置开关）：编辑人员时只填工号、邮箱为空 → 工号@域名
watch(() => fields.find(f => f.name === 'employeeNumber')?.edit, (emp) => {
  if (!policy.value?.emailAutofill || !baseDomain.value) return
  const mail = fields.find(f => f.name === 'mail')
  const e = (emp ?? '').trim()
  if (mail && e && !mail.edit) mail.edit = `${e}@${baseDomain.value}`
})

async function saveSimple() {
  if (!detail.value) return
  // 非人员条目：角色支持改名（cn=RDN），其余只保存说明
  if (!isPerson.value) {
    const cur = detail.value.attrs.description?.[0] ?? ''
    const curCN = detail.value.attrs.cn?.[0] ?? ''
    const newCN = roleCN.value.trim()
    if (newCN && newCN !== curCN) {
      if (isGroup.value) {
        ElMessage.warning('组的名称请在「用户组」页管理')
        return
      }
      if (/[,.=+<>#;"\\/]/.test(newCN) || newCN.includes(' ')) {
        ElMessage.warning('名称不能包含 , . = + 空格 等特殊字符')
        return
      }
      try {
        const dup = await searchEntries({ filter: `(cn=${newCN})`, attr: ['dn'], limit: 5 })
        if ((dup.entries ?? []).some(x => x.dn !== detail.value?.dn)) {
          ElMessage.warning(`名称 ${newCN} 已被同层的其他条目使用，请换一个`)
          return
        }
      } catch { /* 查重失败交由服务端兜底 */ }
      try {
        await ElMessageBox.confirm(
          `改名 ${curCN} → ${newCN} 将同时更新条目 DN（如它是登录账号，登录名即刻生效）。继续？`,
          '修改名称', { type: 'warning' })
      } catch { return }
      busy.value = true
      try {
        const nd = await moveEntry(detail.value.dn, `cn=${newCN}`)
        // 改名后再保存说明（若同时改了）
        if (otherDesc.value.trim() !== cur) {
          const fresh = await getEntry(nd)
          await updateEntry(nd, fresh.entryCSN, [
            { op: 'replace', attr: 'description', vals: otherDesc.value.trim() ? [otherDesc.value.trim()] : [] },
          ])
        }
        ElMessage.success('已改名并保存')
        emit('saved', nd)
      } catch (e) {
        ElMessage.error(errText(e))
      } finally { busy.value = false }
      return
    }
    if (otherDesc.value.trim() === cur) { ElMessage.info('没有修改需要保存'); return }
    busy.value = true
    try {
      await updateEntry(detail.value.dn, detail.value.entryCSN, [
        { op: 'replace', attr: 'description', vals: otherDesc.value.trim() ? [otherDesc.value.trim()] : [] },
      ])
      ElMessage.success('已保存')
      emit('saved', detail.value.dn)
      load()
    } catch (e: any) {
      ElMessage.error(errText(e))
    } finally { busy.value = false }
    return
  }
  // 必填校验（姓名、账号）
  for (const f of fields) {
    if (f.required && !f.edit.trim()) {
      ElMessage.warning(`「${f.label}」为必填项`)
      return
    }
  }
  busy.value = true
  try {
    let dn = detail.value.dn
    let expectedCSN = detail.value.entryCSN

    // 账号 (uid) 变更 = 条目重命名（RDN 变化），不能按普通属性替换提交
    const uidField = fields.find(f => f.name === 'uid')
    const oldUID = detail.value.attrs.uid?.[0] ?? ''
    const newUID = (uidField?.edit ?? '').trim()
    if (newUID && oldUID && newUID !== oldUID) {
      if (/[,.=+<>#;"\\/]/.test(newUID)) {
        ElMessage.warning('账号不能包含 , . = + < > # ; " \\ / 等字符')
        return
      }
      // 改名前查重：uid 需全组织唯一（排除自身条目）
      try {
        const dup = await searchEntries({ filter: `(uid=${newUID})`, attr: ['cn'], limit: 5 })
        const takenByOther = (dup.entries ?? []).some(x => x.dn !== detail.value?.dn)
        if (takenByOther) {
          ElMessage.warning(`账号 ${newUID} 已被他人使用（uid 需全组织唯一）`)
          return
        }
      } catch { /* 查重失败交由服务端兜底 */ }
      try {
        await ElMessageBox.confirm(
          `修改账号 ${oldUID} → ${newUID} 将同时更新条目 DN（登录名即刻生效）。继续？`,
          '修改账号', { type: 'warning' })
      } catch { return }
      const res = await moveEntry(dn, `uid=${newUID}`)
      dn = res
      // 重命名会改变 entryCSN：重读最新版本再提交属性修改
      const fresh = await getEntry(dn)
      expectedCSN = fresh.entryCSN
    }

    const changes: Array<{ op: string; attr: string; vals: string[] }> = []
    for (const f of fields) {
      if (f.name === 'uid') continue // 已由重命名处理
      const cur = detail.value.attrs[f.name]?.[0] ?? ''
      const next = f.edit.trim()
      if (next !== cur) {
        // 邮箱多值：只改首个，保留其余
        if (f.multi && (detail.value.attrs[f.name]?.length ?? 0) > 1) {
          const rest = detail.value.attrs[f.name].slice(1)
          changes.push({ op: 'replace', attr: f.name, vals: [next, ...rest.filter(v => v !== next)] })
        } else {
          changes.push({ op: 'replace', attr: f.name, vals: next ? [next] : [] })
        }
      }
    }
    if (changes.length) {
      await updateEntry(dn, expectedCSN, changes)
    }
    ElMessage.success(changes.length || dn !== detail.value.dn ? '已保存' : '没有修改需要保存')
    emit('saved', dn)
    // DN 未变时原地刷新；变了则交由父组件把抽屉指向新 DN
    if (dn === detail.value.dn && changes.length) load()
  } catch (e: any) {
    if (e?.response?.status === 409) {
      conflict.value = true
      ElMessage.error('条目刚被他人修改，已刷新')
      load()
    } else {
      ElMessage.error(errText(e))
    }
  } finally {
    busy.value = false
  }
}

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
    ElMessage.success('已禁用（可随时启用恢复）')
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
.simple { padding: 0 2px; }
.oc-badges { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 14px; }
.s-head { display: flex; align-items: center; gap: 14px; margin-bottom: 18px; }
.s-name { font-size: 18px; font-weight: 600; }
.s-sub { font-size: 11px; color: #8f959e; margin-top: 3px; max-width: 440px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sec-title { font-weight: 600; font-size: 13px; margin: 14px 0 10px; }
.req { color: var(--el-color-danger); margin-left: 2px; }
.groups { display: flex; flex-wrap: wrap; }
.muted { color: #8f959e; font-size: 12px; }
.mono { font-family: ui-monospace, Consolas, monospace; }
.s-footer { display: flex; align-items: center; gap: 8px; margin-top: 16px; padding-top: 14px; border-top: 1px solid var(--el-border-color-lighter); }
</style>
