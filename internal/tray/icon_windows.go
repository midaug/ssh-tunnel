//go:build windows

package tray

import _ "embed"

//go:embed icon.ico
var iconBytes []byte

//go:embed icon_gray.ico
var iconGrayBytes []byte
