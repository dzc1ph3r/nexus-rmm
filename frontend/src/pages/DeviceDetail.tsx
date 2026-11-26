'use client'

import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { Monitor, Cpu, MemoryStick, Wifi, CheckCircle, MoreHorizontal } from 'lucide-react'
import api from '@/lib/api'

interface Device {
  id: string
  agent_id: string
  name: string
  description: string
  os: string
  os_version: string
  architecture: string
  hostname: string
  ip_address: string
  mac_address: string
  status: string
  version: string
  is_online: boolean
  last_seen_at: string | null
  organization?: { name: string }
  site?: { name: string }
  group?: { name: string }
  hardware_info?: {
    cpu_model: string
    cpu_cores: number
    cpu_threads: number
    ram_total: number
    manufacturer: string
    model: string
    serial_number: string
    gpus?: Array<{ name: string; vram: number }>
    disks?: Array<{ name: string; size: number; type: string }>
    network_interfaces?: Array<{ name: string; mac_address: string; ip_addresses: string[] }>
  }
}

export default function DeviceDetail() {
  const { id } = useParams<{ id: string }>()

  const { data: device, isLoading } = useQuery<Device>({
    queryKey: ['device', id],
    queryFn: async () => {
      const response = await api.get(`/devices/${id}`)
      return response.data
    },
    enabled: !!id,
  })

  if (isLoading) {
    return <div className="flex items-center justify-center h-64">Loading...</div>
  }

  if (!device) {
    return <div className="text-center py-8">Device not found</div>
  }

  const d = device

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{d.name}</h1>
          <p className="text-muted-foreground">{d.agent_id}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm">
            <MoreHorizontal className="mr-2 h-4 w-4" />
            Actions
          </Button>
          <Button>
            <CheckCircle className="mr-2 h-4 w-4" />
            Remote Desktop
          </Button>
        </div>
      </div>

      {/* Status Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Status</CardTitle>
            <Monitor className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Badge variant={d.is_online ? 'success' : 'secondary'}>
                {d.is_online ? 'Online' : 'Offline'}
              </Badge>
              <span className="text-sm text-muted-foreground">{d.status}</span>
            </div>
            <p className="text-xs text-muted-foreground">
              Last seen: {d.last_seen_at ? new Date(d.last_seen_at).toLocaleString() : 'Never'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Operating System</CardTitle>
            <Cpu className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{d.os} {d.os_version}</div>
            <p className="text-xs text-muted-foreground">{d.architecture}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Hostname</CardTitle>
            <Wifi className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{d.hostname}</div>
            <p className="text-xs text-muted-foreground">{d.ip_address}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Agent Version</CardTitle>
            <MemoryStick className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{d.version}</div>
            <p className="text-xs text-muted-foreground">MAC: {d.mac_address}</p>
          </CardContent>
        </Card>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="hardware">Hardware</TabsTrigger>
          <TabsTrigger value="software">Software</TabsTrigger>
          <TabsTrigger value="metrics">Metrics</TabsTrigger>
          <TabsTrigger value="alerts">Alerts</TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>System Information</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Agent ID</p>
                    <p className="font-mono text-sm">{d.agent_id}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Organization</p>
                    <p className="text-sm">{d.organization?.name || 'N/A'}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Site</p>
                    <p className="text-sm">{d.site?.name || 'N/A'}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Group</p>
                    <p className="text-sm">{d.group?.name || 'N/A'}</p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Network</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm text-muted-foreground">IP Address</p>
                      <p className="font-mono text-sm">{d.ip_address}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">MAC Address</p>
                      <p className="font-mono text-sm">{d.mac_address}</p>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="hardware">
          {d.hardware_info ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              <Card>
                <CardHeader>
                  <CardTitle>CPU</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <p className="font-medium">{d.hardware_info.cpu_model}</p>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <p className="text-muted-foreground">Cores</p>
                      <p>{d.hardware_info.cpu_cores}</p>
                    </div>
                    <div>
                      <p className="text-muted-foreground">Threads</p>
                      <p>{d.hardware_info.cpu_threads}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Memory</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <p className="text-2xl font-bold">
                    {(d.hardware_info.ram_total / (1024 ** 3)).toFixed(1)} GB
                  </p>
                  <p className="text-sm text-muted-foreground">Total RAM</p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>System</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2 text-sm">
                  <div className="grid grid-cols-2 gap-2">
                    <span className="text-muted-foreground">Manufacturer</span>
                    <span>{d.hardware_info.manufacturer}</span>
                    <span className="text-muted-foreground">Model</span>
                    <span>{d.hardware_info.model}</span>
                    <span className="text-muted-foreground">Serial</span>
                    <span>{d.hardware_info.serial_number}</span>
                  </div>
                </CardContent>
              </Card>

              {d.hardware_info.gpus && d.hardware_info.gpus.length > 0 && (
                <Card className="md:col-span-2">
                  <CardHeader>
                    <CardTitle>GPUs</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Name</TableHead>
                          <TableHead>VRAM</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {d.hardware_info.gpus.map((gpu: { name: string; vram: number }, index: number) => (
                          <TableRow key={index}>
                            <TableCell>{gpu.name}</TableCell>
                            <TableCell>{(gpu.vram / (1024 ** 3)).toFixed(1)} GB</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              )}

              {d.hardware_info.disks && d.hardware_info.disks.length > 0 && (
                <Card className="md:col-span-2">
                  <CardHeader>
                    <CardTitle>Disks</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Name</TableHead>
                          <TableHead>Size</TableHead>
                          <TableHead>Type</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {d.hardware_info.disks.map((disk: { name: string; size: number; type: string }, index: number) => (
                          <TableRow key={index}>
                            <TableCell>{disk.name}</TableCell>
                            <TableCell>{(disk.size / (1024 ** 3)).toFixed(1)} GB</TableCell>
                            <TableCell>{disk.type}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              )}

              {d.hardware_info.network_interfaces && d.hardware_info.network_interfaces.length > 0 && (
                <Card className="md:col-span-2">
                  <CardHeader>
                    <CardTitle>Network Interfaces</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Name</TableHead>
                          <TableHead>MAC</TableHead>
                          <TableHead>IPs</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {d.hardware_info.network_interfaces.map((nic: { name: string; mac_address: string; ip_addresses: string[] }, index: number) => (
                          <TableRow key={index}>
                            <TableCell>{nic.name}</TableCell>
                            <TableCell>{nic.mac_address}</TableCell>
                            <TableCell>{nic.ip_addresses.join(', ')}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              )}
            </div>
          ) : (
            <Card>
              <CardContent className="py-8 text-center text-muted-foreground">
                No hardware information available
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="software">
          <Card>
            <CardHeader>
              <CardTitle>Installed Software</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">Software inventory not loaded</p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="metrics">
          <Card>
            <CardHeader>
              <CardTitle>Performance Metrics</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">Metrics not loaded</p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="alerts">
          <Card>
            <CardHeader>
              <CardTitle>Device Alerts</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">Alerts not loaded</p>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}