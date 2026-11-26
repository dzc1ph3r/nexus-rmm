'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card } from '@/components/ui/card'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Search, Plus, Filter } from 'lucide-react'
import api from '@/lib/api'
import { NavLink } from 'react-router-dom'

interface Device {
  id: string
  agent_id: string
  name: string
  os: string
  os_version: string
  hostname: string
  ip_address: string
  status: string
  is_online: boolean
  last_seen_at: string | null
  created_at: string
}

export default function Devices() {
  const [search, setSearch] = useState('')

  const { data, isLoading } = useQuery({
    queryKey: ['devices', search],
    queryFn: async () => {
      const response = await api.get('/devices', {
        params: { search, limit: 50 },
      })
      return response.data
    },
  })

  const getStatusBadge = (status: string, isOnline: boolean) => {
    if (!isOnline) return <Badge variant="secondary">Offline</Badge>
    switch (status) {
      case 'online':
        return <Badge className="bg-green-100 text-green-800">Online</Badge>
      case 'busy':
        return <Badge className="bg-blue-100 text-blue-800">Busy</Badge>
      case 'maintenance':
        return <Badge className="bg-amber-100 text-amber-800">Maintenance</Badge>
      default:
        return <Badge variant="outline">{status}</Badge>
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Devices</h1>
          <p className="text-muted-foreground">Manage and monitor your devices</p>
        </div>
        <Button asChild>
          <NavLink to="/devices/new">
            <Plus className="mr-2 h-4 w-4" />
            Add Device
          </NavLink>
        </Button>
      </div>

      <div className="flex gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search devices..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-10"
          />
        </div>
        <Button variant="outline">
          <Filter className="mr-2 h-4 w-4" />
          Filters
        </Button>
      </div>

      <Card>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Hostname</TableHead>
              <TableHead>OS</TableHead>
              <TableHead>IP Address</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Last Seen</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center py-8">
                  Loading...
                </TableCell>
              </TableRow>
            ) : data?.data?.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center py-8">
                  No devices found
                </TableCell>
              </TableRow>
            ) : (
              data?.data?.map((device: Device) => (
                <TableRow key={device.id}>
                  <TableCell>
                    <NavLink to={`/devices/${device.id}`} className="font-medium hover:underline">
                      {device.name}
                    </NavLink>
                  </TableCell>
                  <TableCell>{device.hostname}</TableCell>
                  <TableCell>{device.os} {device.os_version}</TableCell>
                  <TableCell>{device.ip_address}</TableCell>
                  <TableCell>{getStatusBadge(device.status, device.is_online)}</TableCell>
                  <TableCell>
                    {device.last_seen_at
                      ? new Date(device.last_seen_at).toLocaleString()
                      : 'Never'}
                  </TableCell>
                  <TableCell className="text-right">
                    <NavLink to={`/devices/${device.id}`} className="text-primary hover:underline">
                      View
                    </NavLink>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>

      {/* Pagination */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          Showing {data?.data?.length || 0} of {data?.total || 0} devices
        </p>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" disabled>Previous</Button>
          <Button variant="outline" size="sm" disabled>Next</Button>
        </div>
      </div>
    </div>
  )
}