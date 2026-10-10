package config

import (
	"os"
	"strconv"
	"strings"
)

// Config конфигурация приложения
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Logger   LoggerConfig   `json:"logger"`
	Redis    RedisConfig    `json:"redis"`
	Media    MediaConfig    `json:"media"`
	Kafka    KafkaConfig    `json:"kafka"`
	Telegram TelegramConfig `json:"telegram"`
	Auth     AuthConfig     `json:"auth"`
	Spam     SpamConfig     `json:"spam"`
}

// ServerConfig конфигурация HTTP сервера
type ServerConfig struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	ReadTimeout  int    `json:"read_timeout"`
	WriteTimeout int    `json:"write_timeout"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
	SSLMode  string `json:"ssl_mode"`
}

// LoggerConfig конфигурация логирования
type LoggerConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	File   string `json:"file"`
}

// RedisConfig конфиг редиса
type RedisConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// MediaConfig конфигурация хранения медиа-файлов заказов (фото/видео)
type MediaConfig struct {
	// Driver выбирает бэкенд хранения: "local" или "s3"
	Driver        string        `json:"driver"`
	MaxFileSizeMB int64         `json:"max_file_size_mb"`
	Local         LocalMedia    `json:"local"`
	S3            S3MediaConfig `json:"s3"`
}

// LocalMedia настройки локального файлового хранилища
type LocalMedia struct {
	Path string `json:"path"`
	// PublicBaseURL префикс, по которому файлы отдаются наружу (см. GET /media/{path...}).
	// По умолчанию "/media" — подходит для локальной разработки без доп. настройки.
	PublicBaseURL string `json:"public_base_url"`
}

// S3MediaConfig настройки S3-совместимого хранилища
type S3MediaConfig struct {
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	// PublicBaseURL публичный домен для отдачи файлов (например, CDN перед бакетом).
	// Если не задан, строится URL по умолчанию для AWS S3 или Endpoint.
	PublicBaseURL string `json:"public_base_url"`
}

// KafkaConfig конфигурация Kafka
type KafkaConfig struct {
	// Brokers список адресов брокеров Kafka (host:port)
	Brokers []string `json:"brokers"`
	// GroupID consumer group ID (используется сервисами-читателями, например telegram-bot)
	GroupID string `json:"group_id"`
	Topics  Topics `json:"topics"`
}

// Topics топики Kafka, используемые в системе
type Topics struct {
	// Orders топик событий по заказам (создание, смена статуса)
	Orders string `json:"orders"`
}

// TelegramConfig конфигурация telegram-бота
type TelegramConfig struct {
	// BotToken секретный токен бота, выданный BotFather. Список пользователей бота
	// (кто получает уведомления и имеет доступ) хранится в БД, в таблице telegram_users.
	BotToken string `json:"-"`
}

// AuthConfig конфигурация JWT-аутентификации админ-панели
type AuthConfig struct {
	// JWTSecret секрет для подписи токенов. Обязателен — приложение не должно запускаться без него.
	JWTSecret string `json:"-"`
	// JWTTTLMinutes время жизни токена в минутах
	JWTTTLMinutes int `json:"jwt_ttl_minutes"`
}

// SpamConfig конфигурация защиты публичных форм (заявки, отзывы) от спама
type SpamConfig struct {
	// TrustedProxies адреса или подсети (CIDR) прокси перед приложением (nginx), которым
	// можно верить в заголовке X-Forwarded-For. Запрос не от доверенного прокси —
	// IP клиента берется из самого соединения, заголовок игнорируется.
	TrustedProxies []string `json:"trusted_proxies"`

	// LeadRateLimit / ReviewRateLimit — сколько POST-запросов можно сделать с одного IP за окно.
	LeadRateLimit   RateLimitConfig `json:"lead_rate_limit"`
	ReviewRateLimit RateLimitConfig `json:"review_rate_limit"`

	// LeadDuplicateWindowMinutes / ReviewDuplicateWindowMinutes — в течение какого времени
	// повторная заявка/отзыв с тем же телефоном отклоняется. 0 — проверка отключена.
	LeadDuplicateWindowMinutes   int `json:"lead_duplicate_window_minutes"`
	ReviewDuplicateWindowMinutes int `json:"review_duplicate_window_minutes"`
}

// RateLimitConfig ограничение частоты запросов: не больше MaxRequests за WindowMinutes.
// MaxRequests <= 0 — ограничение отключено.
type RateLimitConfig struct {
	MaxRequests   int `json:"max_requests"`
	WindowMinutes int `json:"window_minutes"`
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnv("SERVER_PORT", "8080"),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 10),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 10),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "zahruman"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "nexera_back_db"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			File:   getEnv("LOG_FILE", ""),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Media: MediaConfig{
			Driver:        getEnv("MEDIA_DRIVER", "local"),
			MaxFileSizeMB: int64(getEnvAsInt("MEDIA_MAX_FILE_SIZE_MB", 25)),
			Local: LocalMedia{
				Path:          getEnv("MEDIA_LOCAL_PATH", "./uploads"),
				PublicBaseURL: getEnv("MEDIA_LOCAL_PUBLIC_BASE_URL", ""),
			},
			S3: S3MediaConfig{
				Bucket:        getEnv("MEDIA_S3_BUCKET", ""),
				Region:        getEnv("MEDIA_S3_REGION", ""),
				Endpoint:      getEnv("MEDIA_S3_ENDPOINT", ""),
				AccessKey:     getEnv("MEDIA_S3_ACCESS_KEY", ""),
				SecretKey:     getEnv("MEDIA_S3_SECRET_KEY", ""),
				PublicBaseURL: getEnv("MEDIA_S3_PUBLIC_BASE_URL", ""),
			},
		},
		Kafka: KafkaConfig{
			Brokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
			GroupID: getEnv("KAFKA_GROUP_ID", "detailing-crm"),
			Topics: Topics{
				Orders: getEnv("KAFKA_TOPIC_ORDERS", "orders"),
			},
		},
		Telegram: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		},
		Auth: AuthConfig{
			JWTSecret:     getEnv("JWT_SECRET", ""),
			JWTTTLMinutes: getEnvAsInt("JWT_TTL_MINUTES", 1440),
		},
		Spam: SpamConfig{
			TrustedProxies: getEnvAsList("TRUSTED_PROXIES", "127.0.0.1,::1"),
			LeadRateLimit: RateLimitConfig{
				MaxRequests:   getEnvAsInt("LEAD_RATE_LIMIT_MAX", 5),
				WindowMinutes: getEnvAsInt("LEAD_RATE_LIMIT_WINDOW_MINUTES", 10),
			},
			ReviewRateLimit: RateLimitConfig{
				MaxRequests:   getEnvAsInt("REVIEW_RATE_LIMIT_MAX", 3),
				WindowMinutes: getEnvAsInt("REVIEW_RATE_LIMIT_WINDOW_MINUTES", 60),
			},
			LeadDuplicateWindowMinutes:   getEnvAsInt("LEAD_DUPLICATE_WINDOW_MINUTES", 10),
			ReviewDuplicateWindowMinutes: getEnvAsInt("REVIEW_DUPLICATE_WINDOW_MINUTES", 1440),
		},
	}

	return cfg
}

// getEnv получает значение переменной окружения со значением по умолчанию
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt получает значение переменной окружения как int со значением по умолчанию
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getEnvAsList получает значение переменной окружения как список через запятую; пустые
// элементы отбрасываются, так что пустая переменная — пустой список.
func getEnvAsList(key, defaultValue string) []string {
	var result []string
	for _, item := range strings.Split(getEnv(key, defaultValue), ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
