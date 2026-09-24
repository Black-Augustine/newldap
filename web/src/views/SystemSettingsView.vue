<template>
  <div style="max-width: 1100px; margin: 0 auto">
    <div class="nl-page-head">
      <div>
        <h2>系统设置</h2>
        <div class="crumb">录入与安全策略 · 目录连接 —— 策略保存在本机配置文件，目录数据不落地</div>
      </div>
    </div>

    <el-tabs v-model="tab" class="sys-tabs">
      <el-tab-pane name="policy">
        <template #label><span class="tab-l"><el-icon><ShieldCheck /></el-icon>安全与录入策略</span></template>
        <div class="nl-card">
          <div class="nl-card-head">
            <span class="t"><el-icon><KeyRound /></el-icon>账号与密码</span>
            <span class="nl-pill plain">创建人员 / 重置密码时生效</span>
          </div>
          <div class="nl-card-body" style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px 32px">
            <div class="f-item">
              <div class="f-label">账号 (uid) 最小长度</div>
              <el-input-number v-model="form.uidMinLen" :min="1" :max="32" />
              <div class="f-hint">新建人员时账号短于该值将被拒绝</div>
            </div>
            <div class="f-item">
              <div class="f-label">账号 (uid) 最大长度</div>
              <el-input-number v-model="form.uidMaxLen" :min="2" :max="64" />
              <div class="f-hint">输入框同时会限制可输入长度</div>
            </div>
            <div class="f-item">
              <div class="f-label">密码最小长度</div>
              <el-input-number v-model="form.passwordMinLen" :min="4" :max="64" />
              <div class="f-hint">管理员重置密码 / 建人指定初始密码时校验（服务器 ppolicy 策略仍独立生效）</div>
            </div>
            <div class="f-item">
              <div class="f-label">密码复杂度</div>
              <el-switch v-model="form.passwordComplexity" active-text="要求字母 + 数字" inactive-text="不强制" />
            </div>
          </div>
        </div>

        <div class="nl-card" style="margin-top: 14px">
          <div class="nl-card-head">
            <span class="t"><el-icon><AtSign /></el-icon>邮箱自动拼接</span>
            <el-switch v-model="form.emailAutofill" />
          </div>
          <div class="nl-card-body">
            <div class="nl-hint info" style="margin-bottom: 10px">
              <el-icon style="margin-top: 2px"><Lightbulb /></el-icon>
              <div>
                开启后：新建 / 编辑人员时，若<b>只填了工号、邮箱留空</b>，将自动把邮箱填为
                <span class="mono">工号@{{ baseDomain || '域名' }}</span>（域名取自 Base DN 的 dc 部分）。
                手动输入过邮箱则不做覆盖；关闭开关立即停用。
              </div>
            </div>
            <div class="f-demo mono" v-if="baseDomain">示例：工号 E1007 → E1007@{{ baseDomain }}</div>
          </div>
        </div>

        <div style="display: flex; justify-content: flex-end; margin-top: 16px">
          <el-button type="primary" :loading="saving" @click="save"><el-icon><Save /></el-icon>&nbsp;保存设置</el-button>
        </div>
      </el-tab-pane>

      <el-tab-pane name="connection">
        <template #label><span class="tab-l"><el-icon><Plug /></el-icon>目录连接</span></template>
        <ConnectionSettings />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { AtSign, KeyRound, Lightbulb, Plug, Save, ShieldCheck } from 'lucide-vue-next'
import { errText, getPolicy, savePolicy, type Policy } from '../api'
import { baseDomain, setPolicy } from '../store/app'
import ConnectionSettings from '../components/ConnectionSettings.vue'

const tab = ref('policy')
const form = reactive<Policy>({
  passwordMinLen: 8, passwordComplexity: true,
  uidMinLen: 2, uidMaxLen: 32, emailAutofill: false,
})
const saving = ref(false)

async function save() {
  if (form.uidMinLen > form.uidMaxLen) {
    ElMessage.warning('账号最小长度不能大于最大长度')
    return
  }
  saving.value = true
  try {
    const saved = await savePolicy({ ...form })
    Object.assign(form, saved)
    setPolicy(saved)
    ElMessage.success('设置已保存并即时生效（写入本机配置文件，重启后保持）')
  } catch (e) {
    ElMessage.error(errText(e))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    Object.assign(form, await getPolicy())
  } catch (e) {
    ElMessage.error(errText(e))
  }
})
</script>

<style scoped>
.sys-tabs :deep(.el-tabs__item) { font-size: 13.5px; }
.tab-l { display: inline-flex; align-items: center; gap: 6px; }
.f-item { display: flex; flex-direction: column; gap: 6px; }
.f-label { font-size: 13px; color: #1f2329; }
.f-hint { font-size: 11.5px; color: #8f959e; line-height: 1.5; }
.f-demo { font-size: 12px; color: var(--el-color-primary); background: #f0f5ff; border-radius: 6px; padding: 6px 10px; width: fit-content; }
</style>
