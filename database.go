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
	"golang.org/x/crypto/bcrypt")


// Database functions
func initDatabase() {
	dsn := "root:@tcp(127.0.0.1:3306)/goechodatabase?charset=utf8mb4&parseTime=True&loc=Local"
	var err error

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("Warning: ไม่สามารถเปิดการเชื่อมต่อฐานข้อมูลได้: %v", err)
		log.Println("Server will run without database connection (test mode)")
		db = nil
		return
	}

	if err := db.Ping(); err != nil {
		log.Printf("Warning: ไม่สามารถเชื่อมต่อ MySQL ได้ (Ping ไม่สำเร็จ): %v", err)
		db.Close()
		db = nil
		return
	}

	fmt.Println("เชื่อมต่อ MySQL สำเร็จ!")
}

func closeDatabase() {
	if db != nil {
		db.Close()
	}
}