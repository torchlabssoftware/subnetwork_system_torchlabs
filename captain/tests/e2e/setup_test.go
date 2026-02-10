package e2e

import (
	"context"
	"database/sql"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/torchlabssoftware/subnetwork_system/internal/config"
	server "github.com/torchlabssoftware/subnetwork_system/internal/server"
	websocket "github.com/torchlabssoftware/subnetwork_system/internal/server/websocket"
	"github.com/torchlabssoftware/subnetwork_system/tests/e2e/helpers"
)

// Test configuration
var (
	testServer         *httptest.Server
	adminClient        *helpers.TestClient
	workerClient       *helpers.TestClient
	testDB             *sql.DB
	AdminAPIKey        string
	WorkerAPIKey       string
	PostgresURL        string
	ClickHouseURL      string
	ClickHouseDB       string
	ClickHouseUser     string
	ClickHousePassword string
)

func TestMain(m *testing.M) {
	appEnv := os.Getenv("APP_ENV")
	log.Println("APP_ENV:", appEnv)
	if strings.ToLower(appEnv) == "dev" || strings.ToLower(appEnv) == "" {
		if err := godotenv.Load("../../.env.dev"); err != nil {
			log.Println("Error in loading .env.dev file:", err)
		}
	} else {
		log.Println("Running in production mode, using environment variables from Docker")
	}
	envConfig := config.Load()
	AdminAPIKey = envConfig.Admin_API_KEY
	WorkerAPIKey = envConfig.Worker_API_KEY
	PostgresURL = envConfig.POSTGRES_URL
	ClickHouseURL = envConfig.CLICKHOUSE_URL
	ClickHouseDB = envConfig.CLICKHOUSE_DB
	ClickHouseUser = envConfig.CLICKHOUSE_USER
	ClickHousePassword = envConfig.CLICKHOUSE_PASSWORD
	if err := setup(); err != nil {
		log.Fatalf("Failed to setup test environment: %v", err)
	}
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() error {
	//get env and connect to postgres and cliskhouse
	log.Println("[E2E] Setting up test environment...")
	var err error
	testDB, err = sql.Open("pgx", PostgresURL)
	if err != nil {
		return err
	}
	if err = testDB.Ping(); err != nil {
		return err
	}
	log.Println("[E2E] Connected to PostgreSQL")
	clickhouseConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{ClickHouseURL},
		Auth: clickhouse.Auth{
			Database: ClickHouseDB,
			Username: ClickHouseUser,
			Password: ClickHousePassword,
		},
		Debug: false,
		// Debugf: func(format string, v ...interface{}) {
		// 	fmt.Printf(format, v...)
		// },
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:  time.Second * 5,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		return err
	}
	if err = clickhouseConn.Ping(context.Background()); err != nil {
		log.Printf("[E2E] Warning: ClickHouse not available: %v", err)
		clickhouseConn = nil
	} else {
		log.Println("[E2E] Connected to ClickHouse")
	}
	//create router and server
	wsManager := websocket.NewWebsocketManager()
	router := server.NewRouter(testDB, clickhouseConn, wsManager)
	testServer = httptest.NewServer(router)
	log.Printf("[E2E] Test server started at: %s", testServer.URL)
	//create admin and worker client
	adminClient = helpers.NewAdminClient(testServer.URL, AdminAPIKey)
	workerClient = helpers.NewWorkerClient(testServer.URL, WorkerAPIKey)
	return nil
}

func teardown() {
	log.Println("[E2E] Tearing down test environment...")
	if testServer != nil {
		testServer.Close()
	}
	if testDB != nil {
		testDB.Close()
	}
}

func GetAdminClient() *helpers.TestClient {
	return adminClient
}

func GetWorkerClient() *helpers.TestClient {
	return workerClient
}

func GetTestDB() *sql.DB {
	return testDB
}

func GetTestServerURL() string {
	return testServer.URL
}
