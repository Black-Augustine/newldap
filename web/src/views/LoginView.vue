<template>
  <div class="page">
    <div class="card">
      <div class="head">
        <AppLogo :size="56" />
        <h1>NewLDAP 目录管理台</h1>
        <p style="margin-top:8px">使用目录管理员账号登录<br>（<span class="term" title="bind：用某个账号的 DN 和密码向 LDAP 服务器证明身份，相当于“登录”。">bind 登录</span>，权限 = 服务端 ACL）</p>
      </div>
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="目录服务器">
          <el-input :model-value="profile?.profile?.url ?? '未配置连接'" readonly>
            <template #suffix>
              <span class="nl-pill" :class="profile?.configured ? 'ok' : 'warn'" style="margin-right: 4px">
                <span class="dot" />{{ profile?.configured ? '已配置' : '未配置' }}
              </span>
            </template>
          </el-input>
          <a class="switch-link" @click="$router.push('/setup')">{{ profile?.configured ? '连的不是这台？切换目录服务器 →' : '运行连接向导 →' }}</a>
        </el-form-item>
        <el-form-item label="Bind DN">
          <el-input v-model="bindDN" class="mono" :placeholder="`cn=admin,${baseDNHint || 'dc=…'}`" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="password" type="password" show-password @keyup.enter="submit" />
        </el-form-item>
        <el-button type="primary" size="large" style="width: 100%; margin-top: 6px" :loading="loading" @click="submit">
          登&nbsp;&nbsp;录
        </el-button>
      </el-form>
      <div class="nl-hint info" style="margin-top: 16px">
        <el-icon style="margin-top: 2px"><Info /></el-icon>
        <div>登录后你的操作权限 = 该账号在 LDAP 服务端的 ACL 权限，本工具不额外设限。</div>
      </div>
      <div class="foot">
        <template v-if="!profile?.configured">首次使用？<a @click="$router.push('/setup')" style="cursor:pointer">运行连接向导</a></template>
        <template v-else>我是普通员工，<router-link to="/password">只想修改自己的密码</router-link></template>
      </div>    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Info } from 'lucide-vue-next'
import AppLogo from '../components/AppLogo.vue'
import { errText, login, setupStatus, type SetupStatus } from '../api'

const router = useRouter()
// Bind DN 默认值跟随实际 Base DN（不同部署 dc 不同；未配置时留空由占位提示）
const bindDN = ref('')
const baseDNHint = ref('')
const password = ref('')
const loading = ref(false)
const profile = ref<SetupStatus | null>(null)

onMounted(async () => {
  try {
    profile.value = await setupStatus()
    if (!profile.value.configured) {
      router.push('/setup')
      return
    }
    const base = profile.value.profile?.baseDN ?? ''
    baseDNHint.value = base
    if (!bindDN.value && base) bindDN.value = `cn=admin,${base}`
  } catch { /* 忽略 */ }
})

async function submit() {
  loading.value = true
  try {
    await login(bindDN.value, password.value)
    router.push('/overview')
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page { height: 100vh; display: flex; align-items: center; justify-content: center; background: linear-gradient(160deg, #f7f8fc 0%, #eef1fa 100%); position: relative; overflow: hidden; }
.page::before { content: ''; position: absolute; width: 720px; height: 720px; border: 1px solid #dde3f5; border-radius: 50%; top: -260px; right: -180px; }
.page::after { content: ''; position: absolute; width: 460px; height: 460px; border: 1px solid #dde3f5; border-radius: 50%; bottom: -160px; left: -120px; }
.card { width: 400px; background: #fff; border: 1px solid var(--el-border-color); border-radius: 16px; box-shadow: 0 12px 32px rgba(31, 35, 41, 0.14); padding: 36px 36px 26px; position: relative; z-index: 1; }
.head { display: flex; flex-direction: column; align-items: center; gap: 10px; margin-bottom: 22px; text-align: center; }
.head h1 { font-size: 19px; margin: 0; }
.head p { font-size: 12.5px; color: #8f959e; margin: 0; }
.foot { margin-top: 16px; padding-top: 14px; border-top: 1px solid var(--el-border-color-lighter); text-align: center; font-size: 12px; color: #8f959e; }
.foot a { color: var(--el-color-primary); text-decoration: none; }
.switch-link { display: inline-block; margin-top: 5px; font-size: 12px; color: var(--el-color-primary); cursor: pointer; }
.switch-link:hover { text-decoration: underline; }
.term { border-bottom: 1px dashed #8f959e; cursor: help; }
</style>
