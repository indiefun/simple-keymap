//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	gdi32                  = syscall.NewLazyDLL("gdi32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	comctl32               = syscall.NewLazyDLL("comctl32.dll")
	uxtheme                = syscall.NewLazyDLL("uxtheme.dll")
	shellExecute           = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	registerClassEx        = user32.NewProc("RegisterClassExW")
	createWindowEx         = user32.NewProc("CreateWindowExW")
	defWindowProc          = user32.NewProc("DefWindowProcW")
	getMessage             = user32.NewProc("GetMessageW")
	translateMessage       = user32.NewProc("TranslateMessage")
	dispatchMessage        = user32.NewProc("DispatchMessageW")
	isDialogMessage        = user32.NewProc("IsDialogMessageW")
	postQuitMessage        = user32.NewProc("PostQuitMessage")
	postMessage            = user32.NewProc("PostMessageW")
	sendMessage            = user32.NewProc("SendMessageW")
	showWindow             = user32.NewProc("ShowWindow")
	updateWindow           = user32.NewProc("UpdateWindow")
	destroyWindow          = user32.NewProc("DestroyWindow")
	setWindowText          = user32.NewProc("SetWindowTextW")
	setForeground          = user32.NewProc("SetForegroundWindow")
	findWindow             = user32.NewProc("FindWindowW")
	setFocus               = user32.NewProc("SetFocus")
	getForeground          = user32.NewProc("GetForegroundWindow")
	getWindowProcess       = user32.NewProc("GetWindowThreadProcessId")
	setDPIAware            = user32.NewProc("SetProcessDPIAware")
	getDPIForSystem        = user32.NewProc("GetDpiForSystem")
	moveWindow             = user32.NewProc("MoveWindow")
	getClientRect          = user32.NewProc("GetClientRect")
	getWindowRect          = user32.NewProc("GetWindowRect")
	getCursorPos           = user32.NewProc("GetCursorPos")
	getFocus               = user32.NewProc("GetFocus")
	getWindowText          = user32.NewProc("GetWindowTextW")
	getWindowTextLength    = user32.NewProc("GetWindowTextLengthW")
	openClipboard          = user32.NewProc("OpenClipboard")
	closeClipboard         = user32.NewProc("CloseClipboard")
	emptyClipboard         = user32.NewProc("EmptyClipboard")
	setClipboardData       = user32.NewProc("SetClipboardData")
	windowFromPoint        = user32.NewProc("WindowFromPoint")
	adjustWindowRect       = user32.NewProc("AdjustWindowRectEx")
	getSystemMetrics       = user32.NewProc("GetSystemMetrics")
	loadCursor             = user32.NewProc("LoadCursorW")
	loadIcon               = user32.NewProc("LoadIconW")
	loadImage              = user32.NewProc("LoadImageW")
	drawIconEx             = user32.NewProc("DrawIconEx")
	messageBox             = user32.NewProc("MessageBoxW")
	beginPaint             = user32.NewProc("BeginPaint")
	endPaint               = user32.NewProc("EndPaint")
	invalidateRect         = user32.NewProc("InvalidateRect")
	fillRect               = user32.NewProc("FillRect")
	drawText               = user32.NewProc("DrawTextW")
	enableWindow           = user32.NewProc("EnableWindow")
	setTimer               = user32.NewProc("SetTimer")
	killTimer              = user32.NewProc("KillTimer")
	registerHotKey         = user32.NewProc("RegisterHotKey")
	unregisterHotKey       = user32.NewProc("UnregisterHotKey")
	setWindowsHookEx       = user32.NewProc("SetWindowsHookExW")
	unhookWindowsHook      = user32.NewProc("UnhookWindowsHookEx")
	callNextHook           = user32.NewProc("CallNextHookEx")
	sendInput              = user32.NewProc("SendInput")
	mapVirtualKey          = user32.NewProc("MapVirtualKeyW")
	getAsyncKeyState       = user32.NewProc("GetAsyncKeyState")
	getModuleHandle        = kernel32.NewProc("GetModuleHandleW")
	createMutex            = kernel32.NewProc("CreateMutexW")
	closeHandle            = kernel32.NewProc("CloseHandle")
	getCurrentProcessID    = kernel32.NewProc("GetCurrentProcessId")
	globalAlloc            = kernel32.NewProc("GlobalAlloc")
	globalLock             = kernel32.NewProc("GlobalLock")
	globalUnlock           = kernel32.NewProc("GlobalUnlock")
	globalFree             = kernel32.NewProc("GlobalFree")
	createSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	createPen              = gdi32.NewProc("CreatePen")
	selectObject           = gdi32.NewProc("SelectObject")
	deleteObject           = gdi32.NewProc("DeleteObject")
	getStockObject         = gdi32.NewProc("GetStockObject")
	roundRect              = gdi32.NewProc("RoundRect")
	moveTo                 = gdi32.NewProc("MoveToEx")
	lineTo                 = gdi32.NewProc("LineTo")
	setTextColor           = gdi32.NewProc("SetTextColor")
	setBkMode              = gdi32.NewProc("SetBkMode")
	createFont             = gdi32.NewProc("CreateFontW")
	createCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	bitBlt                 = gdi32.NewProc("BitBlt")
	deleteDC               = gdi32.NewProc("DeleteDC")
	initCommonControls     = comctl32.NewProc("InitCommonControlsEx")
	imageListCreate        = comctl32.NewProc("ImageList_Create")
	imageListDestroy       = comctl32.NewProc("ImageList_Destroy")
	setWindowTheme         = uxtheme.NewProc("SetWindowTheme")
	setWindowSubclass      = comctl32.NewProc("SetWindowSubclass")
	defSubclassProc        = comctl32.NewProc("DefSubclassProc")
	createPopupMenu        = user32.NewProc("CreatePopupMenu")
	appendMenu             = user32.NewProc("AppendMenuW")
	trackPopupMenu         = user32.NewProc("TrackPopupMenu")
	destroyMenu            = user32.NewProc("DestroyMenu")
	isChild                = user32.NewProc("IsChild")
)

