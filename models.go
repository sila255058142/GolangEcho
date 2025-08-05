package main
import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v4"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID          int      `json:"id"`
	Username    string   `json:"username"`
	Password    string   `json:"password,omitempty"`
	Firstname   string   `json:"firstname"`
	Lastname    string   `json:"lastname"`
	Age         int      `json:"age"`
	Gender      string   `json:"gender"`
	Description string   `json:"description"`
	Interest    []string `json:"interest"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type JWTClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}