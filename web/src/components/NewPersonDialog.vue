<template>
  <el-dialog :model-value="open" title="新建人员" width="480px" @update:model-value="onDialogUpdate">
    <!-- 成功结果视图：在同一对话框内切换（嵌套 el-dialog 会把外层关闭动画卡死） -->
    <template v-if="done">
      <div class="nl-hint ok" style="margin-bottom: 14px">
        <el-icon style="margin-top: 2px"><CircleCheck /></el-icon>
        <div>人员已创建。初始密码仅显示这一次，请立即复制并安全转交本人。</div>
      </div>
      <div class="result">
        <div class="rrow"><span>姓名</span><b>{{ createdCN }}</b></div>
        <div class="rrow"><span>账号</span><b class="mono">{{ createdUID }}</b></div>
        <div class="rrow"><span>DN</span><span class="mono muted">{{ createdDN }}</span></div>
        <div class="rrow"><span>初始密码</span><b class="mono pw">{{ createdPW }}</b></div>
      </div>
    </template>
    <!-- 表单视图 -->
    <template v-else>
      <div class="nl-hint info" style="margin-bottom: 14px">
        <el-icon style="margin-top: 2px"><Lightbulb /></el-icon>
        <div>填<b>姓名</b>、<b>账号</b>和<b>部门</b>即可。账号可用右侧按钮按姓名拼音生成（张伟 → zhangwei）；其他都是选填。</div>
      </div>
      <el-form label-position="top" :model="form">
        <el-form-item label="姓名 (cn) *" required>
          <el-input v-model="form.cn" placeholder="支持中文，如：许倩" @keyup.enter="submit" />
        </el-form-item>
        <el-form-item label="账号 (uid) *" required>
          <el-input v-model="form.uid" class="mono" placeholder="用于登录，如：zhangwei" :maxlength="policy?.uidMaxLen ?? 32">
            <template #append>
              <el-button :loading="suggesting" @click="fillUID" title="按姓名拼音生成并查重">按姓名生成</el-button>
            </template>
          </el-input>
          <span class="muted">必填；建议英文/数字（{{ policy?.uidMinLen ?? 2 }}-{{ policy?.uidMaxLen ?? 32 }} 位），用于登录，创建后不再变化</span>
          <div v-if="uidTaken" class="uid-taken">该账号已被使用（uid 需全组织唯一），请换一个</div>
        </el-form-item>
        <el-form-item label="所属部门 (ou) *" required>
          <el-tree-select v-model="form.ou" :data="ouOptions" node-key="dn" check-strictly
            :props="{ label: 'name' }" default-expand-all :render-after-expand="false"
            style="width: 100%" placeholder="选择部门" />
        </el-form-item>
        <el-collapse style="border: 0">
          <el-collapse-item title="选填：更多信息" name="more">
            <el-form-item label="工号 (employeeNumber)"><el-input v-model="form.employeeNumber" class="mono" /></el-form-item>
            <el-form-item label="邮箱 (mail)">
              <el-input v-model="form.mail" class="mono" :placeholder="policy?.emailAutofill && baseDomain ? `留空将自动生成 工号@${baseDomain}` : '选填'" />
              <span v-if="policy?.emailAutofill && baseDomain" class="muted">仅填工号未填邮箱时自动拼接，手动输入优先</span>
            </el-form-item>
            <el-form-item label="手机 (mobile)"><el-input v-model="form.mobile" class="mono" /></el-form-item>
            <el-form-item label="职务 (title)"><el-input v-model="form.title" /></el-form-item>
          </el-collapse-item>
        </el-collapse>
      </el-form>
    </template>
    <template #footer>
      <template v-if="done">
        <el-button @click="copy"><el-icon><Copy /></el-icon>&nbsp;复制密码</el-button>
        <el-button type="primary" @click="finish">完成</el-button>
      </template>
      <template v-else>
        <el-button @click="$emit('close')">取消</el-button>
        <el-button type="primary" :loading="busy" :disabled="!form.cn || !form.uid || !form.ou || uidTaken" @click="submit">创建</el-button>
      </template>
    </template>
  </el-dialog>
</template>
<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheck, Copy, Lightbulb } from 'lucide-vue-next'
import { createPerson, errText, ouTree, searchEntries, uidSuggest, type TreeNode } from '../api'
import { baseDomain, policy } from '../store/app'

const props = defineProps<{ open: boolean; defaultOu?: string }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'created', dn: string): void }>()

const form = reactive({
  cn: '', uid: '', employeeNumber: '', title: '', mail: '', mobile: '',
  ou: props.defaultOu ?? '', initialPassword: '',
})
const busy = ref(false)
const suggesting = ref(false)
const done = ref(false)
const createdDN = ref('')
const createdPW = ref('')
const createdUID = ref('')
const createdCN = ref('')

