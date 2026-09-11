# 协议说明:游戏消息

游戏客户端通过 TCP 连接远端服务器 **8195** 端口,使用腾讯 tsf4g 框架的 GCP 模式通信。
字节层(GCP 包头布局、会话密钥与 AES 解密、应用层 internal header、TCP 重组)是 rocom-parse 的
`gcp`/`capture` 包实现,记录在其 docs/protocol.md;本文只记录本项目解析的**游戏消息**,
opcode 名称对应 `ZoneSvrCmd` 枚举(rocom-parse `pcapdump` 可按名称转储)。

## 1. 宠物列表消息

- 客户端打开宠物仓库时发 `ZONE_GET_PET_INFO_BY_PAGE_REQ(0x1345)`；
- 服务器分页回 `ZONE_GET_PET_INFO_BY_PAGE_RSP(0x1346)`，每页约 40KB(常跨多 TCP 段)；
- RSP body 是 `ZoneGetPetInfoByPageRsp`：`total_page=2`、`req_page=3`、
  `pet_info=4`(`PetDataInfoList`，含 `repeated PetData`)、`page_num=5`。

本项目只手动取 field 4 再 `proto.Unmarshal` 成 `PetDataInfoList`，无需编译庞大的 zonesvr 消息。
解析细节见 [parsing.md](parsing.md)。

## 2. 实时位置与场景消息(`internal/scene`,实时地图页)

只跟踪**登录账号自己**的位置(不解析其他玩家/AOI)。字段语义经当前版客户端 Scene luac 坐实、
真实 pcap 验证。

| opcode | 方向 | 消息 | 用途 |
| --- | --- | --- | --- |
| `0x0133` | c2s | `ZONE_SCENE_MOVE_REQ` | 自己移动:`to_pos`(2)、`speed`(4)、`to_rot`(3)、`move_mode`(6)、`stop_move`(8)、`move_seg_list`(12)、`scene_cfg_id`(17) |
| `0x0152` | s2c | `ZONE_ENTER_SCENE_RSP` | 进入场景:`scene_cfg_id`(2)、`scene_res_cfg_id`(3) |
| `0x015c` | s2c | `ZONE_SCENE_TELEPORT_NOTIFY` | 传送:`to_scene_cfg_id`(11)、`to_scene_res_cfg_id`(12)、**落点 `to_pt`(14,`Point{pos,dir}`)**、`home_room_level`(31) |
| `0x0414` | s2c | `ZONE_SCENE_PLAY_ACTS_NOTIFY` | 动作集合 `acts`(1);其中 `enterted_catcher`(61)/`left_catcher`(62)= 区域进/出:`{actor_id, area_id, area_func_conf_id}` |
| `0x1838` | c2s | `ZONE_SCENE_CLIENT_CAVE_STATE_REQ` | 洞穴层:`cave_name`(1,string)+ `pos`(2);**只在传送进流送洞穴时才发,不可靠,未用** |
| `0x1505` | s2c | `ZONE_SCENE_CLIENT_CAVE_STATE_NOTIFY` | 同上(服务器侧下发) |

- **坐标**:`Position{x,y,z}` = UE 世界坐标,1 单位 = 1 厘米,取整。玩家 `to_pos.z` 是**脚底**高度
  (角色中心 +85);`to_rot`/`Point.dir` **不是坐标是旋转**(`FRotator×10`=0.1 度,x=Roll/y=Pitch/z=Yaw)。
- **移动包是事件驱动的,不是定频**(pcap 实测):**操作有变化**(改方向/变速)时约 **0.1s** 一包
  (0.08–0.16s,≈8 次/秒);**输入不变**时退化成 **2.5–3s** 一次心跳(实测密集落在 2.4–3.06s);
  停下补一个 `stop_move=true`(常连发 2–3 条同坐标)。故**收到才画会一顿一顿跳**(心跳期定住数秒)。
  地面与飞行同理——飞行时快速打方向同样是 0.1s 一包(实测绕圈 226 个包,间隔中位 0.108s)。
