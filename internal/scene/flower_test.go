package scene

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

// 下面的字段值取自真实 pcap(rocom-20260825-225024 炫彩魔力猫 / rocom-20260825-230448 非炫彩火神 /
// rocom-20260825-233622 非炫彩普通花种铠甲虫),见 docs/map.md 8。

func vint(b []byte, num protowire.Number, v uint64) []byte {
	b = protowire.AppendTag(b, num, protowire.VarintType)
	return protowire.AppendVarint(b, v)
}

func msg(b []byte, num protowire.Number, sub []byte) []byte {
	b = protowire.AppendTag(b, num, protowire.BytesType)
	return protowire.AppendBytes(b, sub)
}

// bossNpcInfo 构造花种列表里的一项(BossNpcInfo 的实测字段子集)。
func bossNpcInfo(cfgID uint32, star int32, petbase, content uint32, obj uint64, end int64, spec uint32) []byte {
	var b []byte
	b = vint(b, 1, uint64(cfgID))
	b = vint(b, 2, uint64(star))
	b = vint(b, 5, uint64(petbase))
	b = vint(b, 6, uint64(content)<<flowerLogicShift|0xFFFE0001)
	b = vint(b, 7, obj)
	b = vint(b, 8, uint64(content))
	b = vint(b, 10, uint64(end))
	if spec != 0 {
		b = vint(b, 11, uint64(spec))
	}
	return b
}

func TestParseFlowerList(t *testing.T) {
	var flowers []byte
	flowers = msg(flowers, 1, bossNpcInfo(700002, 7, 3007, 2606405, 9279722995167894922, 1787860799, 20038))
	flowers = msg(flowers, 1, bossNpcInfo(20143, 5, 3439, 140309, 9279722995167901301, 1787688000, 0))
	var leaders []byte // world_leader_npcs(3):同样是 BossNpcInfos,但不是花种,不该混进来
	leaders = msg(leaders, 1, bossNpcInfo(65516, 0, 4020, 1200809, 9279722995167889376, 0, 0))

	var body []byte
	body = msg(body, 1, vint(nil, 1, 0)) // ret_info.ret_code = 0
	body = msg(body, 2, flowers)
	body = msg(body, 3, leaders)
	body = append(body, tsf4gMark...) // 尾标记之后的内容不参与解析

	got, ok := ParseFlowerList(body)
	if !ok {
		t.Fatal("自己世界的花种列表应可用")
	}
	if got.Visiting {
		t.Error("自己世界的列表不该标成参观中")
	}
	if len(got.Seeds) != 2 {
		t.Fatalf("花种数 = %d, 期望 2(world_leader_npcs 不该收)", len(got.Seeds))
	}
	f := got.Seeds[0]
	if f.ObjID != 9279722995167894922 || f.CfgID != 700002 || f.Star != 7 ||
		f.PetBase != 3007 || f.ContentID != 2606405 || f.EndTS != 1787860799 || f.SpecID != 20038 {
		t.Errorf("命定花种解析错: %+v", f)
	}
	if got.Seeds[1].SpecID != 0 || got.Seeds[1].Star != 5 {
		t.Errorf("普通花种解析错: %+v", got.Seeds[1])
	}
}

// ret_code 非 0 时必须 ok=false:调用方据此**不**清空库里的花种,否则一次失败的请求
// 会把整层抹掉。
func TestParseFlowerListRejected(t *testing.T) {
	var body []byte
	body = msg(body, 1, vint(nil, 1, 10001)) // ret_code != 0
	if got, ok := ParseFlowerList(body); ok {
		t.Errorf("ret_code 非 0 应 ok=false, 得到 %+v", got)
	}
}

// 传送去好友的世界后打开花种面板,回包讲的是**那个世界**的花(每条都带
// visit_flower_seed_boss_datas):照样解出来(参观时画在地图上提醒好友去捉),但必须标出
// Visiting,调用方据此只放进内存那套、不写自己的库。
// 实测(rocom-20260826-002049):好友世界那份 23 条条条都带,自己的两份一条都没有。
func TestParseFlowerListVisiting(t *testing.T) {
	entry := bossNpcInfo(20143, 5, 3147, 2100070, 9279722995167901302, 1787688000, 0)
	entry = msg(entry, visitFlowerField, vint(nil, 3, 3439)) // visit_flower_seed_boss_datas
	var body []byte
	body = msg(body, 1, vint(nil, 1, 0))
	body = msg(body, 2, msg(nil, 1, entry))
	got, ok := ParseFlowerList(body)
	if !ok || !got.Visiting {
		t.Fatalf("好友世界的列表应解出且标 Visiting, 得到 ok=%v %+v", ok, got)
	}
	if len(got.Seeds) != 1 || got.Seeds[0].ObjID != 9279722995167901302 {
		t.Errorf("好友世界的花种没解出来: %+v", got.Seeds)
	}
}