const ouOptions = ref<TreeNode[]>([])

// uid 全组织唯一：输入防抖 400ms 后查重，命中即时提示并阻止创建
const uidTaken = ref(false)
let uidSeq = 0
watch(() => form.uid, async (v) => {
  const val = (v ?? '').trim()
  uidTaken.value = false
  if (!val || !/^[a-zA-Z0-9._-]+$/.test(val)) return
  const seq = ++uidSeq
  await new Promise(r => setTimeout(r, 400))
  if (seq !== uidSeq) return
  try {
    const res = await searchEntries({ filter: `(uid=${val})`, attr: ['cn'], limit: 2 })
    if (seq === uidSeq) uidTaken.value = res.total > 0
  } catch { /* 查重失败不阻塞 */ }
})

// 邮箱自动拼接（系统设置开关）：只填工号、邮箱为空 → 工号@域名（dc 部分拼接）
watch(() => form.employeeNumber, (emp) => {
  if (!policy.value?.emailAutofill || !baseDomain.value) return
  const e = (emp ?? '').trim()
  if (e && !form.mail) form.mail = `${e}@${baseDomain.value}`
})

watch(() => props.open, async (v) => {
  if (v) {
    form.cn = ''; form.uid = ''; form.employeeNumber = ''; form.title = ''
    form.mail = ''; form.mobile = ''; form.initialPassword = ''
    form.ou = '' // 先空：选项未就绪时 tree-select 会把找不到节点的值清空
    done.value = false
    if (!ouOptions.value.length) {
      try { ouOptions.value = await ouTree() } catch { ouOptions.value = [] }
    }
    // 选项就绪后再设预设部门
    form.ou = props.defaultOu ?? ''
  }
})

// 成功视图下关闭（X/遮罩）与表单视图一样走 close
function onDialogUpdate(v: boolean) {
  if (!v) finish()
}

// 「按姓名生成」：姓名拼音 → 服务端查重加序号后回填
async function fillUID() {
  if (!form.cn.trim()) {
    ElMessage.warning('请先填写姓名')
    return
  }
  suggesting.value = true
  try {
    const { uid } = await uidSuggest(form.cn.trim(), form.ou || props.defaultOu || undefined)
    form.uid = uid
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    suggesting.value = false
  }
}

function finish() {
  done.value = false
  emit('close')
}

function gen() {
  const sets = ['abcdefghijkmnpqrstuvwxyz', 'ABCDEFGHJKLMNPQRSTUVWXYZ', '23456789', '!@#$%&*']
  let pw = ''
  sets.forEach((s) => { pw += s[Math.floor(Math.random() * s.length)] })
  while (pw.length < 12) {
    const all = sets.join('')
    pw += all[Math.floor(Math.random() * all.length)]
  }
  form.initialPassword = pw.split('').sort(() => Math.random() - 0.5).join('')
}


async function submit() {
  // 账号长度策略（服务端同样强制，这里提前拦截给出即时反馈）
  const p = policy.value
  const uidLen = form.uid.trim().length
  if (p && (uidLen < p.uidMinLen || uidLen > p.uidMaxLen)) {
    ElMessage.warning(`账号长度需在 ${p.uidMinLen}-${p.uidMaxLen} 个字符之间（系统设置）`)
    return
  }
  busy.value = true
  try {
    const res = await createPerson({ ...form, ou: form.ou || props.defaultOu || '' })
    createdDN.value = res.dn
    createdPW.value = res.initialPassword
    createdUID.value = res.dn.split(',')[0].replace('uid=', '')
    createdCN.value = form.cn
    done.value = true // 同一对话框切换到成功视图（不关窗、不嵌套弹窗）
    emit('created', res.dn)
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    busy.value = false
  }
}

function copy() {
  navigator.clipboard?.writeText(createdPW.value)
  ElMessage.success('已复制，请安全转交')
}
</script>

<style scoped>
.mono { font-family: Consolas, monospace; font-size: 12px; }
.muted { color: #8f959e; font-size: 12px; }
.uid-taken { color: var(--el-color-danger); font-size: 12px; margin-top: 2px; }
.sec-title { font-weight: 600; margin: 12px 0 4px; }
.result { display: flex; flex-direction: column; gap: 10px; padding: 14px; border: 1px solid var(--el-border-color-lighter); border-radius: 8px; background: #fafbfc; }
.rrow { display: flex; align-items: baseline; gap: 12px; font-size: 13px; }
.rrow > span:first-child { width: 64px; flex: none; color: #8f959e; }
.rrow .muted { font-size: 11px; word-break: break-all; }
.pw { font-size: 16px; letter-spacing: 1px; }
</style>
