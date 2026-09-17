// Command identity starts the identity HTTP service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/service"
	"github.com/Yukinoshita03/gopherops/internal/identity/repository"
	"github.com/Yukinoshita03/gopherops/internal/identity/transport/httpapi"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		log.Printf("identity service stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := os.Getenv("IDENTITY_MYSQL_DSN")
	if dsn == "" {
		return errors.New("IDENTITY_MYSQL_DSN is required")
	}

	addr := os.Getenv("IDENTITY_HTTP_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open identity database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get identity database connection: %w", err)
	}
	defer sqlDB.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPing()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping identity database: %w", err)
	}

	userRepo := repository.NewMySQLUserRepository(db)
	registerService := service.NewRegisterService(userRepo)
	loginService := service.NewLoginService(userRepo)
	router := httpapi.NewRouter(registerService, loginService)

	log.Printf("identity service listening on %s", addr)
	if err := router.Run(addr); err != nil {
		return fmt.Errorf("run identity HTTP: %w", err)
	}
	return nil
}
