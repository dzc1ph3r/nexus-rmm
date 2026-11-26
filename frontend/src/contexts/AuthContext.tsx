'use client'

import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '@/lib/api'
import { toast } from '@/hooks/use-toast'

interface User {
  id: string
  email: string
  name: string
  role: string
  organization_id: string
  mfa_enabled: boolean
  last_login_at: string | null
}

interface AuthContextType {
  user: User | null
  loading: boolean
  login: (email: string, password: string, mfa?: string) => Promise<void>
  logout: () => void
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

  const refreshUser = async () => {
    try {
      const response = await api.get('/auth/me')
      setUser(response.data)
    } catch (error) {
      setUser(null)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const token = localStorage.getItem('access_token')
    if (token) {
      refreshUser()
    } else {
      setLoading(false)
    }
  }, [])

  const login = async (email: string, password: string, mfa?: string) => {
    try {
      const response = await api.post('/auth/login', { email, password, mfa_code: mfa })

      if (response.data.mfa_required) {
        // MFA required - we'll handle this in the login page
        throw { response: { data: { mfa_required: true } } }
      }

      localStorage.setItem('access_token', response.data.access_token)
      localStorage.setItem('refresh_token', response.data.refresh_token)
      setUser(response.data.user)
      toast({ title: 'Welcome back!', description: `Logged in as ${response.data.user.name}` })
      navigate('/dashboard')
    } catch (error: any) {
      if (error.response?.data?.mfa_required) {
        throw error
      }
      toast({
        title: 'Login failed',
        description: error.response?.data?.error || 'Invalid credentials',
        variant: 'destructive',
      })
      throw error
    }
  }

  const logout = () => {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    setUser(null)
    navigate('/login')
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}