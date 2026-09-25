// Package config อ่าน environment variable ครั้งเดียวตอนเริ่มโปรแกรม แล้ว validate
// ถ้าค่าไหนขาดหรือผิด โปรแกรมจะหยุดทันทีพร้อมบอกว่าตัวไหนผิด (ดีกว่าไปพังกลางทาง)
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	_ "github.com/joho/godotenv/autoload" // โหลด .env ถ้ามี (ตอนรันนอก docker)
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type Config struct {
	App      App
	Database Database
	Keycloak Keycloak
}

type App struct {
	Env             string        `env:"APP_ENV" envDefault:"development" validate:"oneof=development production test"`
	Port            int           `env:"APP_PORT" envDefault:"8080" validate:"gte=1,lte=65535"`
	CORSOrigin      string        `env:"CORS_ORIGIN" envDefault:"http://jongyoung.localhost" validate:"required,url"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info" validate:"oneof=debug info warn error"`
	ShutdownTimeout time.Duration `env:"APP_SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

func (a App) Addr() string       { return fmt.Sprintf(":%d", a.Port) }
func (a App) IsDevelopment() bool { return a.Env == "development" }

type Database struct {
	DSN string `env:"POSTGRES_DSN" validate:"required"`
}

// Validate เช็คว่า DSN parse ได้และมีชื่อ database — ห้าม log DSN เต็มเพราะมีรหัสผ่าน
func (d Database) Validate() error {
	u, err := url.Parse(d.DSN)
	if err != nil {
		return errors.New("POSTGRES_DSN: รูปแบบไม่ถูกต้อง")
	}
	if strings.Trim(u.Path, "/") == "" {
		return errors.New("POSTGRES_DSN: ไม่มีชื่อ database")
	}
	return nil
}

type Keycloak struct {
	BaseURL  string `env:"KEYCLOAK_BASE_URL" envDefault:"http://keycloak.jongyoung.localhost" validate:"required,url"`
	Realm    string `env:"KEYCLOAK_REALM" envDefault:"jongyoung" validate:"required"`
	Audience string `env:"KEYCLOAK_AUDIENCE" envDefault:"jongyoung-api" validate:"required"`
}

// Issuer = URL ของ realm — ต้องตรงกับ iss ใน token ทุกตัว (ดู CLAUDE.md 2.3)
func (k Keycloak) Issuer() string {
	return strings.TrimRight(k.BaseURL, "/") + "/realms/" + k.Realm
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("อ่าน environment: %w", err)
	}
	if err := validate.Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("config ไม่ถูกต้อง: %w", err)
	}
	if err := cfg.Database.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
