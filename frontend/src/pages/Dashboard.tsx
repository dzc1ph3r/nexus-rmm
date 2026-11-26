'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Monitor, AlertTriangle, CheckCircle, Building2 } from 'lucide-react'
import { cn } from '@/lib/utils'

const stats = [
  {
    name: 'Total Devices',
    value: '1,234',
    change: '+12%',
    icon: Monitor,
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
  },
  {
    name: 'Online',
    value: '1,156',
    change: '+8%',
    icon: CheckCircle,
    color: 'text-green-500',
    bg: 'bg-green-500/10',
  },
  {
    name: 'Alerts',
    value: '23',
    change: '-5%',
    icon: AlertTriangle,
    color: 'text-amber-500',
    bg: 'bg-amber-500/10',
  },
  {
    name: 'Organizations',
    value: '12',
    change: '+2%',
    icon: Building2,
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
  },
]

const recentActivity = [
  { id: 1, action: 'New device enrolled', device: 'WORKSTATION-001', time: '2 min ago', type: 'success' },
  { id: 2, action: 'Script executed', device: 'SERVER-003', time: '15 min ago', type: 'info' },
  { id: 3, action: 'Alert triggered', device: 'LAPTOP-045', time: '1 hour ago', type: 'warning' },
  { id: 4, action: 'Patch installed', device: 'WORKSTATION-012', time: '3 hours ago', type: 'success' },
  { id: 5, action: 'Device went offline', device: 'SERVER-007', time: '5 hours ago', type: 'destructive' },
]

export default function Dashboard() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
          <p className="text-muted-foreground">Overview of your RMM environment</p>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.name}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{stat.name}</CardTitle>
              <stat.icon className={cn('h-4 w-4', stat.color, stat.bg, 'p-2 rounded-lg')} />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stat.value}</div>
              <p className="text-xs text-muted-foreground">{stat.change} from last month</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Recent Activity */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        <Card className="md:col-span-4 lg:col-span-4">
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {recentActivity.map((activity) => (
                <div key={activity.id} className="flex items-center gap-4 p-3 rounded-lg bg-muted/50">
                  <div
                    className={cn(
                      'h-2 w-2 rounded-full',
                      activity.type === 'success' && 'bg-green-500',
                      activity.type === 'info' && 'bg-blue-500',
                      activity.type === 'warning' && 'bg-amber-500',
                      activity.type === 'destructive' && 'bg-red-500'
                    )}
                  />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{activity.action}</p>
                    <p className="text-xs text-muted-foreground">{activity.device}</p>
                  </div>
                  <p className="text-xs text-muted-foreground whitespace-nowrap">{activity.time}</p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Quick Actions */}
        <Card className="md:col-span-2 lg:col-span-2">
          <CardHeader>
            <CardTitle>Quick Actions</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <button className="w-full flex items-center justify-between p-3 rounded-lg border hover:bg-accent transition-colors">
              <span>Run Script</span>
            </button>
            <button className="w-full flex items-center justify-between p-3 rounded-lg border hover:bg-accent transition-colors">
              <span>Create Schedule</span>
            </button>
            <button className="w-full flex items-center justify-between p-3 rounded-lg border hover:bg-accent transition-colors">
              <span>Remote Session</span>
            </button>
            <button className="w-full flex items-center justify-between p-3 rounded-lg border hover:bg-accent transition-colors">
              <span>View Alerts</span>
            </button>
          </CardContent>
        </Card>

        {/* System Health */}
        <Card className="md:col-span-1 lg:col-span-1">
          <CardHeader>
            <CardTitle>System Health</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <div className="flex justify-between text-sm mb-1">
                <span>CPU Usage</span>
                <span className="font-medium">45%</span>
              </div>
              <div className="h-2 bg-muted rounded-full overflow-hidden">
                <div className="h-full bg-primary rounded-full" style={{ width: '45%' }} />
              </div>
            </div>
            <div>
              <div className="flex justify-between text-sm mb-1">
                <span>Memory Usage</span>
                <span className="font-medium">62%</span>
              </div>
              <div className="h-2 bg-muted rounded-full overflow-hidden">
                <div className="h-full bg-primary rounded-full" style={{ width: '62%' }} />
              </div>
            </div>
            <div>
              <div className="flex justify-between text-sm mb-1">
                <span>Disk Usage</span>
                <span className="font-medium">38%</span>
              </div>
              <div className="h-2 bg-muted rounded-full overflow-hidden">
                <div className="h-full bg-primary rounded-full" style={{ width: '38%' }} />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}