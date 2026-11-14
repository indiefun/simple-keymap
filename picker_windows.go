//go:build windows

package main

func keyCaption(id string) string {
	switch id {
	case "mouse_x1":
		return "鼠标侧键 1"
	case "mouse_x2":
		return "鼠标侧键 2"
	case "mouse_middle":
		return "鼠标中键"
	}
	return keyLabel(id)
}

func keyDetail(id string) string {
	switch id {
	case "mouse_x1":
		return "后退按钮"
	case "mouse_x2":
		return "前进按钮"
	case "mouse_middle":
		return "按下滚轮"
	case "mouse_left", "mouse_right":
		return "鼠标按键"
	}
	return "键盘按键"
}
