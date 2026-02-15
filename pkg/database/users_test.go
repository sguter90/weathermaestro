package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// generateRandomString creates a random string of specified length
func generateRandomString(length int) string {
	bytes := make([]byte, length/2+1)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)[:length]
}

func TestCreateUser(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "testuser_" + generateRandomString(8)
	password := "SecurePassword123!"

	user, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify user was created with ID and timestamps
	if user.ID == uuid.Nil {
		t.Error("Expected user ID to be set")
	}

	if user.Username != username {
		t.Errorf("Expected username=%s, got %s", username, user.Username)
	}

	if user.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	// Verify password was hashed (not stored in plain text)
	// We can't directly check the hash, but we can validate the user
	validatedUser, err := dm.ValidateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to validate user: %v", err)
	}

	if validatedUser.ID != user.ID {
		t.Errorf("Expected validated user ID=%s, got %s", user.ID, validatedUser.ID)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "duplicate_user_" + generateRandomString(8)
	password := "SecurePassword123!"

	// Create first user
	_, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// Try to create duplicate user
	_, err = dm.CreateUser(ctx, username, password)
	if err == nil {
		t.Error("Expected error when creating duplicate user")
	}
}

func TestCreateUser_EmptyUsername(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	password := "SecurePassword123!"

	_, err := dm.CreateUser(ctx, "", password)
	if err == nil {
		t.Error("Expected error when creating user with empty username")
	}
}

func TestValidateUser_Success(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "validuser_" + generateRandomString(8)
	password := "CorrectPassword123!"

	// Create user
	createdUser, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Validate with correct password
	validatedUser, err := dm.ValidateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to validate user: %v", err)
	}

	if validatedUser.ID != createdUser.ID {
		t.Errorf("Expected user ID=%s, got %s", createdUser.ID, validatedUser.ID)
	}

	if validatedUser.Username != username {
		t.Errorf("Expected username=%s, got %s", username, validatedUser.Username)
	}
}

func TestValidateUser_WrongPassword(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "testuser_" + generateRandomString(8)
	correctPassword := "CorrectPassword123!"
	wrongPassword := "WrongPassword456!"

	// Create user
	_, err := dm.CreateUser(ctx, username, correctPassword)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Try to validate with wrong password
	_, err = dm.ValidateUser(ctx, username, wrongPassword)
	if err == nil {
		t.Error("Expected error when validating with wrong password")
	}

	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials' error, got: %v", err)
	}
}

func TestValidateUser_NonExistentUser(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "nonexistent_" + generateRandomString(8)
	password := "SomePassword123!"

	// Try to validate non-existent user
	_, err := dm.ValidateUser(ctx, username, password)
	if err == nil {
		t.Error("Expected error when validating non-existent user")
	}

	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials' error, got: %v", err)
	}
}

func TestValidateUser_EmptyPassword(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "testuser_" + generateRandomString(8)
	password := "CorrectPassword123!"

	// Create user
	_, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Try to validate with empty password
	_, err = dm.ValidateUser(ctx, username, "")
	if err == nil {
		t.Error("Expected error when validating with empty password")
	}

	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials' error, got: %v", err)
	}
}

func TestValidateUser_CaseSensitiveUsername(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "TestUser_" + generateRandomString(8)
	password := "Password123!"

	// Create user with specific case
	_, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Try to validate with different case
	_, err = dm.ValidateUser(ctx, "testuser_"+username[9:], password)
	if err == nil {
		t.Error("Expected error when username case doesn't match")
	}
}

