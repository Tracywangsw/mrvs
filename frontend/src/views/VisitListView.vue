<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus, Refresh, Search } from '@element-plus/icons-vue'
import { fetchMRs, fetchProducts, fetchVisits } from '../api'
import {
  STATUS_OPTIONS,
  abnormalLabel,
  abnormalTone,
  formatDateTime,
  formatDuration,
  formatMeters,
  statusMeta,
} from '../utils/format'

const router = useRouter()
const loading = ref(false)
const mrs = ref([])
const products = ref([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const items = ref([])

const filters = reactive({
  mr_id: null,
  product_id: null,
  status: '',
  abnormal: null,
  dateRange: [],
})

const abnormalOptions = [
  { label: '仅异常', value: true },
  { label: '仅正常', value: false },
]

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
    const data = await fetchVisits({
      mr_id: filters.mr_id,
      product_id: filters.product_id,
      status: filters.status,
      abnormal: filters.abnormal,
      date_from: filters.dateRange?.[0],
      date_to: filters.dateRange?.[1],
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = data.items || []
    total.value = data.total
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  load()
}

function reset() {
  filters.mr_id = null
  filters.product_id = null
  filters.status = ''
  filters.abnormal = null
  filters.dateRange = []
  search()
}

function handleSizeChange(size) {
  pageSize.value = size
  page.value = 1
  load()
}

function goDetail(row) {
  router.push(`/visits/${row.id}`)
}

function openDetail(id) {
  router.push(`/visits/${id}`)
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
        <div class="page-eyebrow">Visit Management</div>
        <h1 class="page-title">拜访列表</h1>
        <p class="page-sub">追踪计划、签到、签退与拜访记录，异常拜访由后端自动标记。</p>
      </div>
      <div class="page-actions">
        <el-button :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
        <el-button type="primary" :icon="Plus" @click="router.push('/visits/new')">
          新建拜访
        </el-button>
      </div>
    </header>

    <section class="panel panel--pad filter-panel">
      <div class="filter-bar">
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
          <span>状态</span>
          <el-select v-model="filters.status" placeholder="全部状态" clearable>
            <el-option
              v-for="item in STATUS_OPTIONS"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
        <div class="filter-item">
          <span>合规结果</span>
          <el-select v-model="filters.abnormal" placeholder="全部结果" clearable>
            <el-option
              v-for="item in abnormalOptions"
              :key="String(item.value)"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
        <div class="filter-item">
          <span>计划日期</span>
          <el-date-picker
            v-model="filters.dateRange"
            type="daterange"
            value-format="YYYY-MM-DD"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            range-separator="至"
            style="width: 252px"
          />
        </div>
        <div class="filter-item">
          <span>&nbsp;</span>
          <div style="display: flex; gap: 8px">
            <el-button type="primary" :icon="Search" @click="search">查询</el-button>
            <el-button @click="reset">重置</el-button>
          </div>
        </div>
      </div>
    </section>

    <section class="panel panel--pad" style="margin-top: 18px">
      <div class="panel-head">
        <h2 class="panel-title">拜访记录</h2>
        <span class="panel-hint">共 {{ total }} 条，点击行可查看详情</span>
      </div>

      <div class="table-wrap">
        <el-table
          v-loading="loading"
          :data="items"
          row-class-name="clickable-row"
          empty-text="暂无拜访记录，试试调整筛选条件或新建拜访"
          @row-click="goDetail"
        >
          <el-table-column prop="visit_no" label="拜访单号" width="150">
            <template #default="{ row }">
              <span class="mono">{{ row.visit_no }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="planned_at" label="计划时间" width="150">
            <template #default="{ row }">{{ formatDateTime(row.planned_at) }}</template>
          </el-table-column>
          <el-table-column prop="mr_name" label="代表" width="100" />
          <el-table-column label="医生 / 科室" min-width="150">
            <template #default="{ row }">
              <div class="stack">
                <span>{{ row.hcp_name }}</span>
                <span class="tiny">{{ row.hcp_department }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="hospital_name" label="医院" min-width="170" show-overflow-tooltip />
          <el-table-column prop="product_name" label="产品" min-width="130" show-overflow-tooltip />
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <span class="chip" :class="statusMeta(row.status).tone">
                {{ statusMeta(row.status).label }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="签到 / 签退" width="160">
            <template #default="{ row }">
              <div class="stack">
                <span class="tiny">签到 {{ row.checkin_at ? formatDateTime(row.checkin_at) : '—' }}</span>
                <span class="tiny">签退 {{ row.checkout_at ? formatDateTime(row.checkout_at) : '—' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="停留时长" width="110">
            <template #default="{ row }">{{ formatDuration(row.duration_seconds) }}</template>
          </el-table-column>
          <el-table-column label="最大偏离" width="110">
            <template #default="{ row }">{{ formatMeters(row.deviation_meters) }}</template>
          </el-table-column>
          <el-table-column label="合规结果" width="100">
            <template #default="{ row }">
              <span class="chip" :class="abnormalTone(row.is_abnormal)">
                {{ abnormalLabel(row.is_abnormal) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click.stop="openDetail(row.id)">查看</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          background
          layout="total, sizes, prev, pager, next"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          @current-change="load"
          @size-change="handleSizeChange"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
.filter-panel {
  padding-top: 18px;
  padding-bottom: 18px;
}
</style>