// 同理:在好友的世界里点开某朵花,0x0338 也带 visit_flower_seed_boss_datas(字段号 35)。
func TestParseFlowerBattleVisiting(t *testing.T) {
	info := teamBattleInfo(700002, 7, 3007, 2606405, 9279722995167894922, true, 1, 1048599)
	info = msg(info, 35, vint(nil, 3, 3007)) // visit_flower_seed_boss_datas
	body := msg(msg(nil, 1, vint(nil, 1, 0)), 2, info)
	got, ok := ParseFlowerBattle(body)
	if !ok || !got.Visiting {
		t.Fatalf("好友世界的单朵详情应解出且标 Visiting, 得到 ok=%v %+v", ok, got)
	}
	if got.GlassType != 1 {
		t.Errorf("好友那朵的炫彩也该解出来(要提醒他去捉): %+v", got)
	}
}

// 0x039d 给出「现在在谁的世界里」:非 0 = 参观中,0/字段缺省 = 回到自己的世界。
func TestParseVisitOwner(t *testing.T) {
	body := vint(vint(nil, 1, 839694713), 2, 1) // online_visit_owner + first_enter_visiting
	if got := ParseVisitOwner(body); got != 839694713 {
		t.Errorf("ParseVisitOwner = %d, 期望 839694713", got)
	}
	if got := ParseVisitOwner(vint(nil, 2, 0)); got != 0 {
		t.Errorf("字段缺省应为 0, 得到 %d", got)
	}
}

// teamBattleInfo 构造单朵花的战斗信息。randedGlass 是 randed_battle_npc_glass(4):
// 实测非炫彩的命定花种同样是 true,故解析必须无视它,只看 glass_info。
func teamBattleInfo(cfgID uint32, star int32, petbase, content uint32, obj uint64, randedGlass bool, glassType, glassValue int32) []byte {
	var b []byte
	b = vint(b, 1, uint64(cfgID))
	b = vint(b, 2, uint64(star))
	if randedGlass {
		b = vint(b, 4, 1)
	}
	b = vint(b, 5, uint64(petbase))
	b = vint(b, 6, uint64(content)<<flowerLogicShift|0xFFFE0001)
	b = vint(b, 7, obj)
	b = vint(b, 25, 20038)
	b = vint(b, 27, 1787860799)
	if glassType != 0 || glassValue != 0 {
		g := vint(vint(nil, 1, uint64(glassType)), 2, uint64(glassValue))
		b = msg(b, 32, g)
	}
	return b
}

func TestParseFlowerBattle(t *testing.T) {
	// 炫彩魔力猫:glass_type=GT_COMMON(1), glass_value=1048599(亮X暗 - 浅绿青·四角星)
	body := msg(msg(nil, 1, vint(nil, 1, 0)), 2,
		teamBattleInfo(700002, 7, 3007, 2606405, 9279722995167894922, true, 1, 1048599))
	b, ok := ParseFlowerBattle(body)
	if !ok {
		t.Fatal("炫彩花种应解析成功")
	}
	if b.GlassType != 1 || b.GlassValue != 1048599 || b.ObjID != 9279722995167894922 || b.Star != 7 {
		t.Errorf("炫彩花种解析错: %+v", b)
	}
	// npc_logic_id 的高 32 位即刷新行 id(本消息不直接给 content_cfg_id)
	if b.ContentID != 2606405 {
		t.Errorf("ContentID = %d, 期望 2606405(由 npc_logic_id 高 32 位还原)", b.ContentID)
	}

	// 非炫彩火神:randed_battle_npc_glass 同样是 true,但 glass_info 是 GT_NULL。
	// 这一条是整个判据的关键——拿 randed 当标志会把每一朵命定花种都判成炫彩。
	body = msg(msg(nil, 1, vint(nil, 1, 0)), 2,
		teamBattleInfo(700003, 7, 3006, 2606406, 9279722995167894923, true, 0, 0))
	b, ok = ParseFlowerBattle(body)
	if !ok {
		t.Fatal("非炫彩花种应解析成功")
	}
	if b.GlassType != 0 || b.GlassValue != 0 {
		t.Errorf("非炫彩花种不该解出炫彩: %+v", b)
	}
}

