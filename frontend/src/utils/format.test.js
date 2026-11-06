import { describe, expect, it } from 'vitest'
import { buildQuery } from '../api/client'
import {
  abnormalLabel,
  currentMonth,
  formatDateTime,
  formatDuration,
  formatMeters,
  formatPercent,
  statusMeta,
  toApiTime,
} from './format'

describe('formatDateTime', () => {
  it('将 UTC 转换为 Asia/Shanghai 时间', () => {
    expect(formatDateTime('2026-11-05T01:30:00Z')).toBe('2026-11-05 09:30')
  })

  it('处理跨日边界', () => {
    expect(formatDateTime('2026-11-05T16:05:00Z')).toBe('2026-11-06 00:05')
  })

  it('空值与非法值返回占位符', () => {
    expect(formatDateTime(null)).toBe('—')
    expect(formatDateTime('not-a-date')).toBe('—')
  })
})

describe('formatDuration', () => {
  it('小于一分钟按秒展示', () => {
    expect(formatDuration(59)).toBe('59 秒')
  })

  it('整分钟与带秒的分钟', () => {
    expect(formatDuration(300)).toBe('5 分钟')
    expect(formatDuration(299)).toBe('4 分 59 秒')
  })

  it('超过一小时', () => {
    expect(formatDuration(7260)).toBe('2 小时 1 分')
  })

  it('空值返回占位符', () => {
    expect(formatDuration(null)).toBe('—')
  })
})

describe('formatMeters / formatPercent', () => {
  it('距离保留两位小数', () => {
    expect(formatMeters(812.4)).toBe('812.40 米')
    expect(formatMeters(0)).toBe('0.00 米')
    expect(formatMeters(null)).toBe('—')
  })

  it('比例转百分比', () => {
    expect(formatPercent(0.1234)).toBe('12.34%')
    expect(formatPercent(0)).toBe('0.00%')
    expect(formatPercent(null)).toBe('—')
  })
})

describe('statusMeta / abnormalLabel', () => {
  it('状态文案', () => {
    expect(statusMeta('reported').label).toBe('已提交记录')
    expect(statusMeta('planned').tone).toBe('planned')
    expect(statusMeta('unknown').label).toBe('unknown')
  })

  it('合规结果文案区分未判定', () => {
    expect(abnormalLabel(null)).toBe('待判定')
    expect(abnormalLabel(false)).toBe('正常')
    expect(abnormalLabel(true)).toBe('异常')
  })
})

describe('toApiTime', () => {
  it('补全秒与东八区偏移', () => {
    expect(toApiTime('2026-11-05T09:30')).toBe('2026-11-05T09:30:00+08:00')
    expect(toApiTime('2026-11-05T09:30:00')).toBe('2026-11-05T09:30:00+08:00')
    expect(toApiTime('2026-11-05 09:30:00')).toBe('2026-11-05T09:30:00+08:00')
  })

  it('空值返回 null', () => {
    expect(toApiTime('')).toBeNull()
  })

  it('Date 按 Asia/Shanghai 输出', () => {
    expect(toApiTime(new Date('2026-11-05T01:30:00Z'))).toBe('2026-11-05T09:30:00+08:00')
  })
})

describe('currentMonth', () => {
  it('当前月份格式为 YYYY-MM', () => {
    expect(currentMonth()).toMatch(/^\d{4}-\d{2}$/)
  })
})

describe('buildQuery', () => {
  it('过滤空值并编码', () => {
    expect(buildQuery({ mr_id: 1, status: '', page: null, abnormal: true })).toBe('?mr_id=1&abnormal=true')
    expect(buildQuery({})).toBe('')
    expect(buildQuery({ month: '2026-11' })).toBe('?month=2026-11')
  })
})