func TestPasswordHashing(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "hashtest_" + generateRandomString(8)
	password := "TestPassword123!"

	// Create user
	user, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Query the password hash directly from database
	query := `SELECT password_hash FROM users WHERE id = $1`
	var passwordHash string
	err = dm.QueryRowWithHealthCheck(ctx, query, user.ID).Scan(&passwordHash)
	if err != nil {
		t.Fatalf("Failed to query password hash: %v", err)
	}

	// Verify password is hashed (not plain text)
	if passwordHash == password {
		t.Error("Password should be hashed, not stored in plain text")
	}

	// Verify new format has v2: prefix
	if !strings.HasPrefix(passwordHash, "v2:") {
		t.Error("Password hash should have v2: prefix for new format")
	}

	// Remove prefix for bcrypt verification
	actualHash := strings.TrimPrefix(passwordHash, "v2:")

	// Verify hash is valid bcrypt hash with SHA-256 pre-hashing
	preHashedPassword := hashPasswordForTest(password)
	err = bcrypt.CompareHashAndPassword([]byte(actualHash), []byte(preHashedPassword))
	if err != nil {
		t.Error("Password hash should be valid bcrypt hash with SHA-256 pre-hashing")
	}

	// Verify hash starts with bcrypt prefix (after removing v2:)
	if len(actualHash) < 4 || (actualHash[:4] != "$2a$" && actualHash[:4] != "$2b$" && actualHash[:4] != "$2y$") {
		t.Error("Password hash should start with bcrypt prefix after v2: prefix")
	}
}

// hashPasswordForTest is a test helper that mirrors the production hashPassword function
func hashPasswordForTest(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func TestCreateUser_LongPassword(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "longpass_" + generateRandomString(8)
	// bcrypt has a 72 byte limit, test with longer password
	password := generateRandomString(100)

	user, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user with long password: %v", err)
	}

	// Verify user can be validated
	validatedUser, err := dm.ValidateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to validate user with long password: %v", err)
	}

	if validatedUser.ID != user.ID {
		t.Error("User validation failed for long password")
	}
}

func TestCreateUser_SpecialCharactersInUsername(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "user@example.com_" + generateRandomString(8)
	password := "Password123!"

	_, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user with special characters: %v", err)
	}

	// Verify user can be validated
	validatedUser, err := dm.ValidateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to validate user with special characters: %v", err)
	}

	if validatedUser.Username != username {
		t.Errorf("Expected username=%s, got %s", username, validatedUser.Username)
	}
}

func TestCreateUser_SpecialCharactersInPassword(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "testuser_" + generateRandomString(8)
	password := "P@ssw0rd!#$%^&*()_+-=[]{}|;:',.<>?/~`"

	user, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user with special characters in password: %v", err)
	}

	// Verify user can be validated
	validatedUser, err := dm.ValidateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to validate user with special characters in password: %v", err)
	}

	if validatedUser.ID != user.ID {
		t.Error("User validation failed for password with special characters")
	}
}

func TestCreateUser_UnicodeCharacters(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "user_日本語_" + generateRandomString(8)
	password := "Пароль123!密码"

	_, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user with unicode characters: %v", err)
	}

	// Verify user can be validated
	validatedUser, err := dm.ValidateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to validate user with unicode characters: %v", err)
	}

	if validatedUser.Username != username {
		t.Errorf("Expected username=%s, got %s", username, validatedUser.Username)
	}
}

func TestValidateUser_EmptyUsername(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	password := "Password123!"

	// Try to validate with empty username
	_, err := dm.ValidateUser(ctx, "", password)
	if err == nil {
		t.Error("Expected error when validating with empty username")
	}

	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials' error, got: %v", err)
	}
}

func TestValidateUser_BothEmpty(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()

	// Try to validate with both empty
	_, err := dm.ValidateUser(ctx, "", "")
	if err == nil {
		t.Error("Expected error when validating with empty username and password")
	}

	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials' error, got: %v", err)
	}
}

func TestCreateUser_WhitespaceUsername(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "  user with spaces  "
	password := "Password123!"

	_, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user with whitespace: %v", err)
	}

	// Verify user can be validated with exact username (including spaces)
	validatedUser, err := dm.ValidateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to validate user with whitespace: %v", err)
	}

	if validatedUser.Username != username {
		t.Errorf("Expected username='%s', got '%s'", username, validatedUser.Username)
	}

	// Verify validation fails without spaces
	_, err = dm.ValidateUser(ctx, "user with spaces", password)
	if err == nil {
		t.Error("Expected error when validating without leading/trailing spaces")
	}
}

