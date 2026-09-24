<template>
  <div class="ss-page">
    <header class="ss-top">
      <AppLogo :size="28" />
      <span class="brand">NewLDAP</span>
      <span class="sub">自助服务 · 员工改密通道</span>
    </header>
    <main class="ss-main">
      <!-- 表单态 -->
      <div class="nl-card ss-card" v-if="!done">
        <h2><el-icon :size="20" color="#2F54EB"><KeyRound /></el-icon>修改我的密码</h2>
        <p class="sub">使用你的<b>域账号</b>（登录公司内部系统的账号）验证后即可自行修改，无需找管理员。</p>

        <el-form label-position="top" @submit.prevent>
          <el-form-item label="我的账号">
            <el-input v-model="account" class="mono" placeholder="如 zhangwei">
              <template #suffix><span class="muted mono">@{{ domainHint }}</span></template>
            </el-input>
          </el-form-item>
          <el-form-item label="当前密码">
            <el-input v-model="oldPw" type="password" show-password @keyup.enter="submit" />
          </el-form-item>
          <el-form-item label="新密码">
            <el-input v-model="newPw" type="password" show-password placeholder="至少 8 位，建议混合字母、数字与符号" />
            <div class="meter">
              <i v-for="i in 3" :key="i" :class="{ on: strength >= i, s1: strength === 1, s2: strength === 2, s3: strength === 3 }" />
            </div>
            <div class="meter-txt">密码强度：{{ strengthText }} · 至少 8 位</div>
          </el-form-item>
          <el-form-item label="确认新密码">
            <el-input v-model="newPw2" type="password" show-password @keyup.enter="submit" />
          </el-form-item>
        </el-form>

        <!-- 服务端友好错误（ppolicy 翻译 / 限流） -->
        <div v-if="error" class="nl-hint" :class="errorKind === 'rate' ? 'warn' : 'dgr'" style="margin-bottom: 14px">
          <el-icon style="margin-top: 2px"><TriangleAlert /></el-icon>
          <div>{{ error }}</div>
        </div>

        <el-button type="primary" size="large" style="width: 100%" :loading="loading" @click="submit">确认修改</el-button>

        <div class="foot muted">
          连续失败会被临时锁定（约 10 分钟）· 密码全程加密传输 · 其他已登录设备（VPN / 邮箱）可能需重新登录
        </div>
      </div>

      <!-- 成功态 -->
      <div class="nl-card ss-card" v-else>
        <div class="done">
          <div class="ok-ic"><el-icon :size="32"><CircleCheck /></el-icon></div>
          <h2 style="justify-content: center">密码修改成功</h2>
          <p class="muted" style="margin-top: 8px; line-height: 1.8">下次登录请使用新密码。<br>其他已登录的设备（VPN / 邮箱）可能需要重新输入。</p>
          <el-button style="margin-top: 20px" @click="reset"><el-icon><RotateCcw /></el-icon>&nbsp;再改一次</el-button>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { CircleCheck, KeyRound, RotateCcw, TriangleAlert } from 'lucide-vue-next'
import AppLogo from '../components/AppLogo.vue'
import { domainFromBaseDN, errText, http, setupStatus } from '../api'

const account = ref('')
const oldPw = ref('')
const newPw = ref('')
const newPw2 = ref('')
const loading = ref(false)
const error = ref('')
const errorKind = ref<'normal' | 'rate'>('normal')
const done = ref(false)
const domainHint = ref(location.hostname)

const strength = computed(() => {
  const v = newPw.value
  if (!v) return 0
  let s = 0
  if (v.length >= 8) s++
  if (/[a-zA-Z]/.test(v) && /\d/.test(v)) s++
  if (/[^a-zA-Z0-9]/.test(v) && v.length >= 10) s++
  return s
})
const strengthText = computed(() => ['', '弱', '中', '强'][strength.value] ?? '—')

async function submit() {
  error.value = ''
  if (!account.value.trim()) { error.value = '请填写你的账号（如 zhangwei）'; return }
  if (!oldPw.value || !newPw.value) { error.value = '请填写当前密码与新密码'; return }
  if (newPw.value.length < 8) { error.value = '新密码至少需要 8 位（这是服务器密码策略的最低要求）'; return }
  if (newPw.value !== newPw2.value) { error.value = '两次输入的新密码不一致，请重新输入'; return }
  if (newPw.value === oldPw.value) { error.value = '新密码不能与当前密码相同'; return }

  loading.value = true
  try {
    await http.post('/api/v1/selfservice/password', {
      account: account.value.trim(),
      oldPassword: oldPw.value,
      newPassword: newPw.value,
    })
    done.value = true
  } catch (e: any) {
    errorKind.value = e?.response?.status === 429 ? 'rate' : 'normal'
    error.value = errText(e) || '修改失败，请稍后再试'
  } finally {
    loading.value = false
  }
}

function reset() {
  done.value = false
  account.value = ''; oldPw.value = ''; newPw.value = ''; newPw2.value = ''
  error.value = ''
}

// 有配置档案时展示真实域名后缀（dc=corp,dc=test → corp.test，多级 dc 全取）
setupStatus().then((s) => {
  const dom = domainFromBaseDN(s.profile?.baseDN ?? '')
  if (dom) domainHint.value = dom
})
</script>

<style scoped>
.ss-page { height: 100vh; display: flex; flex-direction: column; background: var(--el-bg-color-page); }
.ss-top { height: 52px; background: #fff; border-bottom: 1px solid var(--el-border-color); display: flex; align-items: center; padding: 0 22px; gap: 9px; }
.ss-top .brand { font-weight: 600; font-size: 15px; }
.ss-top .sub { font-weight: 400; font-size: 11.5px; color: #8f959e; }
.ss-main { flex: 1; display: flex; align-items: center; justify-content: center; }
.ss-card { width: 420px; padding: 30px 34px 26px; }
.ss-card h2 { font-size: 18px; display: flex; align-items: center; gap: 10px; margin: 0 0 6px; }
.ss-card .sub { font-size: 12.5px; color: #8f959e; margin: 0 0 20px; line-height: 1.7; }
.meter { display: flex; gap: 5px; margin-top: 8px; width: 100%; }
.meter i { height: 4px; flex: 1; border-radius: 2px; background: #e5e6eb; transition: 0.2s; }
.meter i.s1 { background: #f1b21a }
.meter i.s2 { background: #f1b21a }
.meter i.s3 { background: var(--nl-ok) }
.meter i.on.s1 { background: #f1b21a }
.meter i.on.s2 { background: #f1b21a }
.meter i.on.s3 { background: var(--nl-ok) }
.meter-txt { font-size: 11.5px; color: #8f959e; margin-top: 6px; }
.foot { margin-top: 14px; padding-top: 14px; border-top: 1px dashed var(--el-border-color-lighter); text-align: center; font-size: 11.5px; line-height: 1.8; }
.done { display: flex; flex-direction: column; align-items: center; text-align: center; padding: 20px 0; }
.ok-ic { width: 56px; height: 56px; border-radius: 50%; background: var(--nl-ok-bg); color: var(--nl-ok); display: flex; align-items: center; justify-content: center; margin-bottom: 16px; }
</style>
