<template>
  <div class="page">
    <div class="wizard">
      <div v-if="status.configured" class="nl-hint warn" style="margin-bottom: 16px">
        <el-icon style="margin-top: 2px"><TriangleAlert /></el-icon>
        <div>
          <b>当前已配置连接：{{ status.profile?.url }}（Base DN：{{ status.profile?.baseDN }}）。</b>
          继续保存将<b>替换</b>这份连接档案，已登录会话全部失效，需用新目录的账号重新登录。
          <template v-if="status.source === 'env'"><br />注意：该连接由部署环境变量（LDAP_URL 等）指定——网页修改仅在本次进程生效，<b>重启容器后仍以环境变量为准</b>；如需永久更换请修改部署环境变量。</template>
        </div>
      </div>
      <div class="head">
        <h1>连接目录服务器</h1>
        <p>{{ status.configured ? '重新配置：换一台目录服务器，或修正现有连接参数' : '首次使用，先告诉工具你的 OpenLDAP 在哪里（R1.3 连接向导）' }}</p>
      </div>

      <el-steps :active="step" align-center finish-status="success" style="margin-bottom: 26px">
        <el-step title="服务器地址" />
        <el-step title="探测与验证" />
        <el-step title="保存档案" />
      </el-steps>

      <!-- 第 1 步：地址 -->
      <template v-if="step === 0">
        <el-form label-position="top">
          <el-form-item label="档案名称">
            <el-input v-model="form.name" placeholder="例如：公司主目录" />
          </el-form-item>
          <el-form-item label="服务器地址">
            <el-input v-model="form.url" class="mono" placeholder="ldap://192.168.1.10:389 或 ldaps://…:636" @keyup.enter="doProbe" />
          </el-form-item>
          <el-form-item label="连接安全">
            <el-radio-group v-model="tlsMode">
              <el-radio value="plain">明文（ldap://）</el-radio>
              <el-radio value="starttls">StartTLS 升级</el-radio>
              <el-radio value="ldaps">LDAPS（地址前缀 ldaps://）</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-checkbox v-model="form.insecureSkipVerify">跳过证书校验（仅测试环境使用）</el-checkbox>
        </el-form>
        <div class="tip">
          提示：DN 即"条目在目录树中的完整路径"；Base DN 是你要管理的子树根，下一步会自动探测候选值。
        </div>
        <div class="actions">
          <el-button type="primary" :loading="probing" @click="doProbe">连接并探测</el-button>
        </div>
      </template>

      <!-- 第 2 步：探测结果 + bind -->
      <template v-if="step === 1">
        <el-descriptions :column="1" border size="small" style="margin-bottom: 16px">
          <el-descriptions-item label="服务器">{{ form.url }}</el-descriptions-item>
          <el-descriptions-item label="厂商标识">{{ pr?.vendorName || '（未提供）' }}</el-descriptions-item>
          <el-descriptions-item label="服务端能力">
            <el-tag size="small" :type="pr?.paging ? 'success' : 'info'">分页 {{ pr?.paging ? '支持' : '不支持' }}</el-tag>
            <el-tag size="small" :type="pr?.passwordModify ? 'success' : 'info'" style="margin-left: 6px">
              密码修改扩展 {{ pr?.passwordModify ? '支持' : '不支持' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <el-form label-position="top">
          <el-form-item label="Base DN（要管理的子树根）">
            <el-radio-group v-if="namingCtx.length" v-model="form.baseDN" style="display: flex; flex-direction: column; gap: 6px">
              <el-radio v-for="nc in namingCtx" :key="nc" :value="nc">
                <span class="mono">{{ nc }}</span>
                <span class="muted">（服务器命名上下文）</span>
              </el-radio>
            </el-radio-group>
            <el-input v-else v-model="form.baseDN" class="mono" placeholder="dc=你的,dc=域名（如 dc=corp,dc=com）" />
          </el-form-item>
          <el-form-item label="管理账号 Bind DN（用于连接档案与服务器探活，可不填）">
            <el-input v-model="form.bindDN" class="mono" :placeholder="form.baseDN ? `cn=admin,${form.baseDN}` : 'cn=admin,dc=…（跟随上方 Base DN）'" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.bindPassword" type="password" show-password @keyup.enter="testBind" />
          </el-form-item>
        </el-form>

        <el-alert v-if="bindState === 'ok'" type="success" :closable="false" show-icon style="margin-bottom: 12px"
          title="账号验证通过" :description="baseDesc" />
        <el-alert v-else-if="bindState === 'fail'" type="error" :closable="false" show-icon style="margin-bottom: 12px"
          :title="bindError || '账号验证失败'" />

        <div class="actions">
          <el-button @click="step = 0">上一步</el-button>
          <el-button :loading="probing" @click="testBind">验证账号</el-button>
          <el-button type="primary" @click="step = 2">下一步</el-button>
        </div>
      </template>

      <!-- 第 3 步：保存 -->
      <template v-if="step === 2">
        <el-descriptions :column="1" border size="small" style="margin-bottom: 16px">
          <el-descriptions-item label="Base DN"><span class="mono">{{ form.baseDN }}</span></el-descriptions-item>
          <el-descriptions-item label="目录状态">{{ baseStateText }}</el-descriptions-item>
          <el-descriptions-item label="管理账号"><span class="mono">{{ form.bindDN || '（未配置）' }}</span></el-descriptions-item>
        </el-descriptions>

        <el-checkbox v-model="form.rememberPassword">记住管理账号密码（加密保存，用于服务器探活）</el-checkbox>
        <el-alert v-if="!status.secretKeySet" type="warning" :closable="false" show-icon style="margin-top: 12px"
          title="未设置 NEWLDAP_SECRET_KEY 环境变量"
          description="密码将以明文写入配置文件。生产环境建议设置密钥后重新保存（R11.2 凭据加密）。" />

        <div class="actions">
          <el-button @click="step = 1">上一步</el-button>
          <el-button type="primary" :loading="saving" @click="save">保存并进入</el-button>
        </div>
      </template>

      <!-- 第 4 步：空目录初始化 + 结构模板（R1.5 / R5.5 / R10.2） -->
      <template v-if="step === 3">
        <el-result icon="success" title="连接已保存" sub-title="检测到这是空目录 —— 先搭好“房间结构”">
          <template #extra>
            <p class="muted" style="max-width: 460px">
              目录好比一套空房子：先选一个结构模板把部门骨架搭起来（之后可自由增删改），
              然后在管理台里添加人员，或用 Excel 批量导入。
            </p>
            <div class="tpl-grid">
              <div v-for="t in templates" :key="t.id" class="tpl" :class="{ on: tpl === t.id }" @click="tpl = t.id">
                <div class="tt">{{ t.name }}<span v-if="t.tag" class="nl-pill info" style="margin-left: auto">{{ t.tag }}</span></div>
                <pre class="mini mono">{{ t.tree }}</pre>
                <p class="muted">{{ t.desc }}</p>
              </div>
            </div>
            <div style="display: flex; gap: 10px; justify-content: center; margin-top: 6px">
              <el-button type="primary" :loading="bootstrapping" @click="createBase">创建基础条目与结构</el-button>
              <el-button @click="goAdmin">跳过，直接进入管理台</el-button>
            </div>
          </template>
        </el-result>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { TriangleAlert } from 'lucide-vue-next'
import {
  createEntry, errText, login, probe, rdnAttr, rdnValue, saveProfile, setupStatus,
  type ProbeResult, type SetupStatus,
} from '../api'

const router = useRouter()
const step = ref(0)
const status = reactive<SetupStatus>({ configured: false, mock: false, secretKeySet: false })
const form = reactive({
  name: '默认连接',
  url: '',
  startTLS: false,
  insecureSkipVerify: false,
  bindDN: '',
  bindPassword: '',
  baseDN: '',
  rememberPassword: true,
})
const tlsMode = ref<'plain' | 'starttls' | 'ldaps'>('plain')
const pr = ref<ProbeResult | null>(null)
const probing = ref(false)
const saving = ref(false)
const bootstrapping = ref(false)
const tpl = ref<'lam' | 'dept' | 'simple'>('dept')

// 结构模板（R5.5）：LAM 经典 / 企业部门制 / 极简。
// 树形示意跟随实际探测/填写的 Base DN（不同部署 dc 各异，不写死示例域名）。
const templates = computed(() => {
  const b = form.baseDN || 'dc=…'
  return [
    { id: 'lam', name: 'LAM 经典结构', tag: '并存推荐', tree: `${b}\n├─ ou=people（全部人员）\n├─ ou=groups（用户组）\n└─ ou=services（服务账号）`, desc: 'LAM 等传统工具熟悉的经典布局；已用或将用 LAM 的团队选这个。' },
    { id: 'dept', name: '企业部门制', tag: '', tree: `${b}\n├─ ou=技术研发部\n├─ ou=产品设计部\n├─ ou=市场运营部\n└─ ou=groups（用户组）`, desc: '人员直接挂在所属部门下，树即组织架构，最符合直觉（默认推荐）。' },
    { id: 'simple', name: '极简结构', tag: '', tree: `${b}\n├─ ou=people\n└─ ou=groups`, desc: '先跑起来再说，之后随时补建部门。' },
  ] as const
})

function tplOUs(id: string, base: string): Array<[string, string]> {
  if (id === 'lam') return [['people', base], ['groups', base], ['services', base]]
  if (id === 'simple') return [['people', base], ['groups', base]]
  return [['技术研发部', base], ['产品设计部', base], ['市场运营部', base], ['groups', base]]
}
const bindState = ref<'' | 'ok' | 'fail'>('')
const bindError = ref('')

const namingCtx = computed(() => pr.value?.namingContexts ?? [])

const baseDesc = computed(() => {
  const b = pr.value?.base
  if (!b) return ''
  if (!b.exists) return '该 Base DN 尚不存在（空目录）——保存后可创建基础条目。'
  return b.hasChildren ? '目录已有数据（存量目录），登录后直接使用。' : 'Base DN 存在但暂无子条目。'
})
const baseStateText = computed(() => {
  if (bindState.value !== 'ok') return '（未验证）'
  return baseDesc.value
})

function applyTls() {
  form.startTLS = tlsMode.value === 'starttls'
}

async function doProbe() {
  if (!form.url.trim()) {
    ElMessage.warning('请填写服务器地址')
    return
  }
  applyTls()
  probing.value = true
  try {
    pr.value = await probe({ url: form.url, startTLS: form.startTLS, insecureSkipVerify: form.insecureSkipVerify })
    if (!pr.value.ok) {
      ElMessage.error(pr.value.error || '探测失败')
      return
    }
    bindState.value = ''
    bindError.value = ''
    if (namingCtx.value.length) form.baseDN = namingCtx.value[0]
    step.value = 1
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    probing.value = false
  }
}

async function testBind() {
  if (!form.bindDN.trim()) {
    ElMessage.warning('请填写 Bind DN')
    return
  }
  applyTls()
  probing.value = true
  try {
    pr.value = await probe({
      url: form.url, startTLS: form.startTLS, insecureSkipVerify: form.insecureSkipVerify,
      bindDN: form.bindDN, bindPassword: form.bindPassword, baseDN: form.baseDN,
    })
    bindState.value = pr.value.bind === 'ok' ? 'ok' : 'fail'
    bindError.value = pr.value.bindError ?? ''
    if (bindState.value === 'fail') ElMessage.error(bindError.value || '验证失败')
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    probing.value = false
  }
}

async function save() {
  if (!form.baseDN.trim()) {
    ElMessage.warning('请选择或填写 Base DN')
    step.value = 1
    return
  }
  applyTls()
  saving.value = true
  try {
    await saveProfile({
      name: form.name, url: form.url, startTLS: form.startTLS,
      insecureSkipVerify: form.insecureSkipVerify, bindDN: form.bindDN,
      bindPassword: form.bindPassword, baseDN: form.baseDN, rememberPassword: form.rememberPassword,
    })
    // 保存后直接用向导里的账号登录（热生效），失败则回落到登录页
    if (form.bindDN && form.bindPassword) {
      try {
        await login(form.bindDN, form.bindPassword)
        if (pr.value?.base && !pr.value.base.exists) {
          step.value = 3
          return
        }
        goAdmin()
        return
      } catch {
        ElMessage.warning('档案已保存，但自动登录失败，请手动登录')
      }
    }
    router.push('/')
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    saving.value = false
  }
}

async function createBase() {
  bootstrapping.value = true
  try {
    const attr = rdnAttr(form.baseDN)
    const val = rdnValue(form.baseDN)
    const classes = attr === 'dc' ? ['dcObject', 'organization'] : ['organizationalUnit']
    const attrs: Record<string, string[]> = { [attr]: [val] }
    if (attr === 'dc') attrs.o = [val]
    await createEntry(form.baseDN, classes, attrs)
    // 按模板创建部门骨架（R5.5）
    const ous = tplOUs(tpl.value, form.baseDN)
    for (const [name, parent] of ous) {
      await createEntry(`ou=${name},${parent}`, ['organizationalUnit'], { ou: [name] })
    }
    ElMessage.success(`已创建基础条目与 ${ous.length} 个分支`)
    goAdmin()
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    bootstrapping.value = false
  }
}

function goAdmin() {
  router.push('/admin')
}

onMounted(async () => {
  try {
    Object.assign(status, await setupStatus())
    // 演示模式（--mock）连接固定，向导无意义 → 回登录页；
    // 已配置的真实连接则留在向导页（顶部告警），支持换绑服务器（2026-09-23 死锁修复）
    if (status.mock) {
      router.replace('/')
      return
    }
    if (status.configured) {
      // 预填当前档案，便于只改个别参数（如端口 / 加密）
      form.name = status.profile?.name || form.name
      form.url = status.profile?.url || ''
    }
  } catch (e) {
    ElMessage.error(errText(e))
  }
})
</script>

<style scoped>
.page { min-height: 100vh; display: flex; align-items: center; justify-content: center; background: linear-gradient(160deg, #f7f8fc, #eef1fa); padding: 24px; }
.wizard { width: 560px; background: #fff; border: 1px solid #e5e6eb; border-radius: 16px; padding: 32px 36px; box-shadow: 0 12px 32px rgba(31, 35, 41, 0.14); }
.head { text-align: center; margin-bottom: 22px; }
.head h1 { font-size: 19px; margin: 0 0 6px; }
.head p { font-size: 12px; color: #8f959e; margin: 0; }
.actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }
.tip { font-size: 12px; color: #8f959e; background: #f7f8fc; border-radius: 8px; padding: 10px 12px; margin-top: 8px; }
.mono { font-family: ui-monospace, Consolas, monospace; }
.muted { color: #8f959e; font-size: 12px; }
.tpl-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin: 6px 0 14px; text-align: left; }
.tpl { border: 1.5px solid var(--el-border-color); border-radius: 8px; padding: 12px; cursor: pointer; transition: 0.15s; display: flex; flex-direction: column; gap: 8px; }
.tpl:hover { border-color: #adc6ff; }
.tpl.on { border-color: var(--el-color-primary); background: #f0f5ff; }
.tpl .tt { font-size: 13px; font-weight: 600; display: flex; align-items: center; }
.tpl .mini { background: #fff; border: 1px solid #f0f1f3; border-radius: 6px; padding: 8px 10px; font-size: 10.5px; color: #646a73; line-height: 1.8; margin: 0; white-space: pre; overflow: hidden; }
.tpl p { margin: 0; font-size: 11.5px; }
</style>
