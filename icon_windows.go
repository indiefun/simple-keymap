//go:build windows

package main

// Icon resource 1 is embedded by rsrc_windows_amd64.syso. LR_SHARED keeps the
// module-owned icon handles valid for the lifetime of the process.
func loadBrandIcon(instance uintptr, size int) uintptr {
	icon, _, _ := loadImage.Call(instance, 1, 1, uintptr(size), uintptr(size), 0x8000)
	if icon == 0 {
		icon, _, _ = loadIcon.Call(0, 32512)
	}
	return icon
}
