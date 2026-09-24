import React from 'react'
import { createRoot } from 'react-dom/client'
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom'
import App from './App'
import PetList from './pages/pet-list/PetList'
import Events from './pages/events/Events'
import PetDetail from './pages/PetDetail'
import MapPage from './pages/map/MapPage'
import EggList from './pages/eggs/EggList'
import { probePets } from './pets'
// 样式按「基础 → 壳 → 共用面板/部件 → 各页」顺序引入(同名选择器的层叠顺序有意义)。
import './styles/base.css'
import './styles/shell.css'
import './styles/panel.css'
import './styles/pet.css'
import './styles/list.css'
import './styles/events.css'
import './styles/eggs.css'
import './styles/detail.css'
import './styles/map.css'

// 页面打开时探一次本机桌宠(色卡据此决定唤起桌宠预览还是跳 rkpet,见 pets.js)。
probePets()

createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <HashRouter>
      <Routes>
        <Route element={<App />}>
          <Route index element={<Navigate to="/pets" replace />} />
          <Route path="pets" element={<PetList />} />
          <Route path="pets/:gid" element={<PetDetail />} />
          <Route path="events" element={<Events />} />
          <Route path="eggs" element={<EggList />} />
          <Route path="map" element={<MapPage />} />
        </Route>
      </Routes>
    </HashRouter>
  </React.StrictMode>
)
