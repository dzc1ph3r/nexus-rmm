'use client'

import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { MousePointer, Keyboard, Wifi, Volume2, Fullscreen, RotateCcw, LogOut } from 'lucide-react'
import { cn } from '@/lib/utils'

interface RemoteSession {
  id: string
  session_id: string
  device: {
    name: string
    hostname: string
  }
  type: string
  status: string
  protocol: string
  started_at: string
}

export default function RemoteDesktop() {
  const { sessionId } = useParams<{ sessionId: string }>()
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [session, setSession] = useState<RemoteSession | null>(null)
  const [connected, setConnected] = useState(false)
  const [quality, setQuality] = useState('high')
  const [fullscreen, setFullscreen] = useState(false)

  useEffect(() => {
    // In a real implementation, this would connect to the WebSocket/WebRTC session
    // For now, we'll simulate a session
    setSession({
      id: sessionId || '',
      session_id: sessionId || '',
      device: { name: 'WORKSTATION-001', hostname: 'WORKSTATION-001' },
      type: 'desktop',
      status: 'active',
      protocol: 'webrtc',
      started_at: new Date().toISOString(),
    })
    setConnected(true)
  }, [sessionId])

  const handleFullscreen = () => {
    if (canvasRef.current) {
      if (!fullscreen) {
        canvasRef.current.requestFullscreen()
      } else {
        document.exitFullscreen()
      }
      setFullscreen(!fullscreen)
    }
  }

  const handleDisconnect = () => {
    setConnected(false)
    // In real implementation, close WebSocket connection
  }

  return (
    <div className="h-screen flex flex-col bg-black">
      {/* Top toolbar */}
      <div className="flex items-center justify-between p-3 bg-gray-900 border-b border-gray-800">
        <div className="flex items-center gap-4">
          <h2 className="text-white font-medium">
            {session?.device.name} - {session?.device.hostname}
          </h2>
          <span
            className={cn(
              'px-2 py-1 text-xs rounded-full',
              connected ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
            )}
          >
            {connected ? 'Connected' : 'Disconnected'}
          </span>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1 border border-gray-700 rounded-lg p-1">
            <Button
              variant="ghost"
              size="icon"
              className={quality === 'low' ? 'bg-gray-700' : ''}
              onClick={() => setQuality('low')}
              title="Low Quality"
            >
              <span className="text-xs">Low</span>
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className={quality === 'medium' ? 'bg-gray-700' : ''}
              onClick={() => setQuality('medium')}
              title="Medium Quality"
            >
              <span className="text-xs">Med</span>
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className={quality === 'high' ? 'bg-gray-700' : ''}
              onClick={() => setQuality('high')}
              title="High Quality"
            >
              <span className="text-xs">High</span>
            </Button>
          </div>

          <Button variant="ghost" size="icon" onClick={handleFullscreen} title="Fullscreen">
            <Fullscreen className="h-5 w-5" />
          </Button>
          <Button variant="ghost" size="icon" onClick={handleDisconnect} title="Disconnect">
            <LogOut className="h-5 w-5 text-red-400" />
          </Button>
        </div>
      </div>

      {/* Remote desktop viewport */}
      <div className="flex-1 relative overflow-hidden">
        <canvas
          ref={canvasRef}
          className="w-full h-full"
          style={{
            backgroundColor: '#1a1a1a',
            imageRendering: 'pixelated',
          }}
        />
        {!connected && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/80">
            <div className="text-center text-white">
              <Wifi className="h-16 w-16 mx-auto mb-4 text-gray-500" />
              <h3 className="text-xl font-medium mb-2">Connecting to remote session...</h3>
              <p className="text-gray-400">Please wait while we establish the connection</p>
            </div>
          </div>
        )}

        {/* Connection stats overlay */}
        {connected && (
          <div className="absolute bottom-4 right-4 bg-black/60 backdrop-blur-sm rounded-lg p-3 text-xs text-white font-mono">
            <div>Protocol: {session?.protocol?.toUpperCase() || 'WEBRTC'}</div>
            <div>Quality: {quality}</div>
            <div>Latency: <span id="latency">--</span>ms</div>
            <div>FPS: <span id="fps">--</span></div>
          </div>
        )}
      </div>

      {/* Bottom toolbar */}
      <div className="flex items-center justify-between p-3 bg-gray-900 border-t border-gray-800">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" title="Send Ctrl+Alt+Del">
            <Keyboard className="h-5 w-5" />
          </Button>
          <Button variant="ghost" size="icon" title="Toggle Keyboard">
            <Keyboard className="h-5 w-5" />
          </Button>
          <Button variant="ghost" size="icon" title="Toggle Mouse">
            <MousePointer className="h-5 w-5" />
          </Button>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" title="Volume">
            <Volume2 className="h-5 w-5" />
          </Button>
          <Button variant="ghost" size="icon" title="Rotate">
            <RotateCcw className="h-5 w-5" />
          </Button>
        </div>
      </div>
    </div>
  )
}