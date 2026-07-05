package config

import (
	"encoding/json"
	"os"
	"sync"
)

// Config 全局配置结构体
type Config struct {
	Port       int    `json:"port"`
	StorageDir string `json:"storage_dir"`
	MaxSize    int    `json:"max_size"`    // 单位：MB
	APIKey     string `json:"api_key"`     // Bearer Token，用于鉴权
	Timeout    int    `json:"timeout"`     // 请求超时时间，秒

	// 功能开关 - 对应各功能模块
	EnableLog         bool `json:"enable_log"`            // 📝 日志功能：上传/删除/访问等
	EnableDelete      bool `json:"enable_delete"`         // 🗑️ 删除功能：DELETE /img/{path} 删除文件
	EnableAutoClean   bool `json:"enable_auto_clean"`     // 🧹 自动清理：定期删除超过指定时长的图片
	KeepOriginalName  bool `json:"keep_original_name"`    // 🔤 保留原始文件名：否则使用随机文件名
	EnableWebPConvert bool `json:"enable_webp_convert"`   // 自动转换 WebP 格式
	WebPQuality       float32 `json:"webp_quality"`       // WebP 质量 1-100
	KeepOriginalAfterWebP bool `json:"keep_original_after_webp"` // ✨ WebP 转换后是否保留原图

	// 安全与访问控制
	AllowedTypes   []string `json:"allowed_types"`    // 允许的文件类型列表
	AutoCleanHours int      `json:"auto_clean_hours"` // 自动清理：删除超过指定小时的图片
	LogFile        string   `json:"log_file"`         // 日志文件路径
	HomeAvatarURL  string   `json:"home_avatar_url"`  // 首页头像URL，为空则显示默认emoji
}

const configFileName = "config.json"

var (
	globalCfg Config
	mu        sync.RWMutex
	once      sync.Once
)

// 默认配置 - 首次运行时自动生成默认配置文件，方便快速上手
var defaultConfig = Config{
	Port:       8080,        // 默认监听端口
	StorageDir: "./data",    // 图片存储目录
	MaxSize:    10,          // 单文件最大大小，MB
	APIKey:     "",          // Bearer Token鉴权密钥

	// 功能开关 - 默认全部关闭，按需开启
	EnableLog:         false, // 📝 日志功能：上传/删除/访问等
	EnableDelete:      false, // 🗑️ 删除功能：DELETE /img/{path} 删除文件
	EnableAutoClean:   false, // 🧹 自动清理：定期删除超过指定时长的图片
	KeepOriginalName:  false, // 🔤 保留原始文件名：否则使用随机文件名
	EnableWebPConvert: false, // WebP 格式转换，默认关闭
	WebPQuality:       80,    // WebP 质量，默认 80
	KeepOriginalAfterWebP: false, // ✨ WebP 转换后是否保留原图，默认不保留（节省空间）

	// 安全与访问控制
	AllowedTypes:   []string{"jpg", "jpeg", "png", "gif", "webp"}, // 允许的文件类型
	AutoCleanHours: 720,                                           // 自动清理：删除超过指定小时的图片（默认30天=720小时）
	LogFile:        "./logs/app.log",                              // 日志文件路径
	HomeAvatarURL:  "",                                            // 首页头像URL，为空则显示默认emoji
}

// InitConfig 初始化配置
func InitConfig() error {
	var err error
	once.Do(func() {
		err = loadConfigFromFile()
	})
	if err != nil {
		return err
	}
	return Reload()
}

// Reload 重新加载配置文件，支持热更新
func Reload() error {
	cfg, err := readConfigFromFile()
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	globalCfg = cfg
	return nil
}

func loadConfigFromFile() error {
	if _, err := os.Stat(configFileName); os.IsNotExist(err) {
		data, marshalErr := json.MarshalIndent(defaultConfig, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		if writeErr := os.WriteFile(configFileName, data, 0644); writeErr != nil {
			return writeErr
		}
	}
	return Reload()
}

func readConfigFromFile() (Config, error) {
	var cfg Config
	if _, statErr := os.Stat(configFileName); os.IsNotExist(statErr) {
		cfg = defaultConfig
		return cfg, nil
	}
	data, err := os.ReadFile(configFileName)
	if err != nil {
		return Config{}, err
	}
	if err = json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Get 获取当前配置
func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return globalCfg
}