package logger

import (
	"io"
	"nexera_back/internal/config"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger представляет логгер приложения
type Logger struct {
	*logrus.Logger
}

// New Создает новый экземпляр логгера
func New(cfg *config.LoggerConfig) *Logger {
	log := logrus.New()

	// Устанавливаем уровень логирования
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Устанавливаем формат логирования
	if cfg.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// Настройка вывода в файл
	if cfg.File != "" {
		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(io.MultiWriter(os.Stdout, file))
		} else {
			log.WithError(err).Error("Failed to open log file")
		}
	}

	return &Logger{log}
}

// WithField добавляет поле к логгеру
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields добавляет несколько полей к логгеру
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithError добавляет ошибку к логгеру
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}
