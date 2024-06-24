// By Carsen Klock 2024 under the MIT license
// https://github.com/context-labs/ollamark
// https://ollamark.com
// Ollamark Server

package main

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	tollbooth "github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BenchmarkResult struct {
	ModelName       string              `json:"model_name"`
	Timestamp       int64               `json:"timestamp"`
	Duration        float64             `json:"duration"`
	TokensPerSecond float64             `json:"tokens_per_second"`
	EvalCount       int                 `json:"eval_count"`
	EvalDuration    int64               `json:"eval_duration"`
	Iterations      int                 `json:"iterations"`
	SysInfo         *SysInfo            `json:"sys_info"`
	GPUInfo         *GPUInfo            `json:"gpu_info"`
	OllamaVersion   string              `json:"ollama_version"`
	ClientType      string              `json:"client_type"`
	ClientVersion   string              `json:"client_version"`
	SubmissionID    string              `json:"submission_id"`
	IP              string              `json:"ip"`
	ProofOfWork     ProofOfWorkSolution `json:"proof_of_work"`
}

type SysInfo struct {
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Version string `json:"version"`
	Kernel  string `json:"kernel"`
	CPU     string `json:"cpu"`
	CPUName string `json:"cpu_name"`
	Memory  string `json:"memory"`
}

type GPUInfo struct {
	Name          string `json:"name"`
	Vendor        string `json:"vendor"`
	Memory        string `json:"memory"`
	DriverVersion string `json:"driver_version"`
	Count         int    `json:"count"`
}

type ModelInfo struct {
	Name         string
	Parameters   string
	Quantization string
}

// Models supported
var MODELS = []ModelInfo{
	{Name: "llama3", Parameters: "8B", Quantization: "Q4_0"},
	{Name: "phi3", Parameters: "3B", Quantization: "Q4_K_M"},
	{Name: "phi3:14b", Parameters: "14B", Quantization: "Q4_0"},
	{Name: "aya", Parameters: "8B", Quantization: "Q4_0"},
	{Name: "aya:35b", Parameters: "35B", Quantization: "Q4_0"},
	{Name: "gemma", Parameters: "7B", Quantization: "Q4_0"},
	{Name: "gemma:2b", Parameters: "2B", Quantization: "Q4_0"},
	{Name: "falcon2", Parameters: "11B", Quantization: "Q4_0"},
	{Name: "mistral", Parameters: "7B", Quantization: "Q4_0"},
	{Name: "mixtral:8x22b", Parameters: "176B", Quantization: "Q4_0"},
	{Name: "mixtral:8x7b", Parameters: "56B", Quantization: "Q4_0"},
	{Name: "command-r", Parameters: "35B", Quantization: "Q4_0"},
	{Name: "command-r-plus", Parameters: "104B", Quantization: "Q4_0"},
	{Name: "dolphin-llama3", Parameters: "8B", Quantization: "Q4_0"},
	{Name: "dolphin-llama3:70b", Parameters: "70B", Quantization: "Q4_0"},
	{Name: "dolphin-mixtral:8x22b", Parameters: "176B", Quantization: "Q4_0"},
	{Name: "dolphin-mixtral:8x7b", Parameters: "56B", Quantization: "Q4_0"},
	{Name: "llama3-chatqa", Parameters: "8B", Quantization: "Q4_0"},
	{Name: "llama3:70b", Parameters: "70B", Quantization: "Q4_0"},
	{Name: "llama3-gradient:8b", Parameters: "8B", Quantization: "Q4_0"},
	{Name: "llama3-gradient:70b", Parameters: "70B", Quantization: "Q4_0"},
	{Name: "qwen", Parameters: "7B", Quantization: "Q4_0"},
	{Name: "qwen2", Parameters: "7B", Quantization: "Q4_0"},
	{Name: "qwen2:0.5b", Parameters: "0.5B", Quantization: "Q4_0"},
	{Name: "qwen2:1.5b", Parameters: "1.5B", Quantization: "Q4_0"},
	{Name: "llama2", Parameters: "7B", Quantization: "Q4_0"},
}

var cache sync.Map

type CacheItem struct {
	Data      []BenchmarkResult
	Count     int64
	Timestamp time.Time
}

func connectDB() (*mongo.Client, error) {
	mongodblink := os.Getenv("MONGODB")
	clientOptions := options.Client().ApplyURI(mongodblink)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func insertBenchmark(client *mongo.Client, benchmark BenchmarkResult) error {
	collection := client.Database("ollamark_db").Collection("benchmarks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, benchmark)
	if err != nil {
		return err
	}
	return nil
}

func LoadPrivateKey(privateKeyData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyData))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return privateKey.(*rsa.PrivateKey), nil
}

