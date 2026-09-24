<template>
  <el-drawer :model-value="open" size="560px" :title="detail?.attrs?.cn?.[0] ?? dn" @update:model-value="$emit('close')">
    <div v-if="detail" v-loading="loading">
      <el-tabs v-model="tab">
        <!-- 动态表单（R5.2） -->
        <el-tab-pane label="基本信息" name="form">
          <div class="dn mono">{{ detail.dn }}</div>
          <el-alert v-if="conflict" type="error" show-icon :closable="false"
            title="条目已被其他客户端修改（如 LAM），请刷新后重试" style="margin-bottom: 12px" />
          <el-form label-position="top">
            <el-form-item v-for="f in formFields" :key="f.name" :label="f.label + (f.required ? ' *' : '')">
              <el-input v-if="isSimple(f)" v-model="singleValues[f.name]" :placeholder="f.control" />
              <el-input v-else-if="f.control === 'textarea'" v-model="singleValues[f.name]" type="textarea" :rows="2" />
              <el-input-number v-else-if="f.control === 'number'" v-model="numValues[f.name]" class="w100" />
              <el-switch v-else-if="f.control === 'boolean'" v-model="boolValues[f.name]" />
              <div v-else class="multi">
                <el-tag v-for="(v, i) in multiValues[f.name] ?? []" :key="i" closable style="margin-right: 6px"
                  @close="removeValue(f.name, i)">{{ v }}</el-tag>
                <el-input v-if="adding === f.name" v-model="addVal" size="small" style="width: 160px"
                  @keyup.enter="commitAdd(f.name)" @blur="commitAdd(f.name)" />
                <el-button v-else size="small" @click="startAdd(f.name)">+ 添加</el-button>
              </div>
            </el-form-item>
          </el-form>
          <div class="sec-title">所属组（{{ detail.groups.length }}）</div>
          <div class="groups">
            <el-tag v-for="g in detail.groups" :key="g.dn" :type="g.type === '登录组' ? 'info' : 'primary'"
              effect="plain" style="margin: 0 6px 6px 0">{{ g.name }} · {{ g.type }}</el-tag>
            <span v-if="!detail.groups.length" class="muted">未加入任何组</span>
          </div>
        </el-tab-pane>

        <!-- 专家模式（R5.3） -->
        <el-tab-pane label="专家模式" name="expert">
          <div class="sec-title">objectClass</div>
          <el-select v-model="expertClasses" multiple filterable allow-create style="width: 100%" placeholder="选择或输入 objectClass" />
          <div class="sec-title">原始属性</div>
          <table class="raw">
            <tr v-for="(v, k) in expertAttrs" :key="k">
              <td class="k mono">{{ k }}</td>
              <td><el-input v-for="(val, i) in v" :key="i" v-model="v[i]" size="small" style="margin-bottom: 4px"
                  @change="touched.add(String(k))" />
                <el-button size="small" text type="danger" @click="removeAttr(k)">删除属性</el-button>
              </td>
            </tr>
          </table>
          <el-form inline style="margin-top: 8px">
            <el-form-item><el-input v-model="newAttr.name" placeholder="属性名" size="small" class="w120 mono" /></el-form-item>
            <el-form-item><el-input v-model="newAttr.value" placeholder="值" size="small" style="width: 180px" /></el-form-item>
            <el-form-item><el-button size="small" @click="addAttr">添加</el-button></el-form-item>
          </el-form>
          <div class="sec-title">LDIF 视图（只读）</div>
          <pre class="ldif">{{ ldifView }}</pre>
          <el-alert type="info" :closable="false" show-icon
            title="保存按属性级提交并与 entryCSN 比对；不会触碰未修改的字段" />
        </el-tab-pane>
      </el-tabs>
    </div>
    <template #footer>
      <div class="footer">
        <div class="left">
          <el-button size="small" @click="doReset" :loading="busy">重置密码</el-button>
          <el-button size="small" @click="doDisable" :loading="busy" v-if="!disabled">{{ '禁用账号' }}</el-button>
          <el-button size="small" @click="doEnable" :loading="busy" v-else>启用账号</el-button>
          <el-button size="small" @click="moveOpen = true">调动部门</el-button>
        </div>
        <div>
          <el-button @click="$emit('close')">取消</el-button>
          <el-button type="primary" :loading="busy" @click="save">保存修改</el-button>
        </div>
      </div>
      <!-- 重置密码结果 -->
      <el-dialog v-model="pwResult" title="新密码（仅显示一次）" width="380px" append-to-body>
        <p class="mono pw">{{ pwShown }}</p>
        <p class="muted">请立即安全转交该员工；密码不会再次显示，也不写入审计日志。</p>
        <template #footer>
          <el-button @click="copyPw">复制</el-button>
          <el-button type="primary" @click="pwResult = false">我已转交</el-button>
        </template>
      </el-dialog>
      <!-- 调动部门 -->
      <el-dialog v-model="moveOpen" title="调动部门" width="480px" append-to-body>
        <p class="muted">在目录树中选择新的所属部门（modrdn 移动，属性全保留）：</p>
        <el-tree-select v-model="moveTarget" :load="loadNodes" lazy check-strictly :render-after-expand="false"
          node-key="dn" :props="{ label: 'name', isLeaf: (d: any) => !d.hasChildren }" style="width: 100%" />
        <template #footer>
          <el-button @click="moveOpen = false">取消</el-button>
          <el-button type="primary" :disabled="!moveTarget" @click="doMove">调动</el-button>
        </template>
      </el-dialog>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  disablePerson, enablePerson, errText, personDetail, resetPassword, schemaForm, setDept,
  tree, updateEntry, type FormField, type FormModel, type PersonDetail,
} from '../api'

