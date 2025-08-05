package main

import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var db *sql.DB
var jwtSecret = []byte("your-super-secret-key-that-should-be-long-and-random")

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
// cors middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
	}))

	initDatabase()
	defer closeDatabase()

	// check status server
	e.GET("/", func(c echo.Context) error {
		status := "disconnected"
		if db != nil {
			status = "connected"
		}
		return c.JSON(http.StatusOK, map[string]string{
			"status":   "Server is running",
			"database": status,
		})
	})

	// เราท์ส์ที่ไม่ต้องการ JWT
	e.POST("/api/register", register)
	e.POST("/api/login", login)

	// เราท์ส์ที่ต้องการ JWT
	protected := e.Group("/api")
	protected.Use(jwtMiddleware())

	protected.GET("/users", getAllUsers)
	protected.GET("/users/:id", getUserByID)
	protected.POST("/users", createUser)
	protected.PUT("/users/:id", updateUser)
	protected.DELETE("/users/:id", deleteUser)
	protected.GET("/profile", getProfile)

	protected.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Protected API Running and You have access!",
		})
	})

	// เริ่มเซิร์ฟเวอร์ โชว์ที่ Terminal
	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("PATH Endpoints (Unprotected):")
	fmt.Println(" 	POST 	/api/register 	= สมัครสมาชิก")
	fmt.Println(" 	POST 	/api/login 	 	= เข้าสู่ระบบ")
	fmt.Println("\nPATH Endpoints (Protected - requires JWT):")
	fmt.Println(" 	GET 	/api/users 	 	= ดูผู้ใช้ทั้งหมด")
	fmt.Println(" 	GET 	/api/users/:id 	= ดูผู้ใช้ตาม ID")
	fmt.Println(" 	POST 	/api/users 	 	= สร้างผู้ใช้ใหม่")
	fmt.Println(" 	PUT 	/api/users/:id 	= อัปเดตผู้ใช้")
	fmt.Println(" 	DELETE /api/users/:id 	= ลบผู้ใช้")
	fmt.Println(" 	GET 	/api/profile 	= ดูโปรไฟล์ผู้ใช้ปัจจุบัน")
	fmt.Println(" 	GET 	/api/test 	 = ทดสอบ Protected API")

	e.Logger.Fatal(e.Start(":8080"))
}
