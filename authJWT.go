package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateJWT(userID int, username string) (string, error) {
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func jwtMiddleware() echo.MiddlewareFunc {
	config := echojwt.Config{
		SigningKey: jwtSecret,
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(JWTClaims)
		},
	}
	return echojwt.WithConfig(config)
}
func register(c echo.Context) error {
	var user User
	if err := c.Bind(&user); err != nil {
		log.Printf("Register: Error binding user data: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"status":  "error",
			"message": "Invalid body",
		})
	}

	if user.Username == "" || user.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"status":  "error",
			"message": "put Username and password ",
		})
	}

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		log.Printf("Register: Error hash password: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"status":  "error",
			"message": "Failed hash password",
		})
	}

	var count int
	if db != nil {
		err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", user.Username).Scan(&count)
		if err != nil {
			log.Printf("Register: Error checking username: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": "Failed to check username",
			})
		}
	}

	if count > 0 {
		return c.JSON(http.StatusConflict, map[string]string{
			"status":  "error",
			"message": "Username is Exists",
		})
	}

	interestsJSON, err := json.Marshal(user.Interest)
	if err != nil {
		log.Printf("Register: Error marshall interests to JSON: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"status":  "error",
			"message": "Failed to interests",
		})
	}
	var id int64
	if db != nil {
		result, err := db.Exec("INSERT INTO users (username, password, firstname, lastname, age, gender, description, interest) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			user.Username, hashedPassword, user.Firstname, user.Lastname, user.Age, user.Gender, user.Description, string(interestsJSON))
		if err != nil {
			log.Printf("Register: Error inserting new user: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": "Failed to register",
			})
		}

		id, err = result.LastInsertId()
		if err != nil {
			log.Printf("Register: Error getting last insert ID: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": "Failed callto new user ID",
			})
		}
	} else {
		id = 999
		log.Println("Register: Running in test mode. User not saved to database.")
	}

	user.ID = int(id)
	user.Password = ""

	log.Printf("Register: User registered successfully: %s", user.Username)
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Registration successful",
		"data":    user,
	})
}

func login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Login: Error binding request data: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"status":  "error",
			"message": "Username and password are required",
		})
	}

	var user User
	var hashedPassword string
	var interestJSONString sql.NullString

	if db == nil {
		if req.Username == "admin" && req.Password == "admin" {
			token, _ := generateJWT(1, "admin")
			mockUser := User{ID: 1, Username: "admin", Firstname: "Admin", Lastname: "User", Interest: []string{"testing"}}
			return c.JSON(http.StatusOK, map[string]interface{}{
				"status":  "success",
				"data": LoginResponse{
					Token: token,
					User:  mockUser,
				},
			})
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"status":  "error",
			"message": "username or password is incorrect",
		})
	}

	err := db.QueryRow("SELECT id, username, password, firstname, lastname, age, gender, description, interest FROM users WHERE username = ?", req.Username).
		Scan(&user.ID, &user.Username, &hashedPassword, &user.Firstname, &user.Lastname, &user.Age, &user.Gender, &user.Description, &interestJSONString)

	if err == sql.ErrNoRows {
		log.Printf("Login: User not found: %s", req.Username)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"status":  "error",
			"message": "Invalid username or password",
		})
	}
	if err != nil {
		log.Printf("Login: Error querying user: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"status":  "error",
			"message": "Failed to verify credentials",
		})
	}

	if !checkPasswordHash(req.Password, hashedPassword) {
		log.Printf("Login: Invalid password for user: %s", req.Username)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"status":  "error",
			"message": "Invalid username or password",
		})
	}

	if interestJSONString.Valid {
		if err := json.Unmarshal([]byte(interestJSONString.String), &user.Interest); err != nil {
			log.Printf("Login: Error unmarshalling interests for user %s: %v", user.Username, err)
			user.Interest = []string{}
		}
	} else {
		user.Interest = []string{}
	}

	token, err := generateJWT(user.ID, user.Username)
	if err != nil {
		log.Printf("Login: Error generating JWT for user %s: %v", user.Username, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"status":  "error",
			"message": "Failed to generate token",
		})
	}

	user.Password = ""

	log.Printf("Login: User logged in successfully: %s", user.Username)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Login successful",
		"data": LoginResponse{
			Token: token,
			User:  user,
		},
	})
}

func getProfile(c echo.Context) error {
	userToken := c.Get("user").(*jwt.Token)
	claims := userToken.Claims.(*JWTClaims)

	var userData User
	var interestJSONString sql.NullString

	if db == nil {
		mockUser := User{ID: claims.UserID, Username: claims.Username, Firstname: "Test", Lastname: "User", Interest: []string{"test_interest"}}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "success",
			"data":   mockUser,
		})
	}

	err := db.QueryRow("SELECT id, username, firstname, lastname, age, gender, description, interest FROM users WHERE id = ?", claims.UserID).
		Scan(&userData.ID, &userData.Username, &userData.Firstname, &userData.Lastname, &userData.Age, &userData.Gender, &userData.Description, &interestJSONString)

	if err == sql.ErrNoRows {
		log.Printf("GetProfile: User not found from token ID %d", claims.UserID)
		return c.JSON(http.StatusNotFound, map[string]string{
			"status":  "error",
			"message": "User profile not found",
		})
	}
	if err != nil {
		log.Printf("GetProfile: Error querying user by ID %d: %v", claims.UserID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"status":  "error",
			"message": "Failed to call user profile",
		})
	}

	if interestJSONString.Valid {
		if err := json.Unmarshal([]byte(interestJSONString.String), &userData.Interest); err != nil {
			log.Printf("GetProfile: Error unmarshalling interests for user ID %d: %v", userData.ID, err)
			userData.Interest = []string{}
		}
	} else {
		userData.Interest = []string{}
	}

	userData.Password = ""

	log.Printf("GetProfile: User profile retrieved for %s (ID: %d)", userData.Username, userData.ID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   userData,
	})
}
