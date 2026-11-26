'use client'

import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Plus } from 'lucide-react'
import { NavLink } from 'react-router-dom'

export default function Schedules() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Schedules</h1>
          <p className="text-muted-foreground">Manage scheduled tasks</p>
        </div>
        <Button asChild>
          <NavLink to="/schedules/new">
            <Plus className="mr-2 h-4 w-4" />
            Create Schedule
          </NavLink>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Scheduled Tasks</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">Schedules coming soon...</p>
        </CardContent>
      </Card>
    </div>
  )
}