func DecryptData(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	return privateKey.Decrypt(nil, data, &rsa.OAEPOptions{Hash: crypto.SHA256})
}

func decryptAESGCM(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func verifySignature(submissionID, signature, secretKey string) bool {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(submissionID))
	expectedMAC := mac.Sum(nil)
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(signatureBytes, expectedMAC)
}

func checkSubmissionID(client *mongo.Client, submissionID string) (bool, error) {
	collection := client.Database("ollamark_db").Collection("benchmarks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := collection.CountDocuments(ctx, bson.M{"submissionid": submissionID})
	if err != nil {
		return false, err
	}

	return count == 0, nil
}

// Function to validate JWT token
func validateJWT(tokenString string) (jwt.MapClaims, error) {
	secretKey := os.Getenv("KEY")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		log.Printf("JWT parsing error: %v", err) // Log any JWT parsing errors
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	} else {
		log.Printf("Invalid JWT token") // Log if the token is invalid
		return nil, fmt.Errorf("invalid token")
	}
}

// Middleware to validate JWT token
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
			fmt.Printf("Missing Authorization header: %v", tokenString)
			c.Abort()
			return
		}

		claims, err := validateJWT(strings.TrimPrefix(tokenString, "Bearer "))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			fmt.Printf("Invalid token: %v", err)
			c.Abort()
			return
		}

		// monogo client
		client, err := connectDB()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
			c.Abort()
			return
		}

		// Check if the nonce has been used before to prevent replay attacks
		nonce := claims["nonce"].(string)
		isUnique, err := checkSubmissionID(client, nonce)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check submission"})
			fmt.Printf("Failed to check submission ID: %v", err)
			return
		}

		if !isUnique {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Replay attack detected"})
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}

func contains(models []ModelInfo, modelName string) bool {
	for _, model := range models {
		if model.Name == modelName {
			return true
		}
	}
	return false
}

var ipRequests = make(map[string]int)
var ipLastRequest = make(map[string]time.Time)
var requestLimit = 1
var timeWindow = 1 * time.Second

// checkIP checks if an IP address is spamming and rate limits it
func checkIP(ip string) bool {
	now := time.Now()
	if lastRequest, exists := ipLastRequest[ip]; exists && now.Sub(lastRequest) > timeWindow {
		ipRequests[ip] = 0
	}

	ipRequests[ip]++
	ipLastRequest[ip] = now

	return ipRequests[ip] <= requestLimit
}

