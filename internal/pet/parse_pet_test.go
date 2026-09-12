package pet

import (
	"encoding/hex"
	"strings"
	"testing"
)

// 2026-09-13 pcap 里连续三次改同一只宠物(gid 34052)伙伴标记的 0x0403 回包原始 AppBody:
// 先移除(PPMT_NONE)、再加回(PPMT_HOMELAND=4)、最后换样式(PPMT_DEFENCE=8)。
// 尾部 32 字节起是 AES 填充与 tsf4g 残留,解析须照常容忍。
var collectTagRspHex = []string{
	`0a1f0800221b0a1108271002180028007084 8a02 78d4b9b729 10c4de16 18b9be34` +
		`672b3d7d899c05 3425bf85383f6bea 74736634 6715`,
	`0a1f0800221b0a1108271002180428007084 8a02 78bee7b729 10c4de16 18b9be34` +
		`3bea9b572e54eaf9 95 8973 dad1 3d0c 74736634 6715`,
	`0a1f0800221b0a1108271002180828007084 8a02 78b799b829 10c4de16 18b9be34` +
		`bb6f4cca9613f246 a170cb3cec5815 74736634 6715`,
}

func TestParseCollectTagRsp(t *testing.T) {
	want := []int32{0, 4, 8} // 移除 / 家园标记 / 防守标记
	for i, h := range collectTagRspHex {
		body, err := hex.DecodeString(strings.ReplaceAll(h, " ", ""))
		if err != nil {
			t.Fatalf("第 %d 条 hex 无效: %v", i+1, err)
		}
		gid, mark, ok := ParseCollectTagRsp(body)
		if !ok {
			t.Fatalf("第 %d 条未解出伙伴标记变更", i+1)
		}
		if gid != 34052 || mark != want[i] {
			t.Errorf("第 %d 条 = (gid %d, mark %d), 期望 (34052, %d)", i+1, gid, mark, want[i])
		}
	}
}

// 不相关的回包(无 GT_PET_MARK 条目)不得误报,否则会把别的宠物标记改掉。
func TestParseCollectTagRspIgnoresOthers(t *testing.T) {
	if _, _, ok := ParseCollectTagRsp([]byte{0x0a, 0x02, 0x08, 0x00}); ok { // 只有 ret_code=0
		t.Error("空 goods_change_info 不应解出标记变更")
	}
	if _, _, ok := ParseCollectTagRsp(nil); ok {
		t.Error("空 body 不应解出标记变更")
	}
}