const props = defineProps<{ open: boolean; dn: string; disabled?: boolean; initialTab?: 'form' | 'expert' }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'saved', dn: string): void; (e: 'moved', dn: string): void }>()

const loading = ref(false)
const busy = ref(false)
const detail = ref<PersonDetail | null>(null)
const formModel = ref<FormModel | null>(null)
const tab = ref('form')
const conflict = ref(false)
const disabled = ref(false)

const formFields = computed(() => formModel.value?.fields ?? [])
const isSimple = (f: FormField) =>
  !f.multi && (f.control === 'text' || f.control === 'email' || f.control === 'tel' || f.control === 'datetime')

const singleValues = reactive<Record<string, string>>({})
const multiValues = reactive<Record<string, string[]>>({})
const numValues = reactive<Record<string, number | undefined>>({})
const boolValues = reactive<Record<string, boolean>>({})

// 专家模式状态
const expertClasses = ref<string[]>([])
const expertAttrs = reactive<Record<string, string[]>>({})
const touched = reactive(new Set<string>())
const removed = ref(new Set<string>())
const newAttr = reactive({ name: '', value: '' })

const adding = ref('')
const addVal = ref('')

const pwResult = ref(false)
const pwShown = ref('')
const moveOpen = ref(false)
const moveTarget = ref('')

async function load() {
  if (!props.dn) return
  loading.value = true
  conflict.value = false
  try {
    detail.value = await personDetail(props.dn)
    formModel.value = await schemaForm('inetOrgPerson')
    for (const k of Object.keys(singleValues)) delete singleValues[k]
    for (const k of Object.keys(multiValues)) delete multiValues[k]
    for (const k of Object.keys(numValues)) delete numValues[k]
    for (const k of Object.keys(boolValues)) delete boolValues[k]
    for (const f of formModel.value.fields) {
      const cur = detail.value.attrs[f.name] ?? []
      if (f.control === 'number') numValues[f.name] = cur[0] ? Number(cur[0]) : undefined
      else if (f.control === 'boolean') boolValues[f.name] = cur[0] === 'TRUE'
      else if (f.multi) multiValues[f.name] = [...cur]
      else singleValues[f.name] = cur[0] ?? ''
    }
    // 专家模式
    expertClasses.value = [...(detail.value.attrs.objectClass ?? [])]
    for (const k of Object.keys(expertAttrs)) delete expertAttrs[k]
    for (const [k, v] of Object.entries(detail.value.attrs)) {
      if (k.toLowerCase() === 'objectclass') continue
      expertAttrs[k] = [...v]
    }
    touched.clear()
    removed.value = new Set()
    disabled.value = detail.value.disabled // 服务端判断禁用态，密码不回传
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    loading.value = false
  }
}

watch(() => [props.open, props.dn], ([open]) => { if (open) { tab.value = props.initialTab ?? 'form'; load() } }, { immediate: true })

const ldifView = computed(() => {
  if (!detail.value) return ''
  const lines = [`dn: ${detail.value.dn}`, ...expertClasses.value.map((c) => `objectClass: ${c}`)]
  for (const [k, v] of Object.entries(expertAttrs)) {
    for (const val of v) lines.push(`${k}: ${val}`)
  }
  return lines.join('\n')
})

