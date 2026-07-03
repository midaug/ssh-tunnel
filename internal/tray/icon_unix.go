//go:build !windows

package tray

import _ "embed"

//go:embed icon.png
var iconBytes []byte

//go:embed icon_gray.png
var iconGrayBytes []byte
