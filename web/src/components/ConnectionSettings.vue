<template>
  <div>
    <div class="nl-card" style="margin-bottom: 14px">
      <div class="nl-card-head">
        <span class="t"><el-icon><Globe /></el-icon>组织 Base DN（dc）</span>
        <span class="nl-pill plain">{{ sourceLabel }}</span>
      </div>
      <div class="nl-card-body">
        <div class="kv">
          <span class="k">当前 Base DN</span><span class="v mono">{{ profile?.profile?.baseDN ?? me?.baseDN }}</span>
          <span class="k">域名形式</span><span class="v mono">{{ domain || '—' }}</span>
        </div>
        <div class="nl-hint info" style="margin-top: 12px">
          <el-icon style="margin-top: 2px"><Lightbulb /></el-icon>
          <div>
            Base DN 是本工具管理的组织子树根，决定登录页预填、邮箱自动拼接域名等各处取值。
            修改它只改变<b>管理范围</b>，不会重命名目录服务器上的后缀；保存后所有用户访问即用新值，
            且<b>优先于部署环境变量</b>（重启不回退）。要换一台目录服务器请用「切换目录服务器」。
          </div>
        </div>
        <div style="display: flex; gap: 8px; margin-top: 14px">
          <el-button type="primary" plain @click="bdOpen = true"><el-icon><Pencil /></el-icon>&nbsp;修改 Base DN</el-button>
        </div>
      </div>
    </div>

    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 14px; margin-bottom: 14px">
      <div class="nl-card">
        <div class="nl-card-head">
          <span class="t"><el-icon><Plug /></el-icon>当前连接档案</span>
          <span class="nl-pill ok"><span class="dot" />{{ health?.ldap === 'up' ? '已连接' : '离线' }}</span>
        </div>
        <div class="nl-card-body">
          <div class="kv">
            <span class="k">档案名</span><span class="v">{{ profile?.profile?.name ?? '默认连接' }}</span>
            <span class="k">服务器</span><span class="v mono">{{ profile?.profile?.url }}</span>
            <span class="k">Base DN</span><span class="v mono">{{ profile?.profile?.baseDN ?? me?.baseDN }}</span>
            <span class="k">Bind DN</span><span class="v mono">{{ me?.bindDN }}</span>
            <span class="k">凭据存储</span><span class="v">{{ profile?.secretKeySet ? 'AES-GCM 加密（NEWLDAP_SECRET_KEY）' : '未设密钥（明文/演示）' }}</span>
          </div>
          <div style="display: flex; gap: 8px; margin-top: 16px">
            <el-button @click="testConn" :loading="testing"><el-icon><SearchCheck /></el-icon>&nbsp;测试连接</el-button>
            <el-button @click="$router.push('/setup')"><el-icon><ArrowRightLeft /></el-icon>&nbsp;切换目录服务器</el-button>
          </div>
        </div>
      </div>

      <div class="nl-card">
        <div class="nl-card-head"><span class="t"><el-icon><ScanSearch /></el-icon>目录状态（自动探测）</span></div>
        <div class="nl-card-body">
          <div class="probe-row"><el-icon class="ok"><BadgeCheck /></el-icon><span class="k">健康检查</span><span class="v">{{ health?.ldap === 'up' ? '正常' : '异常：' + (health?.detail ?? '') }}</span></div>
          <div class="probe-row"><el-icon class="ok"><BadgeCheck /></el-icon><span class="k">运行模式</span><span class="v">{{ health?.mock ? '内置演示目录（--mock）' : '外部目录' }}</span></div>
          <div class="probe-row"><el-icon class="ok"><BadgeCheck /></el-icon><span class="k">编辑保护</span><span class="v">entryCSN 冲突检测（与 LAM 并存）</span></div>
          <div class="probe-row"><el-icon class="ok"><BadgeCheck /></el-icon><span class="k">Schema</span><span class="v">动态表单已启用（服务器加载什么就能管理什么）</span></div>
        </div>
      </div>
    </div>

    <div class="nl-card">
      <div class="nl-card-head"><span class="t"><el-icon><Shield /></el-icon>安全</span></div>
      <div class="nl-card-body" style="display: flex; flex-direction: column; gap: 10px">
        <div class="nl-hint info">
          <el-icon style="margin-top: 2px"><Lock /></el-icon>
          <div>bind 密码{{ profile?.secretKeySet ? '已加密保存在本机配置中' : '当前为明文/演示保存——生产环境请设置 NEWLDAP_SECRET_KEY 环境变量后重新保存档案' }}；会话 30 分钟无操作自动登出。</div>
        </div>
        <div class="nl-hint info">
          <el-icon style="margin-top: 2px"><ClipboardList /></el-icon>
          <div>全部写操作（含批量导入、策略修改）均写入审计日志，可在「审计日志」页检索。</div>
        </div>
      </div>
    </div>

            <!-- 修改 Base DN 对话框 -->
    <el-dialog v-model="bdOpen" title="修改组织 Base DN" width="520px">
      <el-form label-position="top">
        <el-form-item label="新的 Base DN" required>
          <el-input v-model="bdNew" class="mono" placeholder="dc=corp,dc=com 或 ou=org,dc=…" />
          <div class="muted" style="font-size: 11.5px; margin-top: 4px">
            当前：{{ profile?.profile?.baseDN ?? me?.baseDN }}。新值必须是这台目录服务器上已存在的条目；
            保存前会实际读取验证。
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="bdOpen = false">取消</el-button>
        <el-button type="primary" :loading="bdSaving" @click="saveBaseDN">验证并保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowRightLeft, BadgeCheck, ClipboardList, Globe, Lightbulb, Lock, Pencil, Plug, ScanSearch, SearchCheck, Shield } from 'lucide-vue-next'