// bossNpcMutation 追加一项的变异字段:mutation_type(9)非 0 才发(与服务器一致),
// glass_info(25)则连非炫彩的花也发(glass_type=GT_NULL)——**有这一支就说明这次下发带变异结果**。
func bossNpcMutation(b []byte, mtype, gtype, gvalue int32) []byte {
	if mtype != 0 {
		b = vint(b, 9, uint64(mtype))
	}
	var gi []byte
	gi = vint(gi, 1, uint64(gtype))
	gi = vint(gi, 2, uint64(gvalue))
	return msg(b, 25, gi)
}

// 列表自带异色/炫彩(2026-09-10 大版本起,实测每条都有 glass_info):
// 异色与炫彩可同时置位,glass_info 说明是哪一种炫彩;没有 glass_info 那一支的条目留给 0x0338 兜底。
func TestParseFlowerListMutation(t *testing.T) {
	shiny := bossNpcMutation(bossNpcInfo(20135, 5, 3012, 2100044, 9279722995167909004, 1789243200, 0),
		MutationShiny|MutationGlass, 2, 1)
	glassy := bossNpcMutation(bossNpcInfo(20143, 5, 3174, 2100034, 9279722995167909003, 1789243200, 0),
		MutationGlass, 1, 0x100002)
	plain := bossNpcMutation(bossNpcInfo(20139, 5, 3335, 652588, 9279722995167909005, 1789243200, 0), 0, 0, 0)
	legacy := bossNpcInfo(20128, 5, 3401, 652590, 9279722995167909006, 1789243200, 0) // 不带变异那一支

	var flowers []byte
	for _, one := range [][]byte{shiny, glassy, plain, legacy} {
		flowers = msg(flowers, 1, one)
	}
	var body []byte
	body = msg(body, 1, vint(nil, 1, 0)) // ret_info.ret_code = 0
	body = msg(body, 2, flowers)
	body = append(body, tsf4gMark...)

	got, ok := ParseFlowerList(body)
	if !ok || len(got.Seeds) != 4 {
		t.Fatalf("解析结果 ok=%v 花种数=%d, 期望 true/4", ok, len(got.Seeds))
	}
	want := []struct {
		mutation           int32
		gtype, gvalue      int32
		hasMutation        bool
		shinyBit, glassBit bool
	}{
		{MutationShiny | MutationGlass, 2, 1, true, true, true},
		{MutationGlass, 1, 0x100002, true, false, true},
		{0, 0, 0, true, false, false},
		{0, 0, 0, false, false, false},
	}
	for i, w := range want {
		f := got.Seeds[i]
		if f.HasMutation != w.hasMutation {
			t.Errorf("第 %d 朵 HasMutation = %v, 期望 %v", i+1, f.HasMutation, w.hasMutation)
		}
		if f.MutationType != w.mutation || f.GlassType != w.gtype || f.GlassValue != w.gvalue {
			t.Errorf("第 %d 朵 = (mutation %d, glass %d/%d), 期望 (%d, %d/%d)",
				i+1, f.MutationType, f.GlassType, f.GlassValue, w.mutation, w.gtype, w.gvalue)
		}
		if (f.MutationType&MutationShiny != 0) != w.shinyBit || (f.MutationType&MutationGlass != 0) != w.glassBit {
			t.Errorf("第 %d 朵位标志解读有误: mutation=%d", i+1, f.MutationType)
		}
		// 描述列照旧解出来(变异字段不该挤掉别的)
		if f.ObjID == 0 || f.CfgID == 0 || f.ContentID == 0 {
			t.Errorf("第 %d 朵描述字段丢了: %+v", i+1, f)
		}
	}
}

// 0x0376 增量通知:flowers(1) 里是同构的 BossNpcInfo,没有 ret_info。
func TestParseFlowerSeedNty(t *testing.T) {
	one := bossNpcMutation(bossNpcInfo(20135, 5, 3012, 2100044, 9279722995167909004, 1789243200, 0),
		MutationShiny, 0, 0)
	body := msg(nil, 1, msg(nil, 1, one))
	body = append(body, tsf4gMark...)

	got, ok := ParseFlowerSeedNty(body)
	if !ok {
		t.Fatal("增量通知应可用")
	}
	if len(got.Seeds) != 1 {
		t.Fatalf("增量花种数 = %d, 期望 1", len(got.Seeds))
	}
	f := got.Seeds[0]
	if f.ObjID != 9279722995167909004 || f.MutationType != MutationShiny || !f.HasMutation {
		t.Errorf("增量项 = %+v", f)
	}
	// 没有 flowers 那一支时不该当成空列表(否则会把整层清掉)
	if _, ok := ParseFlowerSeedNty(append([]byte{}, tsf4gMark...)); ok {
		t.Error("空通知不该 ok")
	}
}
