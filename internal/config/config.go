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
	MaxSize    int    `json:"max_size"` // 单位：MB
	APIKey     string `json:"api_key"`  // Bearer Token，空则关闭鉴权
	Timeout    int    `json:"timeout"`  // 请求超时时间，秒

	// 功能开关 - 默认全部关闭
	EnableLog         bool    `json:"enable_log"`          // 📝 操作日志：记录上传/删除/访问/错误
	EnableDelete      bool    `json:"enable_delete"`       // 🗑️ 删除接口：DELETE /img/{path} 删除图片
	EnableAutoClean   bool    `json:"enable_auto_clean"`   // 🧹 自动清理：定期删除超过N小时的文件
	KeepOriginalName  bool    `json:"keep_original_name"`  // 📄 原始文件名：保留上传文件名+随机后缀
	EnableWebPConvert bool    `json:"enable_webp_convert"` // 是否启用 WebP 转换
	WebPQuality       float32 `json:"webp_quality"`        // WebP 质量 1-100

	// 功能详细配置
	AllowedTypes   []string `json:"allowed_types"`    // 允许的文件类型白名单
	AutoCleanHours int      `json:"auto_clean_hours"` // 自动清理：超过多少小时的文件被删除
	LogFile        string   `json:"log_file"`         // 日志文件路径
	HomeAvatarURL  string   `json:"home_avatar_url"`  // 首页头像图片URL（支持本地路径或网络图片）
}

const configFileName = "config.json"

var (
	globalCfg Config
	mu        sync.RWMutex
	once      sync.Once
)

// 默认配置 - 所有功能开关默认关闭，用户按需开启
var defaultConfig = Config{
	Port:       8080,     // 服务监听端口
	StorageDir: "./data", // 图片存储目录
	MaxSize:    10,       // 单文件最大大小（MB）
	APIKey:     "",       // Bearer Token鉴权密钥，空则关闭鉴权
	Timeout:    30,       // 请求超时时间（秒）

	// 功能开关 - 默认全部关闭，按需开启
	EnableLog:         false, // 📝 操作日志：记录上传/删除/访问/错误
	EnableDelete:      false, // 🗑️ 删除接口：DELETE /img/{path} 删除图片
	EnableAutoClean:   false, // 🧹 自动清理：定期删除超过N小时的文件
	KeepOriginalName:  false, // 📄 原始文件名：保留上传文件名+随机后缀
	EnableWebPConvert: false, // WebP 转换：默认关闭
	WebPQuality:       80,    // WebP 质量：默认 80

	// 功能详细配置
	AllowedTypes:   []string{"jpg", "jpeg", "png", "gif", "webp"}, // 允许的文件类型白名单
	AutoCleanHours: 720,                                           // 自动清理：超过多少小时的文件被删除（默认30天=720小时）
	LogFile:        "./logs/app.log",                              // 日志文件路径
	HomeAvatarURL:  "",                                            // 首页头像图片URL（支持本地路径或网络图片，空则使用默认emoji）
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

// Reload 重新加载配置文件。
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

// Get 获取全局配置
func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return globalCfg
}
