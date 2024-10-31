package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"guarddev/logger"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type DatabaseConnectProps struct {
	Logger *logger.LogMiddleware
}

type Database struct {
	Queries
	logger *logger.LogMiddleware
}

func Connect(ctx context.Context, args DatabaseConnectProps) *Database {
	tracer := otel.Tracer("postgres/Connect")
	ctx, span := tracer.Start(ctx, "Connect")
	defer span.End()

	connectRetries := 5
	var conn *sql.DB
	var err error

	logger := args.Logger.Logger(ctx)

	for connectRetries > 0 {
		conn, err = getConnection(ctx)
		if err == nil {
			logger.Info("[Postgres] Database client started")
			break
		}
		connectRetries -= 1
		sleepTime := 5
		logger.Error(
			"[Postgres] Could not connect to Postgres. Retrying after sleeping.",
			zap.Error(err),
			zap.Int("Retries Left", connectRetries),
			zap.Int("Sleep Time", sleepTime))
		time.Sleep(time.Second * time.Duration(sleepTime))
	}

	if connectRetries <= 0 {
		logger.Error("[Postgres] Failed to Connect to Postgres")
		span.RecordError(fmt.Errorf("failed to connect to Postgres"))
		os.Exit(1)
	}

	queries := New(conn)
	return &Database{Queries: *queries, logger: args.Logger}
}

func getConnection(ctx context.Context) (*sql.DB, error) {
	tracer := otel.Tracer("postgres/getConnection")
	_, span := tracer.Start(ctx, "getConnection")
	defer span.End()

	host := os.Getenv("POSTGRES_DB_HOST")
	port := os.Getenv("POSTGRES_DB_PORT")
	user := os.Getenv("POSTGRES_DB_USER")
	password := os.Getenv("POSTGRES_DB_PASS")
	dbname := os.Getenv("POSTGRES_DB_NAME")

	sslMode := "disable"

	postgresqlDbInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslMode,
	)

	db, err := sql.Open("postgres", postgresqlDbInfo)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return db, nil
}

type SetupNewUserProps struct {
	EmailAddr string
	FullName  string
}

func (d *Database) SetupNewUser(ctx context.Context, args SetupNewUserProps) (*UserInfo, error) {
	tracer := otel.Tracer("postgres/SetupNewUser")
	ctx, span := tracer.Start(ctx, "SetupNewUser")
	defer span.End()

	fName := args.FullName
	emailAddr := args.EmailAddr

	fullName := cases.Title(language.Und).String(strings.ToLower(fName))

	user, err := d.Queries.AddUser(ctx, AddUserParams{
		Email:    emailAddr,
		FullName: fullName,
	})
	if err != nil {
		d.logger.Logger(ctx).Error(
			"[Postgres] Could not setup new user",
			zap.Error(err),
			zap.String("user_email", emailAddr),
			zap.String("user_name", fName),
		)
		span.RecordError(err)
		return nil, fmt.Errorf("could not setup new user")
	}

	return &user, err
}