function startAdd(name: string) { adding.value = name; addVal.value = '' }
function commitAdd(name: string) {
  const v = addVal.value.trim()
  if (v && Array.isArray(multiValues[name])) multiValues[name].push(v)
  adding.value = ''
}
function removeValue(name: string, i: number) {
  multiValues[name]?.splice(i, 1)
}

function addAttr() {
  const k = newAttr.name.trim()
  if (!k) return
  expertAttrs[k] = [...(expertAttrs[k] ?? []), newAttr.value]
  touched.add(k)
  newAttr.name = ''; newAttr.value = ''
}
function removeAttr(k: string) {
  removed.value.add(k)
  touched.add(k)
  delete expertAttrs[k]
}

async function save() {
  if (!detail.value || !formModel.value) return
  busy.value = true
  try {
    const changes: Array<{ op: string; attr: string; vals: string[] }> = []
    if (tab.value === 'expert') {
      for (const k of removed.value) changes.push({ op: 'delete', attr: k, vals: [] })
      for (const k of touched) {
        if (removed.value.has(k)) continue
        changes.push({ op: 'replace', attr: k, vals: expertAttrs[k] ?? [] })
      }
      const orig = detail.value.attrs.objectClass ?? []
      if (JSON.stringify(orig) !== JSON.stringify(expertClasses.value)) {
        changes.push({ op: 'replace', attr: 'objectClass', vals: expertClasses.value })
      }
    } else {
      for (const f of formFields.value) {
        let next: string[]
        if (f.control === 'number') next = numValues[f.name] != null ? [String(numValues[f.name])] : []
        else if (f.control === 'boolean') next = [boolValues[f.name] ? 'TRUE' : 'FALSE']
        else if (f.multi) next = multiValues[f.name] ?? []
        else next = singleValues[f.name] ? [singleValues[f.name]] : []
        const prev = detail.value.attrs[f.name] ?? []
        if (JSON.stringify(prev) !== JSON.stringify(next)) {
          changes.push({ op: 'replace', attr: f.name, vals: next })
        }
      }
    }
    if (!changes.length) {
      ElMessage.info('没有修改需要保存')
      return
    }
    await updateEntry(detail.value.dn, detail.value.entryCSN, changes)
    ElMessage.success('已保存（属性级修改，LAM 侧实时可见）')
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

async function doReset() {
  busy.value = true
  try {
    const { password } = await resetPassword(props.dn)
    pwShown.value = password
    pwResult.value = true
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    busy.value = false
  }
}

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
    ElMessage.success('已启用，原密码恢复')
    emit('saved', props.dn)
  } catch (e) { ElMessage.error(errText(e)) } finally { busy.value = false }
}

async function loadNodes(node: any, resolve: (d: any[]) => void) {
  const base = node.level === 0 ? '' : node.data.dn
  try {
    const kids = await tree(base || (await (await import('../api')).me()).baseDN)
    resolve(kids.filter((k) => k.rdn.startsWith('ou=')))
  } catch { resolve([]) }
}

async function doMove() {
  busy.value = true
  try {
    const { dn } = await setDept(props.dn, moveTarget.value)
    ElMessage.success('已调动至 ' + dn)
    moveOpen.value = false
    emit('moved', dn)
    emit('close')
  } catch (e) { ElMessage.error(errText(e)) } finally { busy.value = false }
}
</script>

<style scoped>
.dn { color: #8f959e; font-size: 11.5px; margin-bottom: 14px; }
.sec-title { font-weight: 600; font-size: 13px; margin: 16px 0 8px; }
.groups { display: flex; flex-wrap: wrap; }
.muted { color: #8f959e; font-size: 12px; }
.mono { font-family: Consolas, monospace; }
.multi { display: flex; align-items: center; flex-wrap: wrap; gap: 4px; }
.raw { width: 100%; border-collapse: collapse; }
.raw td { vertical-align: top; padding: 4px 6px 4px 0; }
.raw .k { font-size: 12px; width: 130px; color: #1f2329; }
.ldif { background: #101a3e; color: #d6defa; padding: 12px; border-radius: 8px; font-size: 11.5px; line-height: 1.8; overflow: auto; max-height: 240px; font-family: Consolas, monospace; }
.footer { display: flex; justify-content: space-between; width: 100%; }
.footer .left { display: flex; gap: 8px; }
.w100 { width: 100%; }
.w120 { width: 120px; }
.pw { font-size: 18px; letter-spacing: 1px; text-align: center; margin: 8px 0; }
</style>
