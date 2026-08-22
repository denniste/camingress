import React from 'react'
import ReactDOM from 'react-dom/client'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import '@livekit/components-styles'
import './index.css'
import App from './App'
import Dashboard from './pages/Dashboard'
import Channels from './pages/Channels'
import Discovery from './pages/Discovery'
import Player from './pages/Player'
import Meeting from './pages/Meeting'

const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: <Dashboard /> },
      { path: 'channels', element: <Channels /> },
      { path: 'discovery', element: <Discovery /> },
      { path: 'meeting', element: <Meeting /> },
      { path: 'player/:room', element: <Player /> },
    ],
  },
])

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>
)