import { changeBaseDN, domainFromBaseDN, errText, healthz, me as apiMe, setupStatus, type Me, type SetupStatus } from '../api'
import { baseDomain } from '../store/app'

const me = ref<Me | null>(null)
const profile = ref<SetupStatus | null>(null)
const health = ref<any>(null)
const testing = ref(false)

const domain = computed(() => domainFromBaseDN(profile.value?.profile?.baseDN ?? me.value?.baseDN ?? ''))
const sourceLabel = computed(() => {
  const s = profile.value?.source
  if (s === 'env') return '环境变量部署'
  if (s === 'file') return '网页保存（优先于环境变量）'
  if (s === 'mock') return '演示模式'
  return '默认'
})

const bdOpen = ref(false)
const bdNew = ref('')
const bdSaving = ref(false)
async function saveBaseDN() {
  const v = bdNew.value.trim().replace(/,+$/, '')
  if (!v || !v.includes('=')) {
    ElMessage.warning('请输入合法 DN，如 dc=corp,dc=com')
    return
  }
  bdSaving.value = true
  try {
    await changeBaseDN(v)
    ElMessage.success(`Base DN 已修改为 ${v}，对所有用户生效（重启后保持）`)
    bdOpen.value = false
    bdNew.value = ''
    profile.value = await setupStatus()
    baseDomain.value = domainFromBaseDN(profile.value?.profile?.baseDN ?? '')
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    bdSaving.value = false
  }
}


async function testConn() {
  testing.value = true
  try {
    health.value = await healthz()
    ElMessage.success(health.value.ldap === 'up' ? '连接正常' : '目录不可达：' + (health.value.detail ?? ''))
  } catch (e) { ElMessage.error(errText(e)) } finally { testing.value = false }
}

onMounted(async () => {
  me.value = await apiMe()
  profile.value = await setupStatus()
  health.value = await healthz()
})
</script>

<style scoped>
.kv { display: grid; grid-template-columns: auto 1fr; gap: 8px 18px; font-size: 13px; }
.kv .k { color: #8f959e; white-space: nowrap; }
.kv .v { text-align: right; font-size: 12.5px; }
.probe-row { display: flex; align-items: center; gap: 10px; padding: 9px 0; border-bottom: 1px solid var(--el-border-color-lighter); font-size: 13px; }
.probe-row:last-child { border-bottom: 0; }
.probe-row .ok { color: var(--nl-ok); }
.probe-row .k { color: #8f959e; width: 80px; flex: none; }
.probe-row .v { flex: 1; text-align: right; font-size: 12.5px; }
</style>
