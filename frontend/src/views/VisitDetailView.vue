<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Location, Refresh } from '@element-plus/icons-vue'
import {
  cancelVisit,
  checkInVisit,
  checkOutVisit,
  fetchVisit,
  submitVisitReport,
} from '../api'
import { formatCoordinate, formatDateTime, formatDuration, statusMeta } from '../utils/format'

const route = useRoute()
const router = useRouter()

const visit = ref(null)
const loading = ref(false)
const submittingCheck = ref(false)
const submittingReport = ref(false)
const cancelling = ref(false)

const checkForm = reactive({
  latitude: null,
  longitude: null,
})

const reportForm = reactive({
  talking_points: '',
  hcp_feedback: '',
  materials_distributed: false,
  follow_up: '',
  materials: [],
})

const materialTypes = [
  { value: 'literature', label: '文献' },
  { value: 'guideline', label: '指南' },
  { value: 'slide', label: '学术幻灯片' },
  { value: 'other', label: '其他' },
]

const canReport = computed(() =>
  visit.value ? ['checked_out', 'reported'].includes(visit.value.status) : false,
)

const isReported = computed(() => visit.value?.status === 'reported')

const timeline = computed(() => {
  const detail = visit.value
  if (!detail) return []
  const items = [
    {
      title: '创建拜访计划',
      time: formatDateTime(detail.created_at),
      desc: `计划拜访时间 ${formatDateTime(detail.planned_at)}`,
      tone: '',
    },
  ]
  if (detail.checkin) {
    items.push({
      title: '现场签到',
      time: formatDateTime(detail.checkin.at),
      desc: `坐标 ${formatCoordinate(detail.checkin.latitude)}, ${formatCoordinate(detail.checkin.longitude)}`,
      tone: '',
    })
  }
  if (detail.checkout) {
    items.push({
      title: '拜访签退',
      time: formatDateTime(detail.checkout.at),
      desc: `坐标 ${formatCoordinate(detail.checkout.latitude)}, ${formatCoordinate(detail.checkout.longitude)} · 停留 ${formatDuration(detail.duration_seconds)}`,
      tone: '',
    })
  }
  if (detail.report?.report_submitted_at) {
    items.push({
      title: '提交拜访记录',
      time: formatDateTime(detail.report.report_submitted_at),
      desc: detail.report.talking_points || '已提交谈话要点',
      tone: '',
    })
  }
  if (detail.cancelled_at) {
    items.push({
      title: '取消拜访',
      time: formatDateTime(detail.cancelled_at),
      desc: detail.cancel_reason || '未填写取消原因',
      tone: 'danger-item',
    })
  }
  return items
})

function resetCheckForm() {
  checkForm.latitude = null
  checkForm.longitude = null
}

function syncReportForm(detail) {
  reportForm.talking_points = detail.report?.talking_points || ''
  reportForm.hcp_feedback = detail.report?.hcp_feedback || ''
  reportForm.materials_distributed = detail.report?.materials_distributed ?? false
  reportForm.follow_up = detail.report?.follow_up || ''
  reportForm.materials = (detail.materials || []).map((item) => ({
    material_name: item.material_name,
    material_type: item.material_type || 'literature',
    quantity: item.quantity,
  }))
}

async function load() {
  loading.value = true
  try {
    const detail = await fetchVisit(route.params.id)
    visit.value = detail
    syncReportForm(detail)
    resetCheckForm()
  } catch (error) {
    ElMessage.error(error.message)
    if (error.status === 404) router.push('/visits')
  } finally {
    loading.value = false
  }
}

function fillHospitalCoords() {
  if (!visit.value) return
  checkForm.latitude = Number(visit.value.hospital.latitude)
  checkForm.longitude = Number(visit.value.hospital.longitude)
}

