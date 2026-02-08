package engine

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectorConfig holds PostgreSQL connection settings
type ConnectorConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	// Pool settings
	MaxConns    int32
	MinConns    int32
	MaxIdleTime time.Duration
}

// DefaultConfig returns sensible defaults
func DefaultConfig() ConnectorConfig {
	return ConnectorConfig{
		Host:        "localhost",
		Port:        5432,
		Database:    "chameleon",
		User:        "postgres",
		Password:    "",
		MaxConns:    10,
		MinConns:    2,
		MaxIdleTime: 5 * time.Minute,
	}
}

// ConnectionString builds the pgx connection string
func (c ConnectorConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		c.Host, c.Port, c.Database, c.User, c.Password,
	)
}

// Connector manages the PostgreSQL connection pool
type Connector struct {
	pool   *pgxpool.Pool
	config ConnectorConfig
}

// NewConnector creates a new connector (does not connect yet)
func NewConnector(config ConnectorConfig) *Connector {
	return &Connector{config: config}
}

// Connect establishes the connection pool
func (c *Connector) Connect(ctx context.Context) error {
	poolConfig, err := pgxpool.ParseConfig(c.config.ConnectionString())
	if err != nil {
		return fmt.Errorf("invalid connection config: %w", err)
	}

	poolConfig.MaxConns = c.config.MaxConns
	poolConfig.MinConns = c.config.MinConns
	poolConfig.MaxConnIdleTime = c.config.MaxIdleTime

	pool, err := pgxpool.New(ctx, poolConfig.ConnString())
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	c.pool = pool
	return nil
}

// Pool returns the underlying connection pool
// Returns nil if not connected
func (c *Connector) Pool() *pgxpool.Pool {
	return c.pool
}

// IsConnected returns true if the pool is active
func (c *Connector) IsConnected() bool {
	return c.pool != nil
}

// Ping verifies the connection is alive
func (c *Connector) Ping(ctx context.Context) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	return c.pool.Ping(ctx)
}

// Close closes the connection pool
func (c *Connector) Close() {
	if c.pool != nil {
		c.pool.Close()
		c.pool = nil
	}
}

// ParseConnectionString parses a PostgreSQL connection URL
// Format: postgresql://user:password@host:port/dbname
// or: postgres://user:password@host:port/dbname
func ParseConnectionString(connStr string) (ConnectorConfig, error) {
	parsed, err := url.Parse(connStr)
	if err != nil {
		return ConnectorConfig{}, fmt.Errorf("invalid connection string: %w", err)
	}

	if parsed.Scheme != "postgresql" && parsed.Scheme != "postgres" {
		return ConnectorConfig{}, fmt.Errorf("unsupported scheme: %s (expected postgresql or postgres)", parsed.Scheme)
	}

	config := DefaultConfig()

	// Host
	config.Host = parsed.Hostname()
	if config.Host == "" {
		config.Host = "localhost"
	}

	// Port
	if portStr := parsed.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return ConnectorConfig{}, fmt.Errorf("invalid port: %w", err)
		}
		config.Port = port
	}

	// Database
	if parsed.Path != "" && parsed.Path != "/" {
		config.Database = parsed.Path[1:] // Remove leading slash
	}

	// User
	if parsed.User != nil {
		config.User = parsed.User.Username()
		password, ok := parsed.User.Password()
		if ok {
			config.Password = password
		}
	}

	return config, nil
}

// Query executes a SQL query and returns rows
func (c *Connector) Query(ctx context.Context, sql string) ([]map[string]interface{}, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to database")
	}

	rows, err := c.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	columns := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col.Name] = values[i]
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
