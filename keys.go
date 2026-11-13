package main

import "fmt"

// All IDs are stable configuration values; labels are only for display.
type key struct {
	ID       string
	Label    string
	VK       uint16
	Extended bool
	Mouse    bool
}

var keys = makeKeys()

func makeKeys() []key {
	k := []key{
		{ID: "mouse_x1", Label: "鼠标侧键 1（后退）", Mouse: true},
		{ID: "mouse_x2", Label: "鼠标侧键 2（前进）", Mouse: true},
		{ID: "mouse_middle", Label: "鼠标中键（滚轮按下）", Mouse: true},
		{ID: "mouse_left", Label: "鼠标左键", Mouse: true},
		{ID: "mouse_right", Label: "鼠标右键", Mouse: true},
	}
	for c := 'A'; c <= 'Z'; c++ {
		k = append(k, key{ID: string(c), Label: string(c), VK: uint16(c)})
	}
	for c := '0'; c <= '9'; c++ {
		k = append(k, key{ID: string(c), Label: string(c), VK: uint16(c)})
	}
	for n := 1; n <= 12; n++ {
		name := fmt.Sprintf("F%d", n)
		k = append(k, key{ID: name, Label: name, VK: uint16(0x6F + n)})
	}
	k = append(k, []key{
		{ID: "space", Label: "空格", VK: 0x20},
		{ID: "enter", Label: "Enter", VK: 0x0D},
		{ID: "tab", Label: "Tab", VK: 0x09},
		{ID: "escape", Label: "Esc", VK: 0x1B},
		{ID: "backspace", Label: "Backspace", VK: 0x08},
		{ID: "delete", Label: "Delete", VK: 0x2E, Extended: true},
		{ID: "insert", Label: "Insert", VK: 0x2D, Extended: true},
		{ID: "home", Label: "Home", VK: 0x24, Extended: true},
		{ID: "end", Label: "End", VK: 0x23, Extended: true},
		{ID: "page_up", Label: "Page Up", VK: 0x21, Extended: true},
		{ID: "page_down", Label: "Page Down", VK: 0x22, Extended: true},
		{ID: "up", Label: "↑ 上方向键", VK: 0x26, Extended: true},
		{ID: "down", Label: "↓ 下方向键", VK: 0x28, Extended: true},
		{ID: "left", Label: "← 左方向键", VK: 0x25, Extended: true},
		{ID: "right", Label: "→ 右方向键", VK: 0x27, Extended: true},
		{ID: "lctrl", Label: "左 Ctrl", VK: 0xA2},
		{ID: "rctrl", Label: "右 Ctrl", VK: 0xA3, Extended: true},
		{ID: "lshift", Label: "左 Shift", VK: 0xA0},
		{ID: "rshift", Label: "右 Shift", VK: 0xA1},
		{ID: "lalt", Label: "左 Alt", VK: 0xA4},
		{ID: "ralt", Label: "右 Alt", VK: 0xA5, Extended: true},
		{ID: "caps_lock", Label: "Caps Lock", VK: 0x14},
		{ID: "minus", Label: "- 减号", VK: 0xBD},
		{ID: "equals", Label: "= 等号", VK: 0xBB},
		{ID: "lbracket", Label: "[ 左方括号", VK: 0xDB},
		{ID: "rbracket", Label: "] 右方括号", VK: 0xDD},
		{ID: "backslash", Label: "\\ 反斜杠", VK: 0xDC},
		{ID: "semicolon", Label: "; 分号", VK: 0xBA},
		{ID: "quote", Label: "' 引号", VK: 0xDE},
		{ID: "comma", Label: ", 逗号", VK: 0xBC},
		{ID: "period", Label: ". 句号", VK: 0xBE},
		{ID: "slash", Label: "/ 斜杠", VK: 0xBF},
		{ID: "grave", Label: "` 反引号", VK: 0xC0},
	}...)
	for n := 0; n <= 9; n++ {
		k = append(k, key{ID: fmt.Sprintf("num%d", n), Label: fmt.Sprintf("小键盘 %d", n), VK: uint16(0x60 + n)})
	}
	k = append(k, []key{
		{ID: "num_add", Label: "小键盘 +", VK: 0x6B},
		{ID: "num_subtract", Label: "小键盘 -", VK: 0x6D},
		{ID: "num_multiply", Label: "小键盘 *", VK: 0x6A},
		{ID: "num_divide", Label: "小键盘 /", VK: 0x6F, Extended: true},
		{ID: "num_decimal", Label: "小键盘 .", VK: 0x6E},
		{ID: "num_enter", Label: "小键盘 Enter", VK: 0x0D, Extended: true},
		{ID: "volume_up", Label: "音量 +", VK: 0xAF, Extended: true},
		{ID: "volume_down", Label: "音量 -", VK: 0xAE, Extended: true},
		{ID: "volume_mute", Label: "静音", VK: 0xAD, Extended: true},
		{ID: "media_play", Label: "播放 / 暂停", VK: 0xB3, Extended: true},
		{ID: "media_next", Label: "下一首", VK: 0xB0, Extended: true},
		{ID: "media_prev", Label: "上一首", VK: 0xB1, Extended: true},
	}...)
	return k
}

func findKey(id string) (key, bool) {
	for _, k := range keys {
		if k.ID == id {
			return k, true
		}
	}
	return key{}, false
}

func keyLabel(id string) string {
	k, _ := findKey(id)
	return k.Label
}
