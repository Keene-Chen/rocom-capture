package store

import (
	"path/filepath"
	"testing"
)

// 花种列表每次都是全量:列表里没有的行必须删掉(整朵重投后 obj_id 会全换,旧行留着就是幽灵点),
// 列表里有的行只覆盖描述列——已检测出的炫彩结果不能被一次列表刷新抹掉。
func TestReplaceFlowers(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.db.Close()
	sc := st.For("UID:1")

	old := []FlowerRow{
		{ObjID: 9279722995167894922, CfgID: 700002, Star: 7, PetBase: 3007, ContentID: 2606405, EndTS: 1787860799},
		{ObjID: 9279722995167901301, CfgID: 20143, Star: 5, PetBase: 3439, ContentID: 140309, EndTS: 1787688000},
	}
	if err := sc.ReplaceFlowers(old); err != nil {
		t.Fatal(err)
	}
	// 检测出一朵炫彩
	det := old[0]
	det.State, det.Glass = FlowerGlassy, "亮X暗 - 浅绿青·四角星"
	if err := sc.SetFlowerDetected(det); err != nil {
		t.Fatal(err)
	}
	// 同一批再来一次:检测结果必须保住
	if err := sc.ReplaceFlowers(old); err != nil {
		t.Fatal(err)
	}
	got := sc.Flowers(0)
	if len(got) != 2 {
		t.Fatalf("重复下发同一列表后行数 = %d, 期望 2", len(got))
	}
	for _, r := range got {
		if r.ObjID == det.ObjID && (r.State != FlowerGlassy || r.Glass == "") {
			t.Errorf("列表刷新不该抹掉已检测出的炫彩: %+v", r)
		}
	}

	// 整朵重投:obj_id 全换,旧行必须消失(实测 2026-08-26 零点前后 23 朵 obj_id 全变)
	fresh := []FlowerRow{
		{ObjID: 9281973836197157848, CfgID: 700002, Star: 7, PetBase: 3007, ContentID: 2606405, EndTS: 1787860799},
		{ObjID: 9281973836197161409, CfgID: 20143, Star: 5, PetBase: 3335, ContentID: 140309, EndTS: 1787688000},
	}
	if err := sc.ReplaceFlowers(fresh); err != nil {
		t.Fatal(err)
	}
	got = sc.Flowers(0)
	if len(got) != 2 {
		t.Fatalf("换代后行数 = %d, 期望 2(旧的两行应被删掉)", len(got))
	}
	for _, r := range got {
		if r.ObjID != fresh[0].ObjID && r.ObjID != fresh[1].ObjID {
			t.Errorf("旧行没被删掉: %+v", r)
		}
		if r.State != FlowerUndetected {
			t.Errorf("新一朵花应是未检测: %+v", r)
		}
	}
}

// 列表自带变异结果(2026-09-10 大版本起)时以列表为准:能把炫彩改成普通(精灵重投后那朵花
// 换了只普通个体),也能标出异色;而列表没带结果(state=0)时绝不能抹掉已检测出的炫彩。
// 增量通知(0x0376)只增改不删:缺席的花照旧开着。
func TestFlowerMutationFromList(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.db.Close()
	sc := st.For("UID:1")

	rows := []FlowerRow{
		{ObjID: 9279722995167909004, CfgID: 20135, Star: 5, PetBase: 3012, ContentID: 2100044, EndTS: 1789243200,
			State: FlowerShiny, Glass: "暗夜拾光"},
		{ObjID: 9279722995167909003, CfgID: 20143, Star: 5, PetBase: 3174, ContentID: 2100034, EndTS: 1789243200,
			State: FlowerGlassy, Glass: "亮X暗 - 浅绿青·四角星"},
	}
	if err := sc.ReplaceFlowers(rows); err != nil {
		t.Fatal(err)
	}
	byID := func() map[uint64]FlowerRow {
		m := map[uint64]FlowerRow{}
		for _, r := range sc.Flowers(0) {
			m[r.ObjID] = r
		}
		return m
	}
	got := byID()
	if got[rows[0].ObjID].State != FlowerShiny || got[rows[1].ObjID].State != FlowerGlassy {
		t.Fatalf("列表带来的变异结果没入库: %+v", got)
	}
	// 既异色又炫彩:State 记异色,炫彩由 Glass 表达,两样都得留住(最稀罕的一种,不能互相盖掉)
	if r := got[rows[0].ObjID]; r.State != FlowerShiny || r.Glass != "暗夜拾光" {
		t.Errorf("既异色又炫彩的那朵丢了信息: %+v", r)
	}

	// 同一朵花里的精灵重投成普通个体:列表说普通就得是普通
	rows[1].State, rows[1].Glass = FlowerPlain, ""
	if err := sc.ReplaceFlowers(rows); err != nil {
		t.Fatal(err)
	}
	if r := byID()[rows[1].ObjID]; r.State != FlowerPlain || r.Glass != "" {
		t.Errorf("列表说普通却没改过来: %+v", r)
	}

	// 某次下发不带变异那一支(state=0):已有结果必须保住
	bare := []FlowerRow{{ObjID: rows[0].ObjID, CfgID: 20135, Star: 5, PetBase: 3012, ContentID: 2100044, EndTS: 1789243200}}
	if err := sc.ReplaceFlowers(bare); err != nil {
		t.Fatal(err)
	}
	if r := byID()[rows[0].ObjID]; r.State != FlowerShiny || r.Glass == "" {
		t.Errorf("不带变异的列表抹掉了已有结果: %+v", r)
	}

	// 增量通知:只动通知里那朵,不碰别的、也不删缺席的
	rows[1].State, rows[1].Glass = FlowerGlassy, "亮X暗 - 浅绿青·四角星" // 复原成炫彩,作为「不该被动到」的对照
	if err := sc.ReplaceFlowers(rows); err != nil {
		t.Fatal(err)
	}
	nty := []FlowerRow{{ObjID: rows[0].ObjID, CfgID: 20135, Star: 5, PetBase: 3012, ContentID: 2100044,
		EndTS: 1789243200, State: FlowerPlain}}
	if err := sc.UpsertFlowers(nty); err != nil {
		t.Fatal(err)
	}
	got = byID()
	if len(got) != 2 {
		t.Fatalf("增量通知后行数 = %d, 期望 2(缺席的花不该被删)", len(got))
	}
	if got[rows[0].ObjID].State != FlowerPlain {
		t.Errorf("增量通知没更新那一朵: %+v", got[rows[0].ObjID])
	}
	if got[rows[1].ObjID].State != FlowerGlassy {
		t.Errorf("增量通知动了别的花: %+v", got[rows[1].ObjID])
	}
}