// ADMIN ONLY: ban ip from submit benchmark
func banIP(ip string) {
	// if ip is in db then remove all its benchmark submissions
	client, err := connectDB()
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database("ollamark_db").Collection("benchmarks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection.DeleteMany(ctx, bson.M{"ip": ip})
}

// ADMIN ONLY: remove benchmark submission
func removeBenchmark(client *mongo.Client, submissionID string) {
	collection := client.Database("ollamark_db").Collection("benchmarks")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection.DeleteOne(ctx, bson.M{"submissionid": submissionID})
}
func fetchBenchmarks(client *mongo.Client, filter bson.M, sortBy string, sortOrder int, page, limit int) ([]BenchmarkResult, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("benchmarks:%s:%d:%d:%d:%s", sortBy, sortOrder, page, limit, filter)
	if item, found := cache.Load(cacheKey); found {
		cacheItem := item.(CacheItem)
		if time.Since(cacheItem.Timestamp) < 5*time.Second {
			return cacheItem.Data, cacheItem.Count, nil
		}
	}

	collection := client.Database("ollamark_db").Collection("benchmarks")

	pipeline := []bson.M{
		{"$match": filter},
		{"$sort": bson.M{sortBy: sortOrder}},
		{"$skip": int64((page - 1) * limit)},
		{"$limit": int64(limit)},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var benchmarks []BenchmarkResult
	if err := cursor.All(ctx, &benchmarks); err != nil {
		return nil, 0, err
	}

	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	cache.Store(cacheKey, CacheItem{Data: benchmarks, Count: total, Timestamp: time.Now()})

	return benchmarks, total, nil
}

// ProofOfWorkChallenge represents a proof-of-work challenge
type ProofOfWorkChallenge struct {
	Challenge  string `json:"challenge"`
	Difficulty int    `json:"difficulty"`
	Timestamp  int64  `json:"timestamp"`
}

// ProofOfWorkSolution represents a solution to a proof-of-work challenge
type ProofOfWorkSolution struct {
	Challenge  string `json:"challenge"`
	Nonce      string `json:"nonce"`
	Timestamp  int64  `json:"timestamp"`
	Difficulty int    `json:"difficulty"`
}

// GenerateProofOfWorkChallenge generates a new proof-of-work challenge
func GenerateProofOfWorkChallenge() ProofOfWorkChallenge {
	difficulty := GetDynamicDifficulty()
	// log.Printf("Generated PoW challenge with difficulty: %d", difficulty)
	challenge := make([]byte, 32)
	rand.Read(challenge)
	return ProofOfWorkChallenge{
		Challenge:  hex.EncodeToString(challenge),
		Difficulty: difficulty,
		Timestamp:  time.Now().Unix(),
	}
}

// VerifyProofOfWork checks if the provided solution is valid
func VerifyProofOfWork(challenge string, nonce string, difficulty int, timestamp int64) bool {
	// Check if the challenge is expired (e.g., valid for 1 minute)
	if time.Now().Unix()-timestamp > 60 {
		return false
	}
	data := challenge + nonce
	hash := sha256.Sum256([]byte(data))
	hashStr := hex.EncodeToString(hash[:])
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hashStr, prefix)
}

var submissionCount int
var submissionCountMutex sync.Mutex

// IncrementSubmissionCount increments the submission count
func IncrementSubmissionCount() {
	submissionCountMutex.Lock()
	defer submissionCountMutex.Unlock()
	submissionCount++
}

// ResetSubmissionCount resets the submission count
func ResetSubmissionCount() {
	submissionCountMutex.Lock()
	defer submissionCountMutex.Unlock()
	submissionCount = 0
}

// GetSubmissionCount returns the current submission count
func GetSubmissionCount() int {
	submissionCountMutex.Lock()
	defer submissionCountMutex.Unlock()
	return submissionCount
}

// Periodically reset the submission count (e.g., every minute)
func StartSubmissionCountReset() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			<-ticker.C
			ResetSubmissionCount()
		}
	}()
}

// GetDynamicDifficulty calculates the difficulty based on the current load
func GetDynamicDifficulty() int {
	count := GetSubmissionCount()
	if count > 100 {
		return 6 // High load, increase difficulty
	} else if count > 50 {
		return 5 // Medium load, moderate difficulty
	}
	return 4 // Low load, default difficulty
}