- **`speed`(field 4)是速度向量**(厘米/秒,跑动约 |v|≈416),客户端据此给其他玩家做平滑。本项目同法:
  后端把它按同一投影换算成「归一化底图坐标/秒」随位置一起推给前端,前端在两包之间逐帧外推
  `pos + speed×Δt`(航位推算)。pcap 回放验证:用上一包外推到下一包实际位置,误差中位 **3cm**
  (原地不动则 43cm),3s 的直线心跳段也仅几米。
- **心跳期里玩家可能其实在转弯**,轨迹全靠 `move_seg_list` 补报(这是实时地图平滑的关键):
  - 触发上报的是**操作变化**,不是位置变化。**推住摇杆让坐骑自行盘旋、或直线巡航**时输入不变,
    客户端就只发 2.5–3s 的心跳——**那几秒里哪怕转了 175°,中途一个包都没有**(实测飞行绕圈:
    两个心跳包之间 `to_rot` 差 175°/154°)。而快速连续打方向时,飞行同样是 0.1s 一包。
    所以「静默转弯最多晚一个心跳(~3s)才可见」是游戏的上报节奏决定的,与本项目链路无关。
  - **`move_seg_list`(field 12,`MoveSegmentInfo{pos, time_stamp}`)补报那段空窗的真实轨迹**:
    按约 0.3s 一个点回传(3s 心跳里通常 7–9 个点),末点时刻≈包时刻、位置≈`to_pos`(实测 `to_pos`
    略新 0.2–0.6 个采样步长,故后端把 `to_pos` 补作轨迹终点)。密集上报时它为空或只有一两个点
    (故以 `SegSpan` 而非「是否飞行」判断值不值得回放,见 [architecture.md](architecture.md) 7)。
  - `speed` 的方向是**机头朝向**(等于 `to_rot`),盘旋时与实际行进方向能差几十度。即便如此,
    **直线外推仍是空窗期各策略中最准的**(实测以补报轨迹为真值:直线巡航均值 247cm、静默转弯
    616cm;而阻尼外推 533/809、圆弧外推 484/660、定住不动 769/1052 都更差)。
- `move_mode`(field 6)= `SceneMoveType`:1/2/3 地面(骑乘地面跑仍算地面)、6-9 飞行
  (`SMT_FLY_UP/DOWN/GLIDING/STATIC`)、11-13 游泳、16-19 攀爬。**上报节奏与该模式无关**(同上)。
- **当前场景 res 必须从 0x0152/0x015c 跟踪**,不能只看移动包的 `scene_cfg_id`——一个 scene_cfg_id
  可对应多个 scene_res(103 → 10003 卡洛西亚 或 10018 魔法学院)。
- **传送落点 `to_pt` 要用起来**:传送通知一下发就带着目的地坐标/朝向,而客户端要过几秒(加载)才落地
  并开始发移动包(实测 3-5s)。据此可立刻把地图切到目的地;否则地图会停在原地干等,玩家落地后若站着
  不动更是一直不更新。实测(3 份 pcap)`to_pt` 与落地后首个移动包的坐标/朝向一致(误差几厘米)。
- 状态机:收 0x0152/0x015c 更新当前 scene_res(并清空区域集合);收 0x0133 取 `to_pos` 配当前
  scene_res 投影;收 0x0414 的区域进/出事件维护「玩家当前所在区域」,据其 `area_func_conf_id`
  选洞穴/楼层分层地图(见 map.md 2)。**区域进/出是服务器在玩家真正踩进/离开触发体(3D 体积)时
  才下发的,是选层的唯一权威依据**——按位置点做 2D 多边形判定会在洞穴正上方的地表误叠洞穴图。
- **重启恢复**:进入/传送只在切场景时下发,区域进/出也只在跨越触发体时下发,游戏中途都不重发,故
  当前 scene_res 与区域集合须像会话密钥一样落盘(`sessions.scene_res` / `sessions.areas`),
  抓包服务重启后从缓存预热;否则重启后虽能解密移动包,却因不知 res 而无法
  定位底图。另有兜底:res 未知时(中途开抓/无缓存)用移动包 `scene_cfg_id` 经 SCENE_CONF 取默认 res
  (`DB.DefaultSceneRes`,同 cfg 多 res 时取主行,子场景仍以缓存/通知的精确 res 为准)。
