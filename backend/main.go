package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type Bitcoin struct {
	Symbol    string    `json:"symbol" db:"symbol"`
	Price     int       `json:"price" db:"price"`
	Rank      *int      `json:"rank,omitempty" db:"rank"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CacheService struct {
	db          *sql.DB
	redisClient *redis.Client
	ctx         context.Context
	cacheTTL    time.Duration
}

const (
	cachePrefix     = "bitcoin:"
	rankCacheKey    = "bitcoin:rankings"
	defaultCacheTTL = 1 * time.Hour
)

func NewCacheService(db *sql.DB, redisClient *redis.Client) *CacheService {
	return &CacheService{
		db:          db,
		redisClient: redisClient,
		ctx:         context.Background(),
		cacheTTL:    defaultCacheTTL,
	}
}

func (cs *CacheService) getBitcoinCacheKey(symbol string) string {
	return fmt.Sprintf("%s%s", cachePrefix, symbol)
}

// CACHE PRIMING: Load all data from DB into cache at startup
func (cs *CacheService) PrimeCache() error {
	log.Println("Starting cache priming...")

	// Get all bitcoins from database
	rows, err := cs.db.Query(`
		SELECT symbol, price, created_at, updated_at
		FROM bitcoins
		ORDER BY price DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to query bitcoins: %w", err)
	}
	defer rows.Close()

	count := 0
	var bitcoins []Bitcoin

	for rows.Next() {
		var b Bitcoin
		if err := rows.Scan(&b.Symbol, &b.Price, &b.CreatedAt, &b.UpdatedAt); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		bitcoins = append(bitcoins, b)

		// Cache individual bitcoin
		data, err := json.Marshal(b)
		if err != nil {
			log.Printf("Error marshaling bitcoin %s: %v", b.Symbol, err)
			continue
		}

		err = cs.redisClient.Set(cs.ctx, cs.getBitcoinCacheKey(b.Symbol), data, cs.cacheTTL).Err()
		if err != nil {
			log.Printf("Error caching bitcoin %s: %v", b.Symbol, err)
			continue
		}

		count++
	}

	log.Printf("Cache priming completed: %d bitcoins loaded into cache", count)
	return nil
}