const (
	wmDestroy      = 0x0002
	wmSize         = 0x0005
	wmActivateApp  = 0x001C
	wmPaint        = 0x000F
	wmClose        = 0x0010
	wmEraseBkgnd   = 0x0014
	wmDrawItem     = 0x002B
	wmNotify       = 0x004E
	wmCommand      = 0x0111
	wmTimer        = 0x0113
	wmHotkey       = 0x0312
	wmCaptured     = 0x8001
	wmInputError   = 0x8002
	wmSendOutput   = 0x8003
	wsChild        = 0x40000000
	wsVisible      = 0x10000000
	wsTabstop      = 0x00010000
	wsVscroll      = 0x00200000
	wsClipChildren = 0x02000000
	wmSetFont      = 0x0030
	cbAddString    = 0x0143
	cbGetCurSel    = 0x0147
	cbSetCurSel    = 0x014E
	lvmFirst       = 0x1000
	appClass       = "SimpleKeymapWindow"
	appTitle       = "轻映 · 按键映射"
)

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type winMessage struct {
	Hwnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	Private        uint32
}
type windowClass struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClassExtra, WndExtra               int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSmall                          uintptr
}
type paintStruct struct {
	DC                 uintptr
	Erase              int32
	Paint              rect
	Restore, IncUpdate int32
	Reserved           [32]byte
}
type drawItemStruct struct {
	Type, ID, ItemID, Action, State uint32
	Hwnd, DC                        uintptr
	Rect                            rect
	Data                            uintptr
}
type notifyHeader struct {
	Hwnd, ID uintptr
	Code     uint32
}
type listColumn struct {
	Mask                                                               uint32
	Format, Width                                                      int32
	Text                                                               *uint16
	TextMax, SubItem, Image, Order, MinWidth, DefaultWidth, IdealWidth int32
}
type listItem struct {
	Mask                         uint32
	Item, SubItem                int32
	State, StateMask             uint32
	Text                         *uint16
	TextMax, Image               int32
	Param                        uintptr
	Indent, GroupID              int32
	Columns                      uint32
	ColumnIndices, ColumnFormats uintptr
	Group                        int32
}

type customDraw struct {
	Header notifyHeader
	Stage  uint32
	DC     uintptr
	Rect   rect
	Item   uintptr
	State  uint32
	Param  uintptr
}

type listCustomDraw struct {
	Draw                                   customDraw
	TextColor, TextBackground              uint32
	SubItem                                int32
	ItemType, FaceColor                    uint32
	IconEffect, IconPhase, PartID, StateID int32
	TextRect                               rect
	Align                                  uint32
}

// LPARAM is a union: most messages carry integers, but WM_DRAWITEM and
// WM_NOTIFY carry pointers to memory owned by Windows for the callback's
// duration. Read that union as a pointer only in those two message branches.
func messageData[T any](param *uintptr) *T {
	return *(**T)(unsafe.Pointer(param))
}

func utf16(s string) *uint16     { return syscall.StringToUTF16Ptr(s) }
func rgb(r, g, b uint32) uintptr { return uintptr(r | g<<8 | b<<16) }
func intArg(n int) uintptr       { return uintptr(n) }
func send(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
	r, _, _ := sendMessage.Call(hwnd, uintptr(msg), w, l)
	return r
}
func setText(hwnd uintptr, text string) {
	setWindowText.Call(hwnd, uintptr(unsafe.Pointer(utf16(text))))
}
func windowText(hwnd uintptr) string {
	n, _, _ := getWindowTextLength.Call(hwnd)
	buffer := make([]uint16, n+1)
	getWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer)
}
func showError(hwnd uintptr, text string) {
	messageBox.Call(hwnd, uintptr(unsafe.Pointer(utf16(text))), uintptr(unsafe.Pointer(utf16(appTitle))), 0x10)
}

func keyboardKey(vk uint32, scan uint32, flags uint32) string {
	// With Num Lock off, the numeric keypad reports navigation virtual keys.
	// Its non-extended scan codes still identify the same physical keys.
	if flags&1 == 0 {
		if id := map[uint32]string{0x52: "num0", 0x4F: "num1", 0x50: "num2", 0x51: "num3", 0x4B: "num4", 0x4C: "num5", 0x4D: "num6", 0x47: "num7", 0x48: "num8", 0x49: "num9", 0x53: "num_decimal"}[scan]; id != "" {
			return id
		}
	}
	if vk == 0x10 { // Generic shift values can appear in synthesized input.
		if scan == 0x36 {
			vk = 0xA1
		} else {
			vk = 0xA0
		}
	}
	if vk == 0x11 {
		if flags&1 != 0 {
			vk = 0xA3
		} else {
			vk = 0xA2
		}
	}
	if vk == 0x12 {
		if flags&1 != 0 {
			vk = 0xA5
		} else {
			vk = 0xA4
		}
	}
	if vk == 0x0D && flags&1 != 0 {
		return "num_enter"
	}
	for _, k := range keys {
		if !k.Mouse && uint32(k.VK) == vk {
			return k.ID
		}
	}
	return ""
}
