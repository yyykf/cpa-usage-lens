import type { TrendPoint } from '../types'
import {
  ResponsiveContainer,
  LineChart,
  Line,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
} from 'recharts'

export default function TrendChart({ data }: { data: TrendPoint[] }) {
  return (
    <div className="rounded-2xl border border-border bg-card p-5 md:p-6 hover:border-primary/40 transition-colors">
      <h2 className="text-foreground text-base font-semibold">每日趋势</h2>
      {data.length === 0 ? (
        <div className="flex h-[280px] items-center justify-center text-muted-foreground">
          暂无数据
        </div>
      ) : (
        <div className="mt-4">
          <ResponsiveContainer width="100%" height={280}>
            <LineChart data={data}>
              <CartesianGrid stroke="#232A40" strokeDasharray="3 3" />
              <XAxis dataKey="date" tick={{ fill: '#8B93A7', fontSize: 12 }} />
              <YAxis tick={{ fill: '#8B93A7', fontSize: 12 }} />
              <Tooltip
                contentStyle={{
                  background: '#111627',
                  border: '1px solid #232A40',
                  borderRadius: 8,
                }}
              />
              <Legend />
              <Line
                type="monotone"
                dataKey="requests"
                stroke="#3B82F6"
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="tokens"
                stroke="#22D3EE"
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="cost"
                stroke="#F97316"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  )
}