func TestCreateUser_VeryLongUsername(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := generateRandomString(200)
	password := "Password123!"

	user, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user with long username: %v", err)
	}

	// Verify user can be validated
	validatedUser, err := dm.ValidateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to validate user with long username: %v", err)
	}

	if validatedUser.ID != user.ID {
		t.Error("User validation failed for long username")
	}
}

func TestPasswordHashConsistency(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	username := "hashconsistency_" + generateRandomString(8)
	password := "TestPassword123!"

	// Create user
	user, err := dm.CreateUser(ctx, username, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Query hash twice
	query := `SELECT password_hash FROM users WHERE id = $1`

	var hash1 string
	err = dm.QueryRowWithHealthCheck(ctx, query, user.ID).Scan(&hash1)
	if err != nil {
		t.Fatalf("Failed to query password hash first time: %v", err)
	}

	var hash2 string
	err = dm.QueryRowWithHealthCheck(ctx, query, user.ID).Scan(&hash2)
	if err != nil {
		t.Fatalf("Failed to query password hash second time: %v", err)
	}

	// Hashes should be identical (not regenerated)
	if hash1 != hash2 {
		t.Error("Password hash should be consistent across queries")
	}
}

func TestMultipleUsersWithSamePassword(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	password := "SharedPassword123!"
	username1 := "user1_" + generateRandomString(8)
	username2 := "user2_" + generateRandomString(8)

	// Create two users with same password
	user1, err := dm.CreateUser(ctx, username1, password)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}

	user2, err := dm.CreateUser(ctx, username2, password)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Query both hashes
	query := `SELECT password_hash FROM users WHERE id = $1`

	var hash1 string
	err = dm.QueryRowWithHealthCheck(ctx, query, user1.ID).Scan(&hash1)
	if err != nil {
		t.Fatalf("Failed to query user1 hash: %v", err)
	}

	var hash2 string
	err = dm.QueryRowWithHealthCheck(ctx, query, user2.ID).Scan(&hash2)
	if err != nil {
		t.Fatalf("Failed to query user2 hash: %v", err)
	}

	// Hashes should be different (bcrypt uses random salt)
	if hash1 == hash2 {
		t.Error("Password hashes should be different even for same password (due to salt)")
	}

	// Both users should validate successfully
	_, err = dm.ValidateUser(ctx, username1, password)
	if err != nil {
		t.Errorf("Failed to validate user1: %v", err)
	}

	_, err = dm.ValidateUser(ctx, username2, password)
	if err != nil {
		t.Errorf("Failed to validate user2: %v", err)
	}
}

func TestValidateUser_SQLInjectionAttempt(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()

	// Try SQL injection in username
	maliciousUsername := "admin' OR '1'='1"
	password := "password"

	_, err := dm.ValidateUser(ctx, maliciousUsername, password)
	if err == nil {
		t.Error("Expected error for SQL injection attempt")
	}

	// Should return invalid credentials, not SQL error
	if err.Error() != "invalid credentials" {
		t.Errorf("Expected 'invalid credentials' error, got: %v", err)
	}
}

func TestCreateUser_ConcurrentCreation(t *testing.T) {
	dm := setupTestDatabaseManager(t)
	if dm == nil {
		t.Skip("Skipping test that requires real database connection")
	}
	defer dm.Close()

	ctx := context.Background()
	baseUsername := "concurrent_" + generateRandomString(8)
	password := "Password123!"

	// Try to create same user concurrently
	done := make(chan error, 2)

	for i := 0; i < 2; i++ {
		go func() {
			_, err := dm.CreateUser(ctx, baseUsername, password)
			done <- err
		}()
	}

	// Collect results
	var successCount, errorCount int
	for i := 0; i < 2; i++ {
		err := <-done
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	// Exactly one should succeed
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful creation, got %d", successCount)
	}

	if errorCount != 1 {
		t.Errorf("Expected exactly 1 failed creation, got %d", errorCount)
	}
}
