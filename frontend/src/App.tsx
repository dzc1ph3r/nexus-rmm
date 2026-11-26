import { Routes, Route, Navigate } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/queryClient'
import { AuthProvider } from '@/contexts/AuthContext'
import Layout from '@/components/layout/Layout'
import Login from '@/pages/Login'
import Dashboard from '@/pages/Dashboard'
import Devices from '@/pages/Devices'
import DeviceDetail from '@/pages/DeviceDetail'
import Organizations from '@/pages/Organizations'
import Scripts from '@/pages/Scripts'
import Schedules from '@/pages/Schedules'
import Tasks from '@/pages/Tasks'
import Reports from '@/pages/Reports'
import Settings from '@/pages/Settings'
import RemoteDesktop from '@/pages/RemoteDesktop'

function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route element={<Layout />}>
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/devices" element={<Devices />} />
        <Route path="/devices/:id" element={<DeviceDetail />} />
        <Route path="/organizations" element={<Organizations />} />
        <Route path="/scripts" element={<Scripts />} />
        <Route path="/schedules" element={<Schedules />} />
        <Route path="/tasks" element={<Tasks />} />
        <Route path="/reports" element={<Reports />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/remote/:sessionId" element={<RemoteDesktop />} />
      </Route>
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  )
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <AppRoutes />
      </AuthProvider>
    </QueryClientProvider>
  )
}

export default App