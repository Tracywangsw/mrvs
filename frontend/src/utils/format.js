const SHANGHAI_TZ = 'Asia/Shanghai'

const dateTimeFormatter = new Intl.DateTimeFormat('en-GB', {
  timeZone: SHANGHAI_TZ,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
})

function shanghaiParts(value) {
  if (!value) return null
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return null
  const parts = dateTimeFormatter.formatToParts(date)
  const result = {}
  parts.forEach((part) => {
    if (part.type !== 'literal') result[part.type] = part.value
  })
  return result
}

/** 将 UTC ISO 时间格式化为 Asia/Shanghai 的 YYYY-MM-DD HH:mm。 */
export function formatDateTime(value) {
  const parts = shanghaiParts(value)
  if (!parts) return '—'
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}`
}

/** 将 UTC ISO 时间格式化为 Asia/Shanghai 的 YYYY-MM-DD。 */
export function formatDate(value) {
  const parts = shanghaiParts(value)
  if (!parts) return '—'
  return `${parts.year}-${parts.month}-${parts.day}`
}

/** 停留秒数转可读时长。 */
export function formatDuration(seconds) {
  if (seconds === null || seconds === undefined || seconds === '') return '—'
  const total = Number(seconds)
  if (!Number.isFinite(total) || total < 0) return '—'
  const value = Math.floor(total)
  if (value < 60) return `${value} 秒`
  const hours = Math.floor(value / 3600)
  const minutes = Math.floor((value % 3600) / 60)
  const rest = value % 60
  if (hours > 0) {
    const minutePart = rest > 0 ? `${minutes} 分 ${rest} 秒` : `${minutes} 分`
    return `${hours} 小时 ${minutePart}`
  }
  return rest > 0 ? `${minutes} 分 ${rest} 秒` : `${minutes} 分钟`
}

/** 距离（米）保留两位小数。 */
export function formatMeters(value) {
  if (value === null || value === undefined || value === '') return '—'
  const number = Number(value)
  if (!Number.isFinite(number)) return '—'
  return `${number.toFixed(2)} 米`
}

/** 比例（0-1）转百分比文本。 */
export function formatPercent(rate, digits = 2) {
  if (rate === null || rate === undefined || rate === '') return '—'
  const number = Number(rate)
  if (!Number.isFinite(number)) return '—'
  return `${(number * 100).toFixed(digits)}%`
}

export const STATUS_META = {
  planned: { label: '已计划', tone: 'planned' },
  checked_in: { label: '已签到', tone: 'checked-in' },
  checked_out: { label: '已签退', tone: 'checked-out' },
  reported: { label: '已提交记录', tone: 'reported' },
  cancelled: { label: '已取消', tone: 'cancelled' },
}

export const STATUS_OPTIONS = Object.entries(STATUS_META).map(([value, meta]) => ({
  value,
  label: meta.label,
}))

export function statusMeta(status) {
  return STATUS_META[status] || { label: status || '未知状态', tone: 'unknown' }
}

/** 合规结果文案：null 表示尚未判定。 */
export function abnormalLabel(value) {
  if (value === null || value === undefined) return '待判定'
  return value ? '异常' : '正常'
}

export function abnormalTone(value) {
  if (value === null || value === undefined) return 'pending'
  return value ? 'abnormal' : 'normal'
}

/**
 * 将日期选择器的本地值转换为后端要求的 RFC3339（Asia/Shanghai 偏移）。
 * 支持 YYYY-MM-DDTHH:mm、YYYY-MM-DDTHH:mm:ss 与 Date。
 */
export function toApiTime(value) {
  if (!value) return null
  if (value instanceof Date) {
    if (Number.isNaN(value.getTime())) return null
    const parts = shanghaiParts(value)
    return `${parts.year}-${parts.month}-${parts.day}T${parts.hour}:${parts.minute}:00+08:00`
  }
  const text = String(value).trim().replace(' ', 'T')
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(text)) return `${text}:00+08:00`
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?$/.test(text)) return `${text}+08:00`
  return text
}

/** 当前 Asia/Shanghai 月份，格式 YYYY-MM。 */
export function currentMonth() {
  const parts = shanghaiParts(new Date())
  return `${parts.year}-${parts.month}`
}

/** 月份选择器展示文案。 */
export function monthLabel(month) {
  if (!month || !/^\d{4}-\d{2}$/.test(month)) return '—'
  const [year, monthNumber] = month.split('-')
  return `${year} 年 ${Number(monthNumber)} 月`
}

export function formatCoordinate(value) {
  if (value === null || value === undefined || value === '') return '—'
  const number = Number(value)
  if (!Number.isFinite(number)) return '—'
  return number.toFixed(6)
}
