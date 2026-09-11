import React from 'react'
import { imgURL } from './icons'

// 炫彩色卡:复刻游戏内点开炫彩标记弹出的那张小卡(客户端 UMG_Pet_DazzlingTips_C),
// 素材与配色由后端 gamedata.GlassCard 下发(素材见 rocom-parse docs/data.md 的炫彩段)。两种画法:
//   隐藏炫彩(赛季款/常驻款)→ card 就是整张烤好的图,配色已画进去,原样贴上;
//   普通炫彩 → 三层叠:base(圆角矩形)着 color2 打底 → wave(上半波浪)着 color1
//              → card(粒子层)原色压最上。base/wave 是纯白+alpha 的遮罩,用 CSS mask 上色。

// maskStyle 把一张遮罩图 + 一个颜色变成一层:颜色铺满,再按遮罩的 alpha 裁形。
// 两个前缀都写死在内联 style 上:详情页导出 PNG 走 html-to-image,它只认内联的
// mask-image / -webkit-mask-image(会把 url 转成 data URI),写在 CSS 类里导出会丢形状。
function maskStyle(src, color) {
  const url = `url(${imgURL(src)})`
  return { background: color, maskImage: url, WebkitMaskImage: url }
}

// GlassCard 色卡,摆在详情页身份区(昵称行 + 天分/系别行)的右侧,竖向跨这两行。
// 界面上只留卡:外观名与赛季归属由左边名称行那枚炫彩标记(badges.jsx 的 Marks)负责,
// 这里再写一遍就是同一句话挂两处。缺素材时不渲染。
export function GlassCard({ p }) {
  const g = p && p.glass
  if (!g || !g.card) return null
  return (
    <div className="glass-card">
      {!g.hidden && g.base && g.color2 && <span className="glass-layer" style={maskStyle(g.base, g.color2)} />}
      {!g.hidden && g.wave && g.color1 && <span className="glass-layer glass-wave" style={maskStyle(g.wave, g.color1)} />}
      <img src={imgURL(g.card)} alt={g.name} />
    </div>
  )
}
