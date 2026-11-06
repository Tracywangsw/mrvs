<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, TrendCharts } from '@element-plus/icons-vue'
import { fetchDashboard, fetchMRs, fetchProducts } from '../api'
import { currentMonth, formatPercent, monthLabel } from '../utils/format'

const loading = ref(false)
const data = ref(null)
const mrs = ref([])
const products = ref([])
const filters = reactive({
  month: currentMonth(),
  product_id: null,
  mr_id: null,
})

const items = computed(() => data.value?.items || [])
const maxTotal = computed(() => {
  const values = items.value.map((item) => item.total_visits)
  return Math.max(1, ...values)
})
const abnormalRate = computed(() => data.value?.abnormal_rate ?? 0)
const circumference = 2 * Math.PI * 42

async function loadMeta() {
  try {
    const [mrData, productData] = await Promise.all([fetchMRs(), fetchProducts()])
    mrs.value = mrData.items || []
    products.value = productData.items || []
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function load() {
  loading.value = true
  try {
    data.value = await fetchDashboard({
      month: filters.month,
      product_id: filters.product_id,
      mr_id: filters.mr_id,
    })
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    loading.value = false
  }
}

function barWidth(item) {
  if (!item.total_visits) return 0
  return Math.max(3, Math.round((item.total_visits / maxTotal.value) * 100))
}

onMounted(() => {
  loadMeta()
  load()
})
</script>

<template>
  <div class="page">
    <header class="page-header">
      <div>
        <div class="page-eyebrow">Product Dashboard</div>
        <h1 class="page-title">产品月度拜访看板</h1>
        <p class="page-sub">
          按产品统计已签退 / 已提交记录的拜访次数与异常率，统计口径为签退时间所在月份。
        </p>
      </div>
      <div class="page-actions">
        <el-button :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
      </div>
    </header>

    <section class="panel panel--pad" style="margin-bottom: 18px">
      <div class="filter-bar">
        <div class="filter-item">
          <span>统计月份</span>
          <el-date-picker
            v-model="filters.month"
            type="month"
            value-format="YYYY-MM"
            format="YYYY-MM"
            :clearable="false"
            style="width: 160px"
          />
        </div>
        <div class="filter-item">
          <span>产品</span>
          <el-select v-model="filters.product_id" placeholder="全部产品" clearable filterable>
            <el-option
              v-for="item in products"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </div>
        <div class="filter-item">
          <span>医药代表</span>
          <el-select v-model="filters.mr_id" placeholder="全部代表" clearable filterable>
            <el-option
              v-for="item in mrs"
              :key="item.id"
              :label="`${item.name}（${item.employee_no}）`"
              :value="item.id"
            />
          </el-select>
        </div>
        <div class="filter-item">
          <span>&nbsp;</span>
          <el-button type="primary" :icon="TrendCharts" @click="load">查看统计</el-button>
        </div>
      </div>
    </section>

    <div v-loading="loading">
      <div class="kpi-grid">
        <div class="kpi primary">
          <div class="kpi-label">{{ monthLabel(filters.month) }} · 总拜访</div>
          <div class="kpi-number">
            {{ data?.total_visits ?? 0 }}
            <small>次</small>
          </div>
          <div class="kpi-foot">
            {{ filters.product_id ? '当前产品的月度拜访次数' : '所有产品合计' }}
          </div>
        </div>

        <div class="kpi">
          <div class="kpi-label">合规正常</div>
          <div class="kpi-number" style="color: #1c7a46">
            {{ data?.normal_visits ?? 0 }}
            <small>次</small>
          </div>
          <div class="kpi-foot muted">停留 ≥ 300 秒且偏离 &lt; 500 米</div>
        </div>

        <div class="kpi">
          <div class="kpi-label">异常拜访</div>
          <div class="kpi-number" style="color: #b23a31">
            {{ data?.abnormal_visits ?? 0 }}
            <small>次</small>
          </div>
          <div class="kpi-foot muted">时长不足或地点偏离命中</div>
        </div>

        <div class="kpi">
          <div class="kpi-label">异常率</div>
          <div class="rate-gauge" style="margin-top: 8px">
            <div class="gauge-ring">
              <svg viewBox="0 0 100 100" aria-hidden="true">
                <circle cx="50" cy="50" r="42" fill="none" stroke="#eef4f3" stroke-width="10" />
                <circle
                  cx="50"
                  cy="50"
                  r="42"
                  fill="none"
                  stroke="#b23a31"
                  stroke-width="10"
                  stroke-linecap="round"
                  :stroke-dasharray="`${abnormalRate * circumference} ${circumference}`"
                  transform="rotate(-90 50 50)"
                />
              </svg>
              <div class="gauge-value">{{ formatPercent(abnormalRate, 1) }}</div>
            </div>
            <div>
              <div style="font-size: 13px; font-weight: 600">异常占比</div>
              <div class="tiny" style="margin-top: 4px">
                异常 {{ data?.abnormal_visits ?? 0 }} / 总计 {{ data?.total_visits ?? 0 }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="dashboard-grid">
        <section class="panel panel--pad">
          <div class="panel-head">
            <h2 class="panel-title">产品拜访次数对比</h2>
            <span class="panel-hint">按总拜访次数降序</span>
          </div>

          <div v-if="items.length" class="bar-chart">
            <div v-for="item in items" :key="item.product_id" class="bar-row">
              <div class="bar-name" :title="item.product_name">
                {{ item.product_name }}
                <small>{{ item.product_code }}</small>
              </div>
              <div class="bar-track">
                <div class="bar-fill" :style="{ width: `${barWidth(item)}%` }" />
              </div>
              <div class="bar-value">
                {{ item.total_visits }} 次
                <small>异常率 {{ formatPercent(item.abnormal_rate, 1) }}</small>
              </div>
            </div>
          </div>
          <div v-else class="empty">
            <div class="empty-mark"><el-icon><TrendCharts /></el-icon></div>
            <div>当前月份暂无已签退的拜访数据</div>
          </div>
        </section>

        <section class="panel panel--pad">
          <div class="panel-head">
            <h2 class="panel-title">月度统计明细</h2>
            <span class="panel-hint">{{ monthLabel(filters.month) }}</span>
          </div>

          <div class="table-wrap">
            <el-table :data="items" empty-text="暂无统计数据">
              <el-table-column label="产品" min-width="140">
                <template #default="{ row }">
                  <div class="stack">
                    <span>{{ row.product_name }}</span>
                    <span class="tiny mono">{{ row.product_code }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="total_visits" label="总次数" width="88" align="right" />
              <el-table-column prop="normal_visits" label="正常" width="80" align="right" />
              <el-table-column prop="abnormal_visits" label="异常" width="80" align="right" />
              <el-table-column label="异常率" width="100" align="right">
                <template #default="{ row }">
                  <span :style="{ color: row.abnormal_rate > 0 ? '#b23a31' : '#64797d' }">
                    {{ formatPercent(row.abnormal_rate) }}
                  </span>
                </template>
              </el-table-column>
            </el-table>
          </div>

          <div class="dashboard-note">
            仅统计状态为「已签退」或「已提交记录」的拜访；异常率 = 异常次数 / 总次数。
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.05fr);
  gap: 18px;
  align-items: start;
}

.dashboard-note {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px dashed var(--line);
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--muted);
}

@media (max-width: 1180px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}
</style>