func main() {
	// gin.SetMode(gin.ReleaseMode) // Uncomment this line to disable debug mode

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}

	privateKeyData := os.Getenv("PRIVATE_KEY")
	privateKey, err := LoadPrivateKey(privateKeyData)
	if err != nil {
		panic(err)
	}

	secretKey := os.Getenv("KEY")

	client, err := connectDB()
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.Background())

	// admin commands?

	r := gin.Default()
	r.Use(cors.Default()) // Enable CORS for all routes

	// Rate limiter configuration: max 10 requests per 5s per IP
	limiter := tollbooth.NewLimiter(10, &limiter.ExpirableOptions{DefaultExpirationTTL: 5 * time.Second})

	StartSubmissionCountReset()

	// Middleware to apply the rate limiter
	r.Use(func(c *gin.Context) {
		httpError := tollbooth.LimitByRequest(limiter, c.Writer, c.Request)
		if httpError != nil {
			c.JSON(httpError.StatusCode, gin.H{"error": httpError.Message})
			c.Abort()
			return
		}
		c.Next()
	})

	r.GET("/api/model-list", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"models": MODELS})
	})

	r.GET("/api/benchmark/:submissionid", func(c *gin.Context) {
		submissionID := c.Param("submissionid")
		collection := client.Database("ollamark_db").Collection("benchmarks")

		var benchmark BenchmarkResult
		err := collection.FindOne(context.Background(), bson.M{"submissionid": submissionID}).Decode(&benchmark)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Benchmark not found"})
			return
		}

		c.JSON(http.StatusOK, benchmark)
	})

	r.GET("/api/pow-challenge", func(c *gin.Context) {
		challenge := GenerateProofOfWorkChallenge()
		c.JSON(http.StatusOK, challenge)
	})

	r.GET("/api/benchmarks", func(c *gin.Context) {
		sortBy := c.DefaultQuery("sort_by", "timestamp")
		order := c.DefaultQuery("order", "desc")
		modelFilter := c.DefaultQuery("model", "")
		ollamaVersionFilter := c.DefaultQuery("ollama_version", "")
		osFilter := c.DefaultQuery("os", "")
		cpuFilter := c.DefaultQuery("cpu", "")
		gpuFilter := c.DefaultQuery("gpu", "")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

		var sortOrder int
		if order == "asc" {
			sortOrder = 1
		} else {
			sortOrder = -1
		}

		if limit == 0 {
			// Set a large limit value when limit is 0
			limit = 1000000 // Adjust this value according to your needs
		}

		filter := bson.M{}
		if modelFilter != "" {
			filter["modelname"] = modelFilter
		}
		if osFilter != "" {
			filter["sysinfo.os"] = bson.M{"$regex": osFilter, "$options": "i"}
		}
		if cpuFilter != "" {
			filter["sysinfo.cpuname"] = bson.M{"$regex": cpuFilter, "$options": "i"}
		}
		if gpuFilter != "" {
			filter["gpuinfo.name"] = bson.M{"$regex": gpuFilter, "$options": "i"}
		}
		if ollamaVersionFilter != "" {
			filter["ollamaversion"] = ollamaVersionFilter
		}

		benchmarks, total, err := fetchBenchmarks(client, filter, sortBy, sortOrder, page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"benchmarks": benchmarks, "total": total})
	})

	r.POST("/api/submit-benchmark", authMiddleware(), func(c *gin.Context) {
		encryptedData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
			fmt.Printf("Invalid request payload: %v", err)
			return
		}

		submissionID := c.GetHeader("X-Submission-ID")
		signature := c.GetHeader("X-Signature")

		if !verifySignature(submissionID, signature, secretKey) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			fmt.Printf("Invalid signature: %v", err)
			return
		}

		// Check for replay attacks by storing and checking used submission IDs
		isUnique, err := checkSubmissionID(client, submissionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check submission"})
			fmt.Printf("Failed to check submission ID: %v", err)
			return
		}

		if !isUnique {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not a unique submission"})
			return
		}

		var payload map[string]string
		if err := json.Unmarshal(encryptedData, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload format"})
			fmt.Printf("Invalid payload format: %v", err)
			return
		}

		encryptedAESKey, _ := base64.StdEncoding.DecodeString(payload["encrypted_key"])
		nonce, _ := base64.StdEncoding.DecodeString(payload["nonce"])
		ciphertext, _ := base64.StdEncoding.DecodeString(payload["data"])

		// Decrypt AES key with RSA private key
		aesKey, err := DecryptData(privateKey, encryptedAESKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Decryption failed"})
			fmt.Printf("Decryption failed: %v", err)
			return
		}

		// Decrypt data with AES key
		decryptedData, err := decryptAESGCM(aesKey, nonce, ciphertext)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Decryption failed"})
			fmt.Printf("Decryption failed: %v", err)
			return
		}

		var benchmarkResult BenchmarkResult
		if err := json.Unmarshal(decryptedData, &benchmarkResult); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid benchmark data"})
			fmt.Printf("Invalid benchmark data: %v", err)
			return
		}

		// Basic verification of benchmark data
		if benchmarkResult.EvalCount <= 0 || benchmarkResult.TokensPerSecond <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid benchmark metrics"})
			return
		}

		// Validate the modelName against the predefined list
		if !contains(MODELS, benchmarkResult.ModelName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model name"})
			return
		}

		// Verify proof-of-work
		if !VerifyProofOfWork(benchmarkResult.ProofOfWork.Challenge, benchmarkResult.ProofOfWork.Nonce, benchmarkResult.ProofOfWork.Difficulty, benchmarkResult.ProofOfWork.Timestamp) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid proof-of-work solution"})
			return
		}

		checkedIP := checkIP(benchmarkResult.IP)
		if !checkedIP {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "IP address is rate limited"})
			return
		}

		log.Println("Benchmark was received successfully:", benchmarkResult)
		log.Printf("SysInfo: %+v\n", *benchmarkResult.SysInfo)
		log.Printf("GPUInfo: %+v\n", *benchmarkResult.GPUInfo)
		benchmarkResult.SubmissionID = submissionID

		// Insert benchmarks into the MongoDB
		err = insertBenchmark(client, benchmarkResult)
		if err != nil {
			fmt.Printf("Failed to insert benchmark: %v", err)
			return
		}

		IncrementSubmissionCount()

		c.JSON(http.StatusOK, gin.H{"message": "Benchmark submitted successfully"})
	})

	port := ":3333"
	log.Printf("Ollamark Server is running on port %s\n", port)
	if err := r.Run(port); err != nil {
		log.Printf("Failed to start Ollamark server: %v\n", err)
	}
}
