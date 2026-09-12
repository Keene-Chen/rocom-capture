package pipeline

import (
	"testing"

	"github.com/whoisnian/rocom-capture/internal/gamedata"
	"github.com/whoisnian/rocom-capture/internal/scene"
	"github.com/whoisnian/rocom-capture/internal/store"
)

// 异色与炫彩可以同时成立(游戏里就有这种精灵):State 记异色、Glass 记炫彩,两样都不能丢。
// 用例里的色号是编的,查不到款式会退成「炫彩」——正好验证「Glass 非空 ⇔ 炫彩」这条。
func TestFlowerRowMutationBits(t *testing.T) {
	db, err := gamedata.Load()
	if err != nil {
		t.Skipf("名称库未生成(先跑 make gamedata): %v", err)
	}
	p := &Pipeline{db: db}
	base := scene.FlowerSeed{ObjID: 1, CfgID: 20135, HasMutation: true}
	cases := []struct {
		name       string
		mutation   int32
		glassType  int32
		wantState  int
		wantGlassy bool
	}{
		{"普通", 0, 0, store.FlowerPlain, false},
		{"炫彩", scene.MutationGlass, 1, store.FlowerGlassy, true},
		{"异色", scene.MutationShiny, 0, store.FlowerShiny, false},
		{"异色又炫彩", scene.MutationShiny | scene.MutationGlass, 2, store.FlowerShiny, true},
	}
	for _, c := range cases {
		f := base
		f.MutationType, f.GlassType, f.GlassValue = c.mutation, c.glassType, 1
		r := p.flowerRow(f)
		if r.State != c.wantState || (r.Glass != "") != c.wantGlassy {
			t.Errorf("%s: state=%d glass=%q, 期望 state=%d 炫彩=%v",
				c.name, r.State, r.Glass, c.wantState, c.wantGlassy)
		}
	}
	// 没带 glass_info 那一支:什么都不知道,留未检测等 0x0338 兜底
	bare := scene.FlowerSeed{ObjID: 1, CfgID: 20135}
	if r := p.flowerRow(bare); r.State != store.FlowerUndetected || r.Glass != "" {
		t.Errorf("不带变异那一支应留未检测: %+v", r)
	}
}
