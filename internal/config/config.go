package config

import (
	"encoding/json"
	"log"
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

	// 📦 固定大小压缩：二分查找质量，压缩到目标大小（不缩放）
	EnableFixedSizeCompression bool `json:"enable_fixed_size_compression"` // 📦 固定大小压缩开关
	TargetFileSizeKB           int  `json:"target_file_size_kb"`           // 固定压缩目标大小（KB）
	CompressionQualityStart    int  `json:"compression_quality_start"`    // 固定压缩起始（最大）质量 1-100
	CompressQueueSize          int  `json:"compress_queue_size"`          // 异步压缩队列大小（0=同步）

	// 🛡️ 内存守护：弱设备防 OOM
	EnableMemoryGuard    bool `json:"enable_memory_guard"`     // 启用内存守护
	MemoryLimitMB        int  `json:"memory_limit_mb"`         // 内存上限（MB），超过则重启
	MemoryCheckInterval  int  `json:"memory_check_interval"`   // 内存检查间隔（秒）

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

	// 📦 固定大小压缩：默认关闭
	EnableFixedSizeCompression: false, // 压缩到目标大小，默认关闭
	TargetFileSizeKB:           500,   // 目标大小 500KB（适合博客图片）
	CompressionQualityStart:    90,    // 起始（最大）质量 90
	CompressQueueSize:          100,   // 异步压缩队列大小（0=同步阻塞，>0=异步不阻塞上传）

	// 🛡️ 内存守护：弱设备防 OOM，默认关闭
	EnableMemoryGuard:    false, // 内存守护，默认关闭
	MemoryLimitMB:        200,   // 内存上限 200MB（玩客云等弱设备推荐）
	MemoryCheckInterval:  30,    // 每 30 秒检查一次

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
	normalizeConfig(&cfg)
	mu.Lock()
	defer mu.Unlock()
	globalCfg = cfg
	return nil
}

// normalizeConfig 规整配置：修正非法值，保证运行期配置始终合法。
// 固定压缩与 WebP 转换可同时开启，协同工作（非互斥）。
func normalizeConfig(cfg *Config) {
	// ① 协同说明：固定压缩与 WebP 转换可同时开启——
	//    固定压缩优先处理（精确控大小，删除原图）；
	//    固定压缩跳过的格式（gif/webp/小图）由 WebP 转换兜底。
	//    keep_original_after_webp 仅在 WebP 转换兜底路径生效。
	if cfg.EnableFixedSizeCompression && cfg.EnableWebPConvert {
		log.Printf("[config] 固定压缩与 WebP 转换同时启用：固定压缩优先，WebP 转换作为兜底")
	}

	// ② 基础字段兜底，避免 0/负数导致除零、死循环或逻辑失效
	if cfg.Port <= 0 || cfg.Port > 65535 {
		cfg.Port = 8080
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = "./data"
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 10
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}

	// ③ 压缩相关参数兜底
	if cfg.TargetFileSizeKB <= 0 {
		cfg.TargetFileSizeKB = 500
	}
	if cfg.CompressionQualityStart < 1 || cfg.CompressionQualityStart > 100 {
		cfg.CompressionQualityStart = 90
	}
	if cfg.WebPQuality < 1 || cfg.WebPQuality > 100 {
		cfg.WebPQuality = 80
	}
	if cfg.CompressQueueSize < 0 {
		cfg.CompressQueueSize = 100
	}

	// ④ 自动清理时长兜底
	if cfg.AutoCleanHours <= 0 {
		cfg.AutoCleanHours = 720
	}

	// ⑤ 内存守护参数兜底
	if cfg.MemoryLimitMB <= 0 {
		cfg.MemoryLimitMB = 200
	}
	if cfg.MemoryCheckInterval <= 0 {
		cfg.MemoryCheckInterval = 30
	}

	// ⑥ 允许类型兜底
	if len(cfg.AllowedTypes) == 0 {
		cfg.AllowedTypes = []string{"jpg", "jpeg", "png", "gif", "webp"}
	}
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