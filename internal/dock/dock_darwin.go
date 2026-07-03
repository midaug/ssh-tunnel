//go:build darwin

package dock

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>

// setAccessory 隐藏 Dock 图标（菜单栏应用模式）
void setAccessory() {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}

// setRegular 恢复 Dock 图标（普通应用模式）
void setRegular() {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
}
*/
import "C"

// Hide 隐藏 Dock 图标
func Hide() {
	C.setAccessory()
}

// Show 恢复 Dock 图标
func Show() {
	C.setRegular()
}
