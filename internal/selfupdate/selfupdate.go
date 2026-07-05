package selfupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pic-bed/pic-bed/internal/version"
)

const (
	githubAPI  = "https://api.github.com/repos/renjiu13/Pic-bed/releases/latest"
	githubDL   = "https://github.com/renjiu13/Pic-bed/releases/download"
)

// Release 表示 GitHub Release 信息
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

// Asset 表示 Release 中的资产文件
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckResult 表示版本检查结果
type CheckResult struct {
	HasUpdate   bool
	Latest      string
	Current     string
	DownloadURL string
}

// CheckUpdate 检查是否有新版本
func CheckUpdate() (*CheckResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	req, err := http.NewRequest("GET", githubAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Pic-bed-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回状态码: %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	current := version.Version
	latest := release.TagName

	// 查找对应平台的下载链接
	binaryName := version.BinaryName()
	downloadURL := ""
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	result := &CheckResult{
		HasUpdate:   isNewer(latest, current),
		Latest:      latest,
		Current:     current,
		DownloadURL: downloadURL,
	}

	return result, nil
}

// isNewer 比较两个版本号，判断 v2 是否比 v1 新
func isNewer(v1, v2 string) bool {
	// 去除前缀 v
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// 分割版本号
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// 确保都有 3 部分
	for len(parts1) < 3 {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < 3 {
		parts2 = append(parts2, "0")
	}

	// 逐位比较
	for i := 0; i < 3; i++ {
		n1 := parseVersionPart(parts1[i])
		n2 := parseVersionPart(parts2[i])
		if n1 > n2 {
			return true
		}
		if n1 < n2 {
			return false
		}
	}

	return false
}

// parseVersionPart 解析版本号的一部分为数字
func parseVersionPart(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// DoUpdate 执行更新
func DoUpdate(downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("下载链接为空")
	}

	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取当前程序路径失败: %w", err)
	}

	// 下载新版本
	fmt.Printf("正在下载新版本...\n")
	fmt.Printf("下载地址: %s\n", downloadURL)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载返回状态码: %d", resp.StatusCode)
	}

	// 创建临时文件
	tmpPath := exePath + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}

	// 写入文件
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 备份旧版本
	backupPath := exePath + ".bak"
	if err := os.Rename(exePath, backupPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("备份旧版本失败: %w", err)
	}

	// 替换为新版本
	if err := os.Rename(tmpPath, exePath); err != nil {
		// 恢复旧版本
		os.Rename(backupPath, exePath)
		os.Remove(tmpPath)
		return fmt.Errorf("替换文件失败: %w", err)
	}

	// Windows 下不需要设置权限
	if runtime.GOOS != "windows" {
		if err := os.Chmod(exePath, 0755); err != nil {
			fmt.Printf("警告: 设置执行权限失败: %v\n", err)
		}
	}

	// 删除备份
	os.Remove(backupPath)

	fmt.Println("✓ 更新完成！")
	fmt.Printf("请重新启动程序以使用新版本\n")

	return nil
}
