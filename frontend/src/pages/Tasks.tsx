'use client'

import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Plus } from 'lucide-react'
import { NavLink } from 'react-router-dom'

export default function Tasks() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Tasks</h1>
          <p className="text-muted-foreground">View and manage tasks</p>
        </div>
        <Button asChild>
          <NavLink to="/tasks/new">
            <Plus className="mr-2 h-4 w-4" />
            Create Task
          </NavLink>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Task Queue</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">Task management coming soon...</p>
        </CardContent>
      </Card>
    </div>
  )
}