function validateCoordinates() {
  const latitude = Number(checkForm.latitude)
  const longitude = Number(checkForm.longitude)
  if (checkForm.latitude === null || checkForm.longitude === null) {
    ElMessage.warning('请手动输入签到/签退坐标')
    return null
  }
  if (!Number.isFinite(latitude) || latitude < -90 || latitude > 90) {
    ElMessage.warning('纬度必须在 -90 至 90 之间')
    return null
  }
  if (!Number.isFinite(longitude) || longitude < -180 || longitude > 180) {
    ElMessage.warning('经度必须在 -180 至 180 之间')
    return null
  }
  return { latitude, longitude }
}

async function handleCheckIn() {
  const payload = validateCoordinates()
  if (!payload) return
  submittingCheck.value = true
  try {
    const detail = await checkInVisit(visit.value.id, payload)
    visit.value = detail
    syncReportForm(detail)
    resetCheckForm()
    ElMessage.success('签到成功，请完成沟通后进行签退')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    submittingCheck.value = false
  }
}

async function handleCheckOut() {
  const payload = validateCoordinates()
  if (!payload) return
  submittingCheck.value = true
  try {
    const detail = await checkOutVisit(visit.value.id, payload)
    visit.value = detail
    syncReportForm(detail)
    resetCheckForm()
    ElMessage.success('签退完成')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    submittingCheck.value = false
  }
}

function addMaterial() {
  reportForm.materials.push({ material_name: '', material_type: 'literature', quantity: 1 })
}

function removeMaterial(index) {
  reportForm.materials.splice(index, 1)
}

watch(
  () => reportForm.materials_distributed,
  (value) => {
    if (!value) reportForm.materials = []
  },
)

async function handleSubmitReport() {
  const talkingPoints = reportForm.talking_points.trim()
  if (!talkingPoints) {
    ElMessage.warning('请填写谈话要点')
    return
  }
  if (talkingPoints.length > 5000) {
    ElMessage.warning('谈话要点不能超过 5000 个字符')
    return
  }
  const materials = reportForm.materials
    .map((item) => ({
      material_name: (item.material_name || '').trim(),
      material_type: item.material_type || null,
      quantity: Number(item.quantity) || 1,
    }))
    .filter((item) => item.material_name)

  if (reportForm.materials_distributed && materials.length === 0) {
    ElMessage.warning('已派发资料时至少填写一条资料明细')
    return
  }
  if (!reportForm.materials_distributed && reportForm.materials.length > 0) {
    ElMessage.warning('未派发资料时请移除资料明细')
    return
  }

  submittingReport.value = true
  try {
    const detail = await submitVisitReport(visit.value.id, {
      talking_points: talkingPoints,
      hcp_feedback: reportForm.hcp_feedback.trim() || null,
      materials_distributed: reportForm.materials_distributed,
      follow_up: reportForm.follow_up.trim() || null,
      materials: reportForm.materials_distributed ? materials : [],
    })
    visit.value = detail
    syncReportForm(detail)
    ElMessage.success('拜访记录已提交')
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    submittingReport.value = false
  }
}

