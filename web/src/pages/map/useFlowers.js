import { useState, useEffect } from 'react'
import { getFlowers, subscribe } from '../../api'

// —— 稀兽花种图层(见 docs/map.md 8)——
// 大世界同时开着二十来朵花种,每朵关着一只混血精灵。点位不是固定的:普通花种每天凌晨 4 点、
// 命定花种每两周周五凌晨 4 点在候选点里重投一遍,战斗里被捕捉/击败的那朵也当场重投,
// 故整层由后端从流量里的花种列表推过来(见 internal/pipeline/flowers.go),前端只管开关与摆放。
//
// **参观好友世界时画的是好友的花**(方便提醒他去捉):后端据 0x039d 判断在谁的世界里,
// 载荷的 visit 标志说明这一套是不是别人的,图层名随之标出来——否则分不清图上这些是谁的花。
//
// **异色/炫彩随花种列表一起来**(mutation_type + glass_info),整层一到就齐全;只在某次下发
// 不带变异那一支时才退回「玩家点开某朵花才知道」,故「未检测」这一态仍要留住
// ——被动抓包不能代替客户端去问,没结果的花绝不能显示成「普通」。
const LS_KEY = 'map.flowerLayer'     // 图层开关(默认关)
const load = (key) => { try { return localStorage.getItem(key) === '1' } catch { return false } }
const save = (key, v) => { try { localStorage.setItem(key, v ? '1' : '0') } catch { /* 隐私模式等 */ } }

// 花里那只精灵的变异状态(与后端 store.Flower* 对应)。**异色与炫彩可以同时成立**
// (游戏里就有既异色又炫彩的精灵):st 只记异色与否,是不是炫彩看 glass 字段(非空 ⇔ 炫彩)。
export const FL_UNDETECTED = 0 // 还没拿到这朵花的变异结果
export const FL_PLAIN = 1      // 普通:非异色非炫彩
export const FL_GLASSY = 2     // 炫彩(非异色)
export const FL_SHINY = 3      // 异色(可能同时炫彩,那时 glass 也非空)

const ST_NAME = { [FL_UNDETECTED]: '未检测', [FL_PLAIN]: '普通', [FL_GLASSY]: '炫彩', [FL_SHINY]: '异色' }

// isMutated 报告这朵花里是不是异色或炫彩个体——开这层就是为了找它们。
export const isMutated = (f) => f.st === FL_SHINY || f.st === FL_GLASSY

// flowerTitle 组一条花种标记的悬停说明:`火神 Lv.60 异色·炫彩` / `铠甲虫 Lv.55 普通`。
// 既异色又炫彩的两样都写出来——那是最稀罕的一种,拿一个盖掉另一个就把它藏起来了。
// 等级后端按星级查表算好(花种列表一到就有);种族在花里的精灵被采收后会变成未知,
// 那时退成「稀兽花种 Lv.55 未检测」。
export function flowerTitle(f) {
  const parts = [f.n || '稀兽花种']
  if (f.lv) parts.push('Lv.' + f.lv)
  if (f.st === FL_SHINY) parts.push(f.glass ? '异色·炫彩' : '异色')
  else parts.push(ST_NAME[f.st] || ST_NAME[FL_UNDETECTED])
  return parts.join(' ')
}

// useFlowers 管理花种图层:订阅后端推送、按当前场景与开关筛出可绘制的标记。
export function useFlowers(account, res) {
  const [flowers, setFlowers] = useState([])
  const [visit, setVisit] = useState(false) // 当前这套花是不是好友世界的
  const [on, setOn] = useState(() => load(LS_KEY))

  useEffect(() => {
    let alive = true
    setFlowers([])
    getFlowers().then((d) => {
      if (!alive || !d) return
      setFlowers(d.flowers || [])
      setVisit(!!d.visit)
    }).catch(() => {})
    return () => { alive = false }
  }, [account])

  // 后端每次列表刷新/检测出结果/花被打掉都推全量,直接替换即可(都是低频事件)。
  useEffect(() => subscribe((m) => {
    if (m.type === 'flowers') {
      setFlowers(m.data.flowers || [])
      setVisit(!!m.data.visit)
    }
  }), [account])

  const toggle = () => setOn((v) => { save(LS_KEY, !v); return !v })

  // 花种只在有底图的大世界场景刷,进副本/家园时本层自然为空。
  const here = flowers.filter((f) => f.res === res)
  const marks = on ? here : []
  const mutated = here.filter(isMutated).length

  return { marks, num: here.length, mutated, visit, on, toggle }
}
