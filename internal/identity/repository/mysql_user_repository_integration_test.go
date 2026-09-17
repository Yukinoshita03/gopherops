package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Yukinoshita03/gopherops/internal/identity/application/service"
	"github.com/Yukinoshita03/gopherops/internal/identity/domain"
	"github.com/Yukinoshita03/gopherops/internal/identity/transport/httpapi"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const identityTestDatabase = "identity_test_db"

func TestMySQLUserRepositoryIntegration(t *testing.T) {
	repo, db := openMySQLUserRepositoryForTest(t)

	t.Run("create and query by username and ID", func(t *testing.T) {
		user := newIntegrationTestUser(t, db)
		if err := repo.Create(context.Background(), user); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if user.ID <= 0 {
			t.Fatalf("Create() ID = %d, want a generated ID", user.ID)
		}
		if user.CreatedAt.IsZero() {
			t.Fatal("Create() left CreatedAt zero")
		}

		byUsername, err := repo.GetByUsername(context.Background(), user.Username)
		if err != nil {
			t.Fatalf("GetByUsername() error = %v", err)
		}
		assertSamePersistedUser(t, byUsername, user)

		byID, err := repo.GetByID(context.Background(), user.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		assertSamePersistedUser(t, byID, user)
	})

	t.Run("missing user maps to domain error", func(t *testing.T) {
		username := integrationTestUsername()
		got, err := repo.GetByUsername(context.Background(), username)
		if got != nil {
			t.Fatalf("GetByUsername() user = %+v, want nil", got)
		}
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("GetByUsername() error = %v, want ErrUserNotFound", err)
		}

		got, err = repo.GetByID(context.Background(), 0)
		if got != nil {
			t.Fatalf("GetByID() user = %+v, want nil", got)
		}
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("GetByID() error = %v, want ErrUserNotFound", err)
		}
	})

	t.Run("duplicate username is rejected by database constraint", func(t *testing.T) {
		user := newIntegrationTestUser(t, db)
		if err := repo.Create(context.Background(), user); err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		duplicate := &domain.User{
			Username:     user.Username,
			PasswordHash: "another-hash",
		}
		if err := repo.Create(context.Background(), duplicate); !errors.Is(err, domain.ErrUserAlreadyExists) {
			t.Fatalf("second Create() error = %v, want ErrUserAlreadyExists", err)
		}
	})

	t.Run("concurrent registrations return one success and one conflict", func(t *testing.T) {
		username := integrationTestUsername()
		t.Cleanup(func() {
			if err := db.Where("username = ?", username).Delete(&userModel{}).Error; err != nil {
				t.Errorf("clean up concurrent registration: %v", err)
			}
		})

		router := httpapi.NewRouter(
			service.NewRegisterService(repo),
			service.NewLoginService(repo),
		)
		ready := make(chan struct{}, 2)
		start := make(chan struct{})
		results := make(chan struct {
			status int
			body   []byte
		}, 2)
		var requests sync.WaitGroup
		for range 2 {
			requests.Add(1)
			go func() {
				defer requests.Done()
				body := fmt.Sprintf(`{"username":%q,"password":"integration-password"}`, username)
				request := httptest.NewRequest(
					http.MethodPost,
					"/v1/auth/register",
					bytes.NewBufferString(body),
				)
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				ready <- struct{}{}
				<-start
				router.ServeHTTP(recorder, request)
				results <- struct {
					status int
					body   []byte
				}{status: recorder.Code, body: recorder.Body.Bytes()}
			}()
		}
		<-ready
		<-ready
		close(start)
		requests.Wait()
		close(results)

		created, conflicts := 0, 0
		for result := range results {
			switch result.status {
			case http.StatusCreated:
				created++
			case http.StatusConflict:
				conflicts++
				var response struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(result.body, &response); err != nil {
					t.Fatalf("decode conflict response: %v", err)
				}
				if response.Code != "username_already_exists" {
					t.Fatalf("conflict code = %q, want %q", response.Code, "username_already_exists")
				}
			default:
				t.Fatalf("registration status = %d, want 201 or 409; body: %s", result.status, result.body)
			}
		}
		if created != 1 || conflicts != 1 {
			t.Fatalf("created=%d conflicts=%d, want exactly one of each", created, conflicts)
		}

		var count int64
		if err := db.Model(&userModel{}).Where("username = ?", username).Count(&count).Error; err != nil {
			t.Fatalf("count registered users: %v", err)
		}
		if count != 1 {
			t.Fatalf("rows for username = %d, want 1", count)
		}
	})

	t.Run("cancelled context is propagated", func(t *testing.T) {
		user := newIntegrationTestUser(t, db)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := repo.Create(ctx, user)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Create() error = %v, want context.Canceled", err)
		}

		got, err := repo.GetByUsername(context.Background(), user.Username)
		if got != nil || !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("cancelled Create() left a row: user=%+v error=%v", got, err)
		}
	})

	t.Run("closed database returns an error", func(t *testing.T) {
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("db.DB() error = %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		if _, err := repo.GetByID(context.Background(), 1); err == nil {
			t.Fatal("GetByID() error = nil after database was closed")
		}
	})
}

func openMySQLUserRepositoryForTest(t *testing.T) (*MySQLUserRepository, *gorm.DB) {
	t.Helper()

	dsn := os.Getenv("IDENTITY_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("IDENTITY_TEST_MYSQL_DSN is not set; skipping MySQL integration test")
	}
	if database := mysqlDatabaseName(dsn); database != identityTestDatabase {
		t.Fatalf("IDENTITY_TEST_MYSQL_DSN must target %q, got %q", identityTestDatabase, database)
	}

	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open MySQL test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get MySQL test connection: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Fatalf("ping MySQL test database: %v", err)
	}
	if !db.Migrator().HasTable(&userModel{}) {
		t.Fatalf("table users is missing in %q; apply migrations/identity/001_create_users.sql first", identityTestDatabase)
	}

	return NewMySQLUserRepository(db), db
}

func newIntegrationTestUser(t *testing.T, db *gorm.DB) *domain.User {
	t.Helper()

	user := &domain.User{
		Username:     integrationTestUsername(),
		PasswordHash: "integration-test-hash",
	}
	t.Cleanup(func() {
		if err := db.Where("username = ?", user.Username).Delete(&userModel{}).Error; err != nil {
			t.Errorf("clean up integration test user: %v", err)
		}
	})
	return user
}

func integrationTestUsername() string {
	return fmt.Sprintf("repo_it_%d", time.Now().UnixNano())
}

func assertSamePersistedUser(t *testing.T, got, want *domain.User) {
	t.Helper()
	if got == nil {
		t.Fatal("persisted user = nil")
	}
	if got.ID != want.ID || got.Username != want.Username || got.PasswordHash != want.PasswordHash {
		t.Fatalf("persisted user = %+v, want ID=%d Username=%q", got, want.ID, want.Username)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("persisted user has zero CreatedAt")
	}
}

func mysqlDatabaseName(dsn string) string {
	separator := strings.Index(dsn, ")/")
	if separator < 0 {
		return ""
	}

	database := dsn[separator+2:]
	if query := strings.IndexByte(database, '?'); query >= 0 {
		database = database[:query]
	}
	return database
}
