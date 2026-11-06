<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Calendar, CircleCheck } from '@element-plus/icons-vue'
import { createVisit, fetchHCPs, fetchHospitals, fetchMRs, fetchProducts } from '../api'
import { toApiTime } from '../utils/format'

const router = useRouter()
const formRef = ref(null)
const submitting = ref(false)
const loadingHcps = ref(false)
const mrs = ref([])
const hospitals = ref([])
const products = ref([])
const hcps = ref([])

const form = reactive({
  mr_id: null,
  hospital_id: null,
  hcp_id: null,
  product_id: null,
  planned_at: '',
})

const rules = {
  mr_id: [{ required: true, message: '请选择医药代表', trigger: 'change' }],
  hospital_id: [{ required: true, message: '请选择医院', trigger: 'change' }],
  hcp_id: [{ required: true, message: '请选择医生', trigger: 'change' }],
  product_id: [{ required: true, message: '请选择产品', trigger: 'change' }],
  planned_at: [{ required: true, message: '请选择计划拜访时间', trigger: 'change' }],
}

const steps = [
  { title: '拜访计划', desc: '选择代表、医院、医生与目标产品' },
  { title: '现场签到', desc: '到达医院后手动输入 GPS 坐标签到' },
  { title: '学术沟通', desc: '向医生传递产品学术信息与临床数据' },
  { title: '拜访签退', desc: '签退时自动计算停留时长与位置偏离' },
  { title: '拜访记录', desc: '补充谈话要点、医生反馈与资料派发' },
]

function defaultPlannedAt() {
  const target = new Date(Date.now() + 24 * 3600 * 1000)
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(target)
  const value = {}
  parts.forEach((part) => {
    if (part.type !== 'literal') value[part.type] = part.value
  })
  return `${value.year}-${value.month}-${value.day}T10:00:00`
}

async function loadMeta() {
  try {
    const [mrData, hospitalData, productData] = await Promise.all([
      fetchMRs(),
      fetchHospitals(),
      fetchProducts(),
    ])
    mrs.value = mrData.items || []
    hospitals.value = hospitalData.items || []
    products.value = productData.items || []
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function loadHcps(hospitalId) {
  hcps.value = []
  if (!hospitalId) return
  loadingHcps.value = true
  try {
    const data = await fetchHCPs(hospitalId)
    hcps.value = data.items || []
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    loadingHcps.value = false
  }
}

watch(
  () => form.hospital_id,
  (value, previous) => {
    if (value !== previous) {
      form.hcp_id = null
      loadHcps(value)
    }
  },
)

function formatHospital(hospital) {
  return hospital.address ? `${hospital.name} · ${hospital.address}` : hospital.name
}

async function submit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const detail = await createVisit({
      mr_id: form.mr_id,
      hcp_id: form.hcp_id,
      hospital_id: form.hospital_id,
      product_id: form.product_id,
      planned_at: toApiTime(form.planned_at),
    })
    ElMessage.success(`拜访 ${detail.visit_no} 创建成功`)
    router.push(`/visits/${detail.id}`)
  } catch (error) {
    const details = error.fields
      ? Object.values(error.fields).join('；')
      : ''
    ElMessage.error(details || error.message)
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  form.planned_at = defaultPlannedAt()
  loadMeta()
})
</script>

