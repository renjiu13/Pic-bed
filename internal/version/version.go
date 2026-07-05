package version

import (
	"fmt"
	"runtime"
)

// 版本信息，通过 ldflags 注入
var (
	Version   = "dev"     // 版本号，如 v1.0.6
	BuildTime = "unknown" // 编译时间
	GitCommit = "unknown" // Git commit hash
)

// Info 返回完整的版本信息
func Info() string {
	return fmt.Sprintf("Pic-bed %s", Version)
}

// FullInfo 返回详细的版本信息
func FullInfo() string {
	return fmt.Sprintf(
		"Pic-bed %s\n  Build: %s\n  Commit: %s\n  Go: %s\n  OS/Arch: %s/%s",
		Version,
		BuildTime,
		GitCommit,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// Platform 返回当前平台标识，用于匹配 release 文件名
func Platform() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// 特殊处理 armv7
	if os == "linux" && arch == "arm" {
		return "linux-armv7"
	}

	return fmt.Sprintf("%s-%s", os, arch)
}

// BinaryName 返回对应平台的二进制文件名
func BinaryName() string {
	platform := Platform()
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("pic-bed-%s.exe", platform)
	}
	return fmt.Sprintf("pic-bed-%s", platform)
}
