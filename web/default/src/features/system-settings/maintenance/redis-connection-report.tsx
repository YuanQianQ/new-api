/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Database, RefreshCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { formatNumber, formatTimestampToDate } from '@/lib/format'

import { getRedisConnectionReport } from '../api'

type MetricProps = {
  label: string
  value: string
}

function Metric({ label, value }: MetricProps) {
  return (
    <div className='min-w-0'>
      <dt className='text-muted-foreground text-xs'>{label}</dt>
      <dd className='mt-1 truncate text-sm font-medium' title={value}>
        {value}
      </dd>
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 Bytes'
  const units = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const unitIndex = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1
  )
  const value = bytes / Math.pow(1024, unitIndex)
  return `${Number.parseFloat(value.toFixed(2))} ${units[unitIndex]}`
}

export function RedisConnectionReport() {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['system-info', 'redis-connection-report'],
    queryFn: getRedisConnectionReport,
    staleTime: 15_000,
    retry: false,
  })
  const report = query.data?.data

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86_400)
    const hours = Math.floor((seconds % 86_400) / 3_600)
    const minutes = Math.floor((seconds % 3_600) / 60)
    const parts: string[] = []
    if (days) parts.push(`${formatNumber(days)} ${t('days')}`)
    if (hours) parts.push(`${hours} ${t('hours')}`)
    if (minutes || parts.length === 0) {
      parts.push(`${minutes || seconds} ${t(minutes ? 'minutes' : 'seconds')}`)
    }
    return parts.join(' ')
  }

  const status = query.isLoading
    ? { label: t('Loading...'), variant: 'neutral' as const }
    : !report
      ? { label: t('Unknown'), variant: 'neutral' as const }
      : !report.enabled
        ? { label: t('Disabled'), variant: 'neutral' as const }
        : report.connected
          ? { label: t('Connected'), variant: 'success' as const }
          : { label: t('Connection failed'), variant: 'danger' as const }

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex min-h-14 items-center justify-between gap-3 border-b px-4 py-3'>
        <div className='flex min-w-0 items-center gap-2'>
          <Database className='text-muted-foreground size-4 shrink-0' />
          <h3 className='truncate text-sm font-semibold'>
            {t('Redis connection report')}
          </h3>
          <StatusBadge
            label={status.label}
            variant={status.variant}
            copyable={false}
          />
        </div>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          onClick={() => query.refetch()}
          disabled={query.isFetching}
          aria-label={t('Refresh Redis connection report')}
          title={t('Refresh Redis connection report')}
        >
          <RefreshCcw className={query.isFetching ? 'animate-spin' : ''} />
        </Button>
      </div>

      {query.isLoading ? (
        <div className='grid gap-5 p-4 sm:grid-cols-2 lg:grid-cols-4'>
          {Array.from({ length: 8 }).map((_, index) => (
            <div key={index} className='space-y-2'>
              <Skeleton className='h-3 w-20' />
              <Skeleton className='h-5 w-28' />
            </div>
          ))}
        </div>
      ) : query.isError || !report ? (
        <div className='p-4'>
          <Alert variant='destructive'>
            <AlertTriangle />
            <AlertTitle>{t('Connection report unavailable')}</AlertTitle>
            <AlertDescription>
              {t('We could not load the Redis connection report.')}
            </AlertDescription>
          </Alert>
        </div>
      ) : !report.enabled ? (
        <p className='text-muted-foreground p-4 text-sm'>
          {t('Redis is not configured for this instance.')}
        </p>
      ) : (
        <div className='space-y-5 p-4'>
          {report.error && (
            <Alert variant='destructive'>
              <AlertTriangle />
              <AlertTitle>{t('Connection failed')}</AlertTitle>
              <AlertDescription className='break-all'>
                {report.error}
              </AlertDescription>
            </Alert>
          )}

          {report.info_error && (
            <Alert>
              <AlertTriangle />
              <AlertTitle>{t('Server metrics are incomplete')}</AlertTitle>
              <AlertDescription className='break-all'>
                {t('Server metrics are unavailable: {{message}}', {
                  message: report.info_error,
                })}
              </AlertDescription>
            </Alert>
          )}

          <section aria-labelledby='redis-connection-heading'>
            <h4
              id='redis-connection-heading'
              className='mb-3 text-xs font-semibold uppercase'
            >
              {t('Connection')}
            </h4>
            <dl className='grid gap-x-6 gap-y-4 sm:grid-cols-2 lg:grid-cols-4'>
              <Metric label={t('Endpoint')} value={report.endpoint || '-'} />
              <Metric label={t('Database')} value={String(report.database)} />
              <Metric
                label={t('SSL/TLS')}
                value={t(report.tls_enabled ? 'Yes' : 'No')}
              />
              <Metric
                label={t('Ping latency')}
                value={t('{{value}}ms', {
                  value: report.ping_latency_ms.toFixed(2),
                })}
              />
              <Metric
                label={t('Checked at')}
                value={formatTimestampToDate(report.checked_at)}
              />
              {report.key_count_available && (
                <Metric
                  label={t('Key count')}
                  value={formatNumber(report.server.key_count)}
                />
              )}
            </dl>
          </section>

          {report.server_info_available && (
            <section aria-labelledby='redis-server-heading'>
              <h4
                id='redis-server-heading'
                className='mb-3 text-xs font-semibold uppercase'
              >
                {t('Redis server')}
              </h4>
              <dl className='grid gap-x-6 gap-y-4 sm:grid-cols-2 lg:grid-cols-4'>
                <Metric label={t('Version')} value={report.server.version} />
                <Metric label={t('Mode')} value={report.server.mode} />
                <Metric label={t('Role')} value={report.server.role} />
                <Metric
                  label={t('Uptime')}
                  value={formatUptime(report.server.uptime_seconds)}
                />
                <Metric
                  label={t('Connected clients')}
                  value={formatNumber(report.server.connected_clients)}
                />
                <Metric
                  label={t('Memory')}
                  value={formatBytes(report.server.used_memory_bytes)}
                />
                <Metric
                  label={t('Peak memory')}
                  value={formatBytes(report.server.peak_memory_bytes)}
                />
                <Metric
                  label={t('Maximum memory')}
                  value={
                    report.server.max_memory_bytes > 0
                      ? formatBytes(report.server.max_memory_bytes)
                      : '-'
                  }
                />
                <Metric
                  label={t('Connections received')}
                  value={formatNumber(
                    report.server.total_connections_received
                  )}
                />
                <Metric
                  label={t('Commands processed')}
                  value={formatNumber(report.server.total_commands_processed)}
                />
                <Metric
                  label={t('Operations per second')}
                  value={formatNumber(report.server.operations_per_second)}
                />
              </dl>
            </section>
          )}

          <section aria-labelledby='redis-pool-heading'>
            <h4
              id='redis-pool-heading'
              className='mb-3 text-xs font-semibold uppercase'
            >
              {t('Connection pool')}
            </h4>
            <dl className='grid gap-x-6 gap-y-4 sm:grid-cols-2 lg:grid-cols-4'>
              <Metric
                label={t('Pool size')}
                value={formatNumber(report.pool.size)}
              />
              <Metric
                label={t('Minimum idle connections')}
                value={formatNumber(report.pool.min_idle_conns)}
              />
              <Metric
                label={t('Pool hits')}
                value={formatNumber(report.pool.hits)}
              />
              <Metric
                label={t('Pool misses')}
                value={formatNumber(report.pool.misses)}
              />
              <Metric
                label={t('Pool timeouts')}
                value={formatNumber(report.pool.timeouts)}
              />
              <Metric
                label={t('Total connections')}
                value={formatNumber(report.pool.total_conns)}
              />
              <Metric
                label={t('Idle connections')}
                value={formatNumber(report.pool.idle_conns)}
              />
              <Metric
                label={t('Stale connections')}
                value={formatNumber(report.pool.stale_conns)}
              />
            </dl>
          </section>
        </div>
      )}
    </div>
  )
}