<template>
  <div class="page">
    <header class="page-header">
      <div>
        <div class="page-eyebrow">New Visit</div>
        <h1 class="page-title">新建拜访计划</h1>
        <p class="page-sub">一次拜访对应一位医生与一个产品；医生必须属于所选医院。</p>
      </div>
      <div class="page-actions">
        <el-button :icon="ArrowLeft" @click="router.push('/visits')">返回列表</el-button>
      </div>
    </header>

    <div class="create-layout">
      <section class="panel panel--pad">
        <div class="panel-head">
          <h2 class="panel-title">拜访信息</h2>
          <span class="panel-hint">提交后状态为「已计划」</span>
        </div>

        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          label-position="top"
          @submit.prevent="submit"
        >
          <div class="form-grid">
            <el-form-item label="医药代表" prop="mr_id">
              <el-select v-model="form.mr_id" placeholder="选择医药代表" filterable style="width: 100%">
                <el-option
                  v-for="item in mrs"
                  :key="item.id"
                  :label="`${item.name}（${item.employee_no}）`"
                  :value="item.id"
                />
              </el-select>
            </el-form-item>

            <el-form-item label="产品" prop="product_id">
              <el-select v-model="form.product_id" placeholder="选择产品" filterable style="width: 100%">
                <el-option
                  v-for="item in products"
                  :key="item.id"
                  :label="item.category ? `${item.name}（${item.category}）` : item.name"
                  :value="item.id"
                />
              </el-select>
            </el-form-item>

            <el-form-item label="医院" prop="hospital_id">
              <el-select
                v-model="form.hospital_id"
                placeholder="选择医院（决定合规坐标基准）"
                filterable
                style="width: 100%"
              >
                <el-option
                  v-for="item in hospitals"
                  :key="item.id"
                  :label="formatHospital(item)"
                  :value="item.id"
                />
              </el-select>
            </el-form-item>

            <el-form-item label="医生" prop="hcp_id">
              <el-select
                v-model="form.hcp_id"
                :placeholder="form.hospital_id ? '选择医生' : '请先选择医院'"
                :disabled="!form.hospital_id"
                :loading="loadingHcps"
                filterable
                style="width: 100%"
              >
                <el-option
                  v-for="item in hcps"
                  :key="item.id"
                  :label="`${item.name} · ${item.department}${item.title ? ' · ' + item.title : ''}`"
                  :value="item.id"
                />
              </el-select>
            </el-form-item>

            <el-form-item label="计划拜访时间（Asia/Shanghai）" prop="planned_at" class="form-span-2">
              <el-date-picker
                v-model="form.planned_at"
                type="datetime"
                value-format="YYYY-MM-DDTHH:mm:ss"
                placeholder="选择计划时间"
                format="YYYY-MM-DD HH:mm"
                style="width: 100%"
              />
            </el-form-item>
          </div>

          <div class="hint-box">
            <el-icon style="font-size: 18px; margin-top: 1px"><CircleCheck /></el-icon>
            <div>
              <strong>合规提示：</strong>
              签到、签退时需手动输入 GPS 坐标。签退后系统将自动计算停留时长与距医院距离，
              停留不足 300 秒或偏离 500 米以上将被标记为异常。
            </div>
          </div>

          <div class="form-actions">
            <el-button type="primary" :loading="submitting" @click="submit">创建拜访计划</el-button>
            <el-button @click="router.push('/visits')">取消</el-button>
          </div>
        </el-form>
      </section>

      <aside class="panel panel--pad flow-panel">
        <div class="panel-head">
          <h2 class="panel-title">拜访流程</h2>
          <el-icon class="muted"><Calendar /></el-icon>
        </div>
        <ol class="flow-list">
          <li v-for="(step, index) in steps" :key="step.title" class="flow-step">
            <span class="flow-index">{{ index + 1 }}</span>
            <div>
              <div class="flow-title">{{ step.title }}</div>
              <div class="flow-desc">{{ step.desc }}</div>
            </div>
          </li>
        </ol>
        <div class="flow-note">
          创建完成后进入拜访详情页，可依次完成签到、签退与拜访记录。
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.create-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(280px, 0.85fr);
  gap: 18px;
  align-items: start;
}

.flow-panel {
  position: sticky;
  top: 24px;
}

.flow-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.flow-step {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.flow-index {
  flex: 0 0 26px;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-size: 12px;
  font-weight: 700;
  color: var(--brand-700);
  background: var(--brand-100);
}

.flow-title {
  font-size: 13.5px;
  font-weight: 650;
}

.flow-desc {
  margin-top: 3px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--muted);
}

.flow-note {
  margin-top: 20px;
  padding-top: 14px;
  border-top: 1px dashed var(--line);
  font-size: 12px;
  line-height: 1.65;
  color: var(--muted);
}

@media (max-width: 1080px) {
  .create-layout {
    grid-template-columns: 1fr;
  }

  .flow-panel {
    position: static;
  }
}
</style>
