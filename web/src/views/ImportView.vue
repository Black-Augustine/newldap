<template>
  <div style="max-width: 1080px; margin: 0 auto">
    <div class="nl-page-head">
      <div>
        <h2>导入导出</h2>
        <div class="crumb">Excel 批量维护 · 后续版本：飞书 / 企业微信 / 钉钉同步</div>
      </div>
    </div>

    <!-- 导入向导 -->
    <div class="nl-card" style="margin-bottom: 14px">
      <div class="nl-card-head">
        <span class="t"><el-icon><FileSpreadsheet /></el-icon>Excel 导入</span>
        <span class="nl-pill plain">第 {{ step }} 步 / 共 4 步</span>
      </div>
      <div class="nl-card-body">
        <el-steps :active="step - 1" finish-status="success" style="margin-bottom: 22px">
          <el-step title="下载模板" />
          <el-step title="上传文件" />
          <el-step title="确认计划" />
          <el-step title="执行完成" />
        </el-steps>

        <!-- 1 模板 -->
        <div v-if="step === 1" style="display: flex; gap: 14px; flex-wrap: wrap">
          <div style="flex: 1; min-width: 300px; border: 1px solid var(--el-border-color-lighter); border-radius: 10px; padding: 16px; display: flex; gap: 14px; align-items: center">
            <el-icon :size="32" color="#2F54EB"><FileSpreadsheet /></el-icon>
            <div style="flex: 1">
              <b>组织与人员导入模板.xlsx</b>
              <div class="muted" style="margin-top: 4px">两个工作表：「部门」和「人员」· 表头均为中文，附填写说明与示例行</div>
            </div>
            <el-button @click="downloadTemplate"><el-icon><Download /></el-icon>&nbsp;下载模板</el-button>
          </div>
          <div style="flex: 1; min-width: 300px">
            <div class="nl-hint info" style="height: 100%">
              <el-icon style="margin-top: 2px"><Lightbulb /></el-icon>
              <div style="line-height: 1.9">
                <b>填写须知</b><br>
                · 部门表：按层级填「部门全路径」，如 数据智能部/算法组<br>
                · 人员表：工号是匹配唯一标识；已存在者默认<b>更新</b>其属性/部门<br>
                · 密码列不需要填：新账号初始密码自动生成，执行后一次性展示<br>
                · 示例行请替换为真实数据
              </div>
            </div>
          </div>
        </div>

        <!-- 2 上传 -->
        <div v-else-if="step === 2">
          <el-upload drag :auto-upload="false" :limit="1" :on-change="onFile" accept=".xlsx"
            :show-file-list="false" class="dropzone">
            <el-icon :size="34" color="#8F959E"><UploadCloud /></el-icon>
            <div style="font-size: 14px; color: #4e5561; margin-top: 8px">点击选择或拖入 .xlsx 文件</div>
            <div class="muted">单次建议 ≤ 5000 行 · 文件仅在导入期间使用，不会被保存</div>
          </el-upload>
          <div v-if="parsing" style="margin-top: 12px"><el-icon class="is-loading"><LoaderCircle /></el-icon> 正在解析并对账目录…</div>
          <div v-if="file" style="margin-top: 12px; display: flex; justify-content: center">
            <span class="nl-pill ok"><el-icon :size="12"><FileCheck2 /></el-icon>{{ file }}</span>
          </div>
        </div>

        <!-- 3 计划确认 -->
        <div v-else-if="step === 3 && plan">
          <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 14px; flex-wrap: wrap">
            <span class="nl-pill ok"><el-icon :size="12"><FileCheck2 /></el-icon>{{ plan.file }} 解析完成</span>
            <span class="nl-pill info">匹配键：工号（其次 uid）</span>
            <span class="nl-pill info">冲突策略：更新已有人员</span>
            <span class="nl-pill info">缺失部门自动创建</span>
          </div>
          <div class="stats">
            <div class="nl-card stat info"><span class="ico"><el-icon><Users /></el-icon></span><div><div class="n">{{ plan.summary.create }}</div><div class="l">新建人员</div></div></div>
            <div class="nl-card stat warn"><span class="ico"><el-icon><PencilLine /></el-icon></span><div><div class="n">{{ plan.summary.update }}</div><div class="l">更新人员</div></div></div>
            <div class="nl-card stat info"><span class="ico"><el-icon><FolderPlus /></el-icon></span><div><div class="n">{{ plan.summary.dept }}</div><div class="l">新建部门</div></div></div>
            <div class="nl-card stat dgr"><span class="ico"><el-icon><TriangleAlert /></el-icon></span><div><div class="n">{{ plan.summary.error }}</div><div class="l">错误行（将跳过）</div></div></div>
          </div>
          <div style="border: 1px solid var(--el-border-color-lighter); border-radius: 10px; overflow: auto; max-height: 360px">
            <el-table :data="plan.items" size="small" :row-class-name="rowClass">
              <el-table-column label="行号" width="60" prop="row" />
              <el-table-column label="类型" width="100">
                <template #default="{ row }">
                  <span class="nl-pill" :class="{ info: row.kind === 'dept', plain: row.kind !== 'dept' }">{{ row.kind === 'dept' ? '部门' : '人员' }}</span>
                </template>
              </el-table-column>
              <el-table-column label="动作" width="80">
                <template #default="{ row }">
                  <span class="nl-pill" :class="pillClass(row.action)">{{ actionCN(row.action) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="名称" prop="name" width="110" />
              <el-table-column label="键" prop="key" width="150">
                <template #default="{ row }"><span class="mono" style="font-size: 11px">{{ row.key || '—' }}</span></template>
              </el-table-column>
              <el-table-column label="详情" min-width="260">
                <template #default="{ row }"><span :style="row.action === 'error' ? 'color: var(--nl-dgr)' : 'color: #646a73'">{{ row.detail }}</span></template>
              </el-table-column>
            </el-table>
          </div>
          <div v-if="plan.summary.error > 0" class="nl-hint warn" style="margin-top: 12px">
            <el-icon style="margin-top: 2px"><TriangleAlert /></el-icon>
            <div>有 <b>{{ plan.summary.error }} 个错误行</b>将被跳过，不会写入目录。修正后可再次导入（已成功的行会按「更新」处理，不会重复创建）。</div>
          </div>
        </div>

        <!-- 4 执行 -->
        <div v-else-if="step === 4">
          <template v-if="executing">
            <div style="display: flex; justify-content: space-between; font-size: 12.5px; margin-bottom: 8px">
              <span>正在写入目录…</span><span class="mono">{{ progress }}%</span>
            </div>
            <el-progress :percentage="progress" :stroke-width="8" />
            <div class="exec-log mono" ref="logEl"></div>
          </template>
          <template v-else-if="report">
            <div class="nl-hint ok" style="margin-bottom: 16px">
              <el-icon style="margin-top: 2px"><CircleCheck /></el-icon>
              <div><b>导入完成。</b>本批次已记入审计日志（操作：import）。新账号的初始密码请立即转交本人——离开本页后无法再次查看。</div>
            </div>
            <div class="stats">
              <div class="nl-card stat ok"><span class="ico"><el-icon><CircleCheck /></el-icon></span><div><div class="n">{{ report.summary.ok }}</div><div class="l">成功</div></div></div>
              <div class="nl-card stat dgr"><span class="ico"><el-icon><CircleX /></el-icon></span><div><div class="n">{{ report.summary.fail }}</div><div class="l">失败</div></div></div>
              <div class="nl-card stat warn"><span class="ico"><el-icon><CircleSlash2 /></el-icon></span><div><div class="n">{{ report.summary.skip }}</div><div class="l">跳过（错误行）</div></div></div>
              <div class="nl-card stat info"><span class="ico"><el-icon><KeyRound /></el-icon></span><div><div class="n">{{ report.initialPasswords.length }}</div><div class="l">初始密码待转交</div></div></div>
            </div>
            <div v-if="report.initialPasswords.length" class="nl-card" style="margin-bottom: 14px">
              <div class="nl-card-head"><span class="t"><el-icon><KeyRound /></el-icon>新账号初始密码（仅此一次）</span>
                <el-button size="small" @click="copyAllPasswords"><el-icon><Copy /></el-icon>&nbsp;复制全部</el-button>
              </div>
              <div class="nl-card-body" style="padding: 8px 18px">
                <div v-for="p in report.initialPasswords" :key="p.dn" class="pw-row">
                  <span>{{ p.uid }}</span><span class="spacer" /><span class="mono">{{ p.password }}</span>
                </div>
              </div>
            </div>
            <div class="nl-hint info">
              <el-icon style="margin-top: 2px"><Lightbulb /></el-icon>
              <div>请通知新员工前往 <b>/password</b> 入口首次登录改密；初始密码不会写入审计日志。</div>
            </div>
          </template>
        </div>

        <!-- 向导导航 -->
        <div style="display: flex; justify-content: space-between; margin-top: 22px">
          <el-button :disabled="step === 1 || executing" @click="step--">
            <el-icon><ArrowLeft /></el-icon>&nbsp;上一步
          </el-button>
          <div>
            <span v-if="step === 3 && plan" class="muted" style="margin-right: 10px">共 {{ plan.summary.create + plan.summary.update + plan.summary.dept }} 项将执行</span>
            <el-button v-if="step < 3" type="primary" :disabled="step === 2 && !file" @click="next">
              下一步<el-icon style="margin-left: 4px"><ArrowRight /></el-icon>
            </el-button>
            <el-button v-if="step === 3" type="primary" :loading="executing" @click="execute">
              <el-icon><CircleCheck /></el-icon>&nbsp;开始执行
            </el-button>
            <el-button v-if="step === 4 && !executing" type="primary" @click="reset">再导一批</el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 导出 -->
    <div class="nl-card">
      <div class="nl-card-head"><span class="t"><el-icon><Download /></el-icon>导出</span></div>
      <div class="nl-card-body" style="display: flex; gap: 14px; align-items: center; flex-wrap: wrap">
        <span style="color: #4e5561">导出整个目录的部门与人员表（可直接作为下一次导入的底稿）</span>
        <span class="spacer" style="flex: 1"></span>
        <el-button type="primary" @click="doExport"><el-icon><Download /></el-icon>&nbsp;导出 Excel</el-button>
        <div class="nl-hint warn" style="flex-basis: 100%">
          <el-icon style="margin-top: 2px"><Lock /></el-icon>
          <div>导出内容<b>永不包含密码</b>及其哈希字段。</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  ArrowLeft, ArrowRight, CircleCheck, CircleSlash2, CircleX, Copy, Download, FileCheck2,
  FileSpreadsheet, FolderPlus, KeyRound, Lightbulb, LoaderCircle, Lock, PencilLine,
  TriangleAlert, UploadCloud, Users,
} from 'lucide-vue-next'
import {
  errText, executeImport, exportURL, parseImport, templateURL,
  type ImportParseResult, type ImportPlanItem,
} from '../api'
import { bumpTree } from '../store/app'

const step = ref(1)
const file = ref('')
const plan = ref<ImportParseResult | null>(null)
const parsing = ref(false)
const executing = ref(false)
const progress = ref(0)
const logEl = ref<HTMLElement>()
const report = ref<any>(null)

function downloadTemplate() { window.open(templateURL(), '_blank') }
function doExport() { window.open(exportURL(), '_blank') }

function onFile(f: any) {
  const raw = f.raw as File
  if (!raw) return
  file.value = raw.name
  parsing.value = true
  parseImport(raw)
    .then((res) => {
      plan.value = res
      step.value = 3
    })
    .catch((e) => { ElMessage.error(errText(e)); file.value = '' })
    .finally(() => { parsing.value = false })
}

function next() {
  if (step.value === 1) step.value = 2
  else if (step.value === 2 && file.value) step.value = 3
}

function rowClass({ row }: any) { return row.action === 'error' ? 'err-row' : '' }
function pillClass(action: string) {
  return action === 'create' ? 'ok' : action === 'update' ? 'warn' : action === 'dept' ? 'info' : 'dgr'
}
function actionCN(a: string) { return a === 'create' ? '新建' : a === 'update' ? '更新' : a === 'error' ? '错误' : '新建' }

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

async function execute() {
  if (!plan.value) return
  executing.value = true
  step.value = 4
  progress.value = 0
  // 进度动画（执行在后端一次性完成，这里给用户可感知的节奏）
  const timer = setInterval(() => {
    progress.value = Math.min(90, progress.value + Math.random() * 12)
  }, 220)
  try {
    const [res] = await Promise.all([
      executeImport(plan.value.items as ImportPlanItem[]),
      sleep(900),
    ])
    clearInterval(timer)
    for (let p = progress.value; p <= 100; p += 7) {
      progress.value = Math.min(100, p)
      await sleep(30)
    }
    report.value = res
    bumpTree() // 刷新侧栏树
  } catch (e) {
    clearInterval(timer)
    ElMessage.error(errText(e))
    step.value = 3
  } finally {
    executing.value = false
    nextTick(() => logEl.value?.scrollTo(0, logEl.value.scrollHeight))
  }
}

function copyAllPasswords() {
  const text = (report.value?.initialPasswords ?? []).map((p: any) => `${p.uid}\t${p.password}`).join('\n')
  navigator.clipboard?.writeText(text)
  ElMessage.success('已复制（账号 + 初始密码），请安全转交')
}

function reset() {
  step.value = 1
  file.value = ''
  plan.value = null
  report.value = null
  progress.value = 0
}
</script>

<style scoped>
.dropzone :deep(.el-upload-dragger) { padding: 38px 20px; border-radius: 12px; }
.stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 14px; }
.stat { padding: 14px; display: flex; align-items: center; gap: 12px; }
.stat .n { font-size: 20px; font-weight: 600; line-height: 1; }
.stat .l { font-size: 12px; color: #8f959e; margin-top: 4px; }
.stat .ico { width: 36px; height: 36px; border-radius: 8px; display: flex; align-items: center; justify-content: center; flex: none; }
.stat.info .ico, .stat.ok .ico { background: #f0f5ff; color: #2f54eb; }
.stat.ok .ico { background: var(--nl-ok-bg); color: var(--nl-ok); }
.stat.ok .n { color: var(--nl-ok); }
.stat.warn .ico { background: var(--nl-warn-bg); color: var(--nl-warn); }
.stat.warn .n { color: var(--nl-warn); }
.stat.dgr .ico { background: var(--nl-dgr-bg); color: var(--nl-dgr); }
.stat.dgr .n { color: var(--nl-dgr); }
.stat.info .n { color: #2f54eb; }
:deep(.err-row) { background: #fdf3f3 !important; }
.pw-row { display: flex; align-items: center; padding: 7px 0; border-bottom: 1px dashed var(--el-border-color-lighter); font-size: 13px; }
.pw-row:last-child { border-bottom: 0; }
.exec-log { margin-top: 14px; background: #fafbfc; border: 1px solid var(--el-border-color-lighter); border-radius: 8px; padding: 12px 14px; line-height: 2; color: #646a73; font-size: 11.5px; max-height: 180px; overflow: auto; min-height: 60px; }
.spacer { flex: 1; }
</style>
