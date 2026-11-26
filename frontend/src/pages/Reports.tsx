'use client'

import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Download, Plus } from 'lucide-react'
import { NavLink } from 'react-router-dom'

export default function Reports() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Reports</h1>
          <p className="text-muted-foreground">Generate and view reports</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline">
            <Download className="mr-2 h-4 w-4" />
            Export
          </Button>
          <Button asChild>
            <NavLink to="/reports/new">
              <Plus className="mr-2 h-4 w-4" />
              Create Report
            </NavLink>
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Reports</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">Reporting coming soon...</p>
        </CardContent>
      </Card>
    </div>
  )
}