async function handleCancel() {
  try {
    const { value } = await ElMessageBox.prompt('请输入取消原因（可选）', '取消拜访', {
      confirmButtonText: '确认取消',
      cancelButtonText: '返回',
      inputPlaceholder: '例如：医生临时外出、计划调整',
      inputType: 'textarea',
    })
    cancelling.value = true
    await cancelVisit(visit.value.id, value || '')
    ElMessage.success('拜访已取消')
    await load()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error?.message || '取消失败')
    }
  } finally {
    cancelling.value = false
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="page">
    <template v-if="visit">
      <section class="panel detail-hero">
        <div>
          <div class="hero-no">
            <span class="mono" style="font-size: 22px">{{ visit.visit_no }}</span>
            <span class="chip" :class="statusMeta(visit.status).tone">
              {{ statusMeta(visit.status).label }}
            </span>
          </div>
          <div class="hero-meta">
            <span>{{ visit.mr.name }}（{{ visit.mr.employee_no }}）</span>
            <span>{{ visit.hospital.name }}</span>
            <span>计划时间 {{ formatDateTime(visit.planned_at) }}</span>
          </div>
        </div>
        <div class="page-actions">
          <el-button :icon="ArrowLeft" @click="router.push('/visits')">返回列表</el-button>
          <el-button :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
        </div>
      </section>

      <section class="panel panel--pad" style="margin-bottom: 18px">
        <div class="info-grid">
          <div class="info-item">
            <div class="info-label">医药代表</div>
            <div class="info-value">
              {{ visit.mr.name }}
              <small>{{ visit.mr.employee_no }}</small>
            </div>
          </div>
          <div class="info-item">
            <div class="info-label">医生</div>
            <div class="info-value">
              {{ visit.hcp.name }}
              <small>{{ visit.hcp.department }}{{ visit.hcp.title ? ' · ' + visit.hcp.title : '' }}</small>
            </div>
          </div>
          <div class="info-item">
            <div class="info-label">医院</div>
            <div class="info-value">
              {{ visit.hospital.name }}
              <small>{{ visit.hospital.address || '未填写地址' }}</small>
            </div>
          </div>
          <div class="info-item">
            <div class="info-label">产品</div>
            <div class="info-value">
              {{ visit.product.name }}
              <small>{{ visit.product.code }}{{ visit.product.category ? ' · ' + visit.product.category : '' }}</small>
            </div>
          </div>
          <div class="info-item">
            <div class="info-label">计划拜访时间</div>
            <div class="info-value">{{ formatDateTime(visit.planned_at) }}</div>
          </div>
          <div class="info-item">
            <div class="info-label">医院坐标</div>
            <div class="info-value mono" style="font-size: 12.5px">
              {{ formatCoordinate(visit.hospital.latitude) }},
              {{ formatCoordinate(visit.hospital.longitude) }}
            </div>
          </div>
          <div class="info-item">
            <div class="info-label">创建时间</div>
            <div class="info-value">{{ formatDateTime(visit.created_at) }}</div>
          </div>
          <div class="info-item">
            <div class="info-label">取消时间</div>
            <div class="info-value">
              {{ visit.cancelled_at ? formatDateTime(visit.cancelled_at) : '—' }}
            </div>
          </div>
        </div>
      </section>

      <div class="detail-grid">
        <div class="stack-col">
          <section class="panel panel--pad">
            <div class="panel-head">
              <h2 class="panel-title">{{ isReported ? '拜访记录（已提交）' : '拜访记录' }}</h2>
              <span v-if="visit.report?.report_submitted_at" class="panel-hint">
                {{
                  isReported
                    ? '已提交，内容不可修改'
                    : `首次提交 ${formatDateTime(visit.report.report_submitted_at)}`
                }}
              </span>
            </div>

            <template v-if="canReport">
              <el-form label-position="top" class="report-form">
                <el-form-item label="谈话要点" required>
                  <el-input
                    v-model="reportForm.talking_points"
                    type="textarea"
                    :rows="4"
                    maxlength="5000"
                    show-word-limit
                    :readonly="isReported"
                    placeholder="记录与医生沟通的产品学术信息、临床数据要点"
                  />
                </el-form-item>

                <div class="form-grid">
                  <el-form-item label="医生反馈">
                    <el-input
                      v-model="reportForm.hcp_feedback"
                      type="textarea"
                      :rows="3"
                      maxlength="5000"
                      show-word-limit
                      :readonly="isReported"
                      placeholder="医生对产品的认可、疑虑或临床反馈"
                    />
                  </el-form-item>
                  <el-form-item label="后续跟进计划">
                    <el-input
                      v-model="reportForm.follow_up"
                      type="textarea"
                      :rows="3"
                      maxlength="5000"
                      show-word-limit
                      :readonly="isReported"
                      placeholder="下一步跟进安排"
                    />
                  </el-form-item>
                </div>

                <el-form-item label="是否派发学术资料">
                  <el-switch
                    v-model="reportForm.materials_distributed"
                    active-text="已派发"
                    inactive-text="未派发"
                    :disabled="isReported"
                  />
                </el-form-item>

                <template v-if="reportForm.materials_distributed">
                  <div v-if="reportForm.materials.length" class="material-list">
                    <div class="material-head">
                      <span>资料名称</span>
                      <span>类型</span>
                      <span>数量</span>
                      <span></span>
                    </div>
                    <div
                      v-for="(material, index) in reportForm.materials"
                      :key="index"
                      class="material-row"
                    >
                      <el-input
                        v-model="material.material_name"
                        maxlength="128"
                        :readonly="isReported"
                        placeholder="如：产品临床数据手册"
                      />
                      <el-select v-model="material.material_type" :disabled="isReported">
                        <el-option
                          v-for="type in materialTypes"
                          :key="type.value"
                          :label="type.label"
                          :value="type.value"
                        />
                      </el-select>
                      <el-input-number
                        v-model="material.quantity"
                        :min="1"
                        :max="1000"
                        :controls="false"
                        :disabled="isReported"
                      />
                      <el-button
                        v-if="!isReported"
                        link
                        type="danger"
                        @click="removeMaterial(index)"
                      >
                        移除
                      </el-button>
                    </div>
                  </div>
                  <div v-else class="material-empty">尚未添加资料明细</div>
                  <el-button
                    size="small"
                    plain
                    :disabled="isReported"
                    @click="addMaterial"
                  >
                    + 添加资料
                  </el-button>
                </template>

                <div class="form-actions">
                  <el-button
                    type="primary"
                    :loading="submittingReport"
                    :disabled="isReported"
                    @click="handleSubmitReport"
                  >
                    提交拜访记录
                  </el-button>
                </div>
              </el-form>
            </template>

            <template v-else>
              <div class="empty">
                <div class="empty-mark">
                  <el-icon><Location /></el-icon>
                </div>
                <div>完成签退后可提交谈话要点、医生反馈与资料派发明细</div>
              </div>
            </template>
          </section>
        </div>

        <div class="stack-col">
          <section class="panel panel--pad action-panel">
            <div class="panel-head">
              <h2 class="panel-title">下一步操作</h2>
            </div>

            <template v-if="visit.status === 'planned'">
              <div class="action-heading">现场签到</div>
              <p class="action-sub">手动输入签到 GPS 坐标，确认已到达拜访地点。</p>
              <div class="coord-grid">
                <el-input-number
                  v-model="checkForm.latitude"
                  :precision="6"
                  :controls="false"
                  placeholder="纬度"
                />
                <el-input-number
                  v-model="checkForm.longitude"
                  :precision="6"
                  :controls="false"
                  placeholder="经度"
                />
              </div>
              <div class="coord-actions">
                <el-button size="small" plain @click="fillHospitalCoords">填入医院坐标</el-button>
              </div>
              <el-button
                type="primary"
                :loading="submittingCheck"
                style="width: 100%"
                @click="handleCheckIn"
              >
                确认签到
              </el-button>
              <el-button
                link
                type="danger"
                :loading="cancelling"
                style="margin-top: 10px"
                @click="handleCancel"
              >
                取消该拜访
              </el-button>
            </template>

            <template v-else-if="visit.status === 'checked_in'">
              <div class="action-heading">拜访签退</div>
              <p class="action-sub">手动输入签退 GPS 坐标，确认本次拜访结束。</p>
              <div class="checkin-summary">
                <div class="summary-row">
                  <span>签到时间</span>
                  <strong>{{ formatDateTime(visit.checkin.at) }}</strong>
                </div>
                <div class="summary-row">
                  <span>签到坐标</span>
                  <strong class="mono">
                    {{ formatCoordinate(visit.checkin.latitude) }},
                    {{ formatCoordinate(visit.checkin.longitude) }}
                  </strong>
                </div>
              </div>
              <div class="coord-grid">
                <el-input-number
                  v-model="checkForm.latitude"
                  :precision="6"
                  :controls="false"
                  placeholder="纬度"
                />
                <el-input-number
                  v-model="checkForm.longitude"
                  :precision="6"
                  :controls="false"
                  placeholder="经度"
                />
              </div>
              <div class="coord-actions">
                <el-button size="small" plain @click="fillHospitalCoords">填入医院坐标</el-button>
              </div>
              <el-button
                type="primary"
                :loading="submittingCheck"
                style="width: 100%"
                @click="handleCheckOut"
              >
                确认签退
              </el-button>
            </template>

            <template v-else-if="visit.status === 'checked_out'">
              <div class="action-heading">等待提交拜访记录</div>
              <p class="action-sub">
                已完成签退，请在左侧填写谈话要点与医生反馈，完成本次拜访闭环。
              </p>
            </template>

            <template v-else-if="visit.status === 'reported'">
              <div class="action-heading">拜访已完成</div>
              <p class="action-sub">记录已提交，内容不可修改。</p>
              <div class="checkin-summary">
                <div class="summary-row">
                  <span>提交时间</span>
                  <strong>{{ formatDateTime(visit.report.report_submitted_at) }}</strong>
                </div>
              </div>
            </template>

            <template v-else-if="visit.status === 'cancelled'">
              <div class="action-heading">该拜访已取消</div>
              <p class="action-sub">
                取消时间 {{ formatDateTime(visit.cancelled_at) }}
              </p>
              <div class="checkin-summary">
                <div class="summary-row">
                  <span>取消原因</span>
                  <strong>{{ visit.cancel_reason || '未填写' }}</strong>
                </div>
              </div>
            </template>
          </section>

          <section class="panel panel--pad">
            <div class="panel-head">
              <h2 class="panel-title">拜访时间线</h2>
            </div>
            <ol class="timeline">
              <li
                v-for="(item, index) in timeline"
                :key="index"
                class="timeline-item"
                :class="item.tone"
              >
                <span class="timeline-dot">{{ index + 1 }}</span>
                <div class="timeline-title">{{ item.title }}</div>
                <div class="timeline-time">{{ item.time }}</div>
                <div class="timeline-desc">{{ item.desc }}</div>
              </li>
            </ol>
          </section>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.action-panel :deep(.el-input-number) {
  width: 100%;
}

.action-heading {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 4px;
}

.action-sub {
  margin: 0 0 14px;
  font-size: 12.5px;
  line-height: 1.65;
  color: var(--muted);
}

.coord-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.coord-actions {
  display: flex;
  gap: 8px;
  margin-top: 10px;
  flex-wrap: wrap;
}

.checkin-summary {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  margin-bottom: 14px;
  border-radius: 10px;
  background: #f6faf9;
  border: 1px solid #e7f0ee;
}

.summary-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 12.5px;
  color: var(--muted);
}

.summary-row strong {
  color: var(--ink);
  font-weight: 600;
  text-align: right;
}

.material-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 10px;
}

.material-head,
.material-row {
  display: grid;
  grid-template-columns: minmax(0, 2fr) 130px 90px 56px;
  gap: 10px;
  align-items: center;
}

.material-head {
  font-size: 11px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--muted);
}

.material-empty {
  padding: 14px;
  margin-bottom: 10px;
  border: 1px dashed var(--line);
  border-radius: 10px;
  text-align: center;
  font-size: 12.5px;
  color: var(--muted);
}

.material-row :deep(.el-input-number) {
  width: 100%;
}

.report-form :deep(.el-textarea__inner[readonly]) {
  background: #f5f7f7;
  color: var(--ink);
}
</style>