// READ-THROUGH: Get bitcoin from cache, fallback to DB if not found
func (cs *CacheService) GetBitcoin(symbol string) (*Bitcoin, error) {
	cacheKey := cs.getBitcoinCacheKey(symbol)

	// Try cache first
	cached, err := cs.redisClient.Get(cs.ctx, cacheKey).Result()
	if err == nil {
		log.Printf("Cache HIT for %s", symbol)
		var bitcoin Bitcoin
		if err := json.Unmarshal([]byte(cached), &bitcoin); err != nil {
			log.Printf("Error unmarshaling cached bitcoin: %v", err)
		} else {
			return &bitcoin, nil
		}
	}

	log.Printf("Cache MISS for %s", symbol)

	// Cache miss - read from database
	var bitcoin Bitcoin
	err = cs.db.QueryRow(`
		SELECT symbol, price, created_at, updated_at
		FROM bitcoins
		WHERE symbol = $1
	`, symbol).Scan(&bitcoin.Symbol, &bitcoin.Price, &bitcoin.CreatedAt, &bitcoin.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Write to cache for future reads
	data, err := json.Marshal(bitcoin)
	if err != nil {
		log.Printf("Error marshaling bitcoin: %v", err)
	} else {
		err = cs.redisClient.Set(cs.ctx, cacheKey, data, cs.cacheTTL).Err()
		if err != nil {
			log.Printf("Error caching bitcoin: %v", err)
		}
	}

	return &bitcoin, nil
}

// WRITE-THROUGH: Write to DB and cache simultaneously
func (cs *CacheService) SetBitcoin(symbol string, price int) (*Bitcoin, error) {
	// Write to database first
	var bitcoin Bitcoin
	err := cs.db.QueryRow(`
		INSERT INTO bitcoins (symbol, price)
		VALUES ($1, $2)
		ON CONFLICT (symbol)
		DO UPDATE SET price = $2, updated_at = CURRENT_TIMESTAMP
		RETURNING symbol, price, created_at, updated_at
	`, symbol, price).Scan(&bitcoin.Symbol, &bitcoin.Price, &bitcoin.CreatedAt, &bitcoin.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Write to cache
	data, err := json.Marshal(bitcoin)
	if err != nil {
		log.Printf("Error marshaling bitcoin: %v", err)
	} else {
		err = cs.redisClient.Set(cs.ctx, cs.getBitcoinCacheKey(symbol), data, cs.cacheTTL).Err()
		if err != nil {
			log.Printf("Error caching bitcoin: %v", err)
		}
	}

	// Invalidate rankings cache since order may have changed
	cs.redisClient.Del(cs.ctx, rankCacheKey)

	log.Printf("Write-through completed for %s", symbol)
	return &bitcoin, nil
}

// Get all bitcoins ranked by price
func (cs *CacheService) GetBitcoinsRanked() ([]Bitcoin, error) {
	// Try cache first
	cached, err := cs.redisClient.Get(cs.ctx, rankCacheKey).Result()
	if err == nil {
		log.Println("Rankings cache HIT")
		var bitcoins []Bitcoin
		if err := json.Unmarshal([]byte(cached), &bitcoins); err != nil {
			log.Printf("Error unmarshaling cached rankings: %v", err)
		} else {
			return bitcoins, nil
		}
	}

	log.Println("Rankings cache MISS")

	// Cache miss - read from database
	rows, err := cs.db.Query(`
		SELECT
			symbol,
			price,
			created_at,
			updated_at,
			ROW_NUMBER() OVER (ORDER BY price DESC) as rank
		FROM bitcoins
		ORDER BY price DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var bitcoins []Bitcoin
	for rows.Next() {
		var b Bitcoin
		if err := rows.Scan(&b.Symbol, &b.Price, &b.CreatedAt, &b.UpdatedAt, &b.Rank); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		bitcoins = append(bitcoins, b)
	}

	// Cache the rankings
	data, err := json.Marshal(bitcoins)
	if err != nil {
		log.Printf("Error marshaling rankings: %v", err)
	} else {
		err = cs.redisClient.Set(cs.ctx, rankCacheKey, data, cs.cacheTTL).Err()
		if err != nil {
			log.Printf("Error caching rankings: %v", err)
		}
	}

	return bitcoins, nil
}

// Delete bitcoin from DB and cache
func (cs *CacheService) DeleteBitcoin(symbol string) (*Bitcoin, error) {
	// Delete from database
	var bitcoin Bitcoin
	err := cs.db.QueryRow(`
		DELETE FROM bitcoins WHERE symbol = $1
		RETURNING symbol, price, created_at, updated_at
	`, symbol).Scan(&bitcoin.Symbol, &bitcoin.Price, &bitcoin.CreatedAt, &bitcoin.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Delete from cache
	cs.redisClient.Del(cs.ctx, cs.getBitcoinCacheKey(symbol))

	// Invalidate rankings cache
	cs.redisClient.Del(cs.ctx, rankCacheKey)

	log.Printf("Deleted %s from DB and cache", symbol)
	return &bitcoin, nil
}

func main() {
	// Database connection
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	dbUser := getEnv("POSTGRES_USER", "postgres")
	dbPassword := getEnv("POSTGRES_PASSWORD", "postgres")
	dbName := getEnv("POSTGRES_DB", "bitcoin_db")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Redis connection
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")

	// Initialize cache service
	cacheService := NewCacheService(db, redisClient)

	// Prime the cache at startup
	if err := cacheService.PrimeCache(); err != nil {
		log.Printf("Warning: Cache priming failed: %v", err)
	}

	// Setup Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Get all bitcoins ranked by price
	router.GET("/api/bitcoins", func(c *gin.Context) {
		bitcoins, err := cacheService.GetBitcoinsRanked()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bitcoins"})
			return
		}
		c.JSON(http.StatusOK, bitcoins)
	})

	// Get single bitcoin by symbol
	router.GET("/api/bitcoins/:symbol", func(c *gin.Context) {
		symbol := c.Param("symbol")
		bitcoin, err := cacheService.GetBitcoin(symbol)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bitcoin"})
			return
		}
		if bitcoin == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bitcoin not found"})
			return
		}
		c.JSON(http.StatusOK, bitcoin)
	})

	// Create or update bitcoin
	router.POST("/api/bitcoins", func(c *gin.Context) {
		var req struct {
			Symbol string `json:"symbol" binding:"required"`
			Price  int    `json:"price" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol and price are required"})
			return
		}

		bitcoin, err := cacheService.SetBitcoin(req.Symbol, req.Price)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create/update bitcoin"})
			return
		}

		c.JSON(http.StatusCreated, bitcoin)
	})

	// Update bitcoin
	router.PUT("/api/bitcoins/:symbol", func(c *gin.Context) {
		symbol := c.Param("symbol")
		var req struct {
			Price int `json:"price" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Price is required"})
			return
		}

		bitcoin, err := cacheService.SetBitcoin(symbol, req.Price)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bitcoin"})
			return
		}

		c.JSON(http.StatusOK, bitcoin)
	})

	// Delete bitcoin
	router.DELETE("/api/bitcoins/:symbol", func(c *gin.Context) {
		symbol := c.Param("symbol")
		bitcoin, err := cacheService.DeleteBitcoin(symbol)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bitcoin"})
			return
		}
		if bitcoin == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bitcoin not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Bitcoin deleted successfully",
			"bitcoin": bitcoin,
		})
	})

	// Cache stats endpoint
	router.GET("/api/cache/stats", func(c *gin.Context) {
		info := redisClient.Info(ctx, "stats").Val()
		c.JSON(http.StatusOK, gin.H{"info": info})
	})

	// Start server
	port := getEnv("PORT", "3000")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server running on port %s", port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
