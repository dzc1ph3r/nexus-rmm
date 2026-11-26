'use client'

import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Plus } from 'lucide-react'
import { NavLink } from 'react-router-dom'

export default function Scripts() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Scripts</h1>
          <p className="text-muted-foreground">Manage and execute scripts</p>
        </div>
        <Button asChild>
          <NavLink to="/scripts/new">
            <Plus className="mr-2 h-4 w-4" />
            Create Script
          </NavLink>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Script Library</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">Script library coming soon...</p>
        </CardContent>
      </Card>
    </div>
  )
}