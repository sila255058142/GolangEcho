package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
) // CRUD
func createUser(c echo.Context) error {
	var user User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if user.Username == "" || user.Password == "" || user.Firstname == "" || user.Lastname == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Please enter all fields"})
	}

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
	}

	interestsJSON, err := json.Marshal(user.Interest)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to call  interests"})
	}

	result, err := db.Exec(
		"INSERT INTO users (username, password, firstname, lastname, age, gender, description, interest) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		user.Username, hashedPassword, user.Firstname, user.Lastname, user.Age, user.Gender, user.Description, string(interestsJSON),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed create user"})
	}

	id, err := result.LastInsertId()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to call new user ID"})
	}
	user.ID = int(id)
	user.Password = ""
	
	return c.JSON(http.StatusCreated, user)
}

func getAllUsers(c echo.Context) error {
	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Database not connected"})
	}

	rows, err := db.Query("SELECT * FROM users")
	if err != nil {
		log.Printf("Error querying users: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve users"})
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var interestJSONString sql.NullString

		err := rows.Scan(&user.ID, &user.Firstname, &user.Lastname, &user.Age, &user.Gender, &user.Description, &interestJSONString)
		if err != nil {
			log.Printf("Error scanning user row: %v", err)
			continue
		}

		if interestJSONString.Valid {
			if err := json.Unmarshal([]byte(interestJSONString.String), &user.Interest); err != nil {
				log.Printf("Error unmarshalling interests for user ID %d: %v", user.ID, err)
				user.Interest = []string{}
			}
		} else {
			user.Interest = []string{}
		}

		users = append(users, user)
	}

	return c.JSON(http.StatusOK, users)
}

func getUserByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Database not connected"})
	}

	var user User
	var interestJSONString sql.NullString

	err = db.QueryRow("SELECT * FROM users WHERE id = ?", id).
		Scan(&user.ID, &user.Firstname, &user.Lastname, &user.Age, &user.Gender, &user.Description, &interestJSONString)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		log.Printf("Error querying user by ID %d: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to call user"})
	}

	if interestJSONString.Valid {
		if err := json.Unmarshal([]byte(interestJSONString.String), &user.Interest); err != nil {
			log.Printf("Error unmarshalling interests for user ID %d: %v", user.ID, err)
			user.Interest = []string{}
		}
	} else {
		user.Interest = []string{}
	}

	return c.JSON(http.StatusOK, user)
}

func updateUser(c echo.Context) error {
	var user User
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Database not connected"})
	}

	if err := c.Bind(&user); err != nil {
		log.Printf("Error binding user data for update: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Incorrect information provided"})
	}

	if user.Firstname == "" || user.Lastname == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "First name and last name cannot empty pls"})
	}

	user.ID = id

	interestsJSON, err := json.Marshal(user.Interest)
	if err != nil {
		log.Printf("Error converting interests to JSON for update: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to interests for update"})
	}

	result, err := db.Exec(
		"UPDATE users SET firstname=?, lastname=?, age=?, gender=?, description=?, interest=? WHERE id=?",
		user.Firstname, user.Lastname, user.Age, user.Gender, user.Description, string(interestsJSON), id,
	)
	if err != nil {
		log.Printf("Error updating user ID %d: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected during update: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Unable to verify update results"})
	}

	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User for update was not found."})
	}

	log.Printf("Update user success: %+v", user)
	return c.JSON(http.StatusOK, user)
}

func deleteUser(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	if db == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Database not connected"})
	}

	result, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		log.Printf("Error deleting user ID %d: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete user"})
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected during delete: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Unable to verify delete results"})
	}

	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User with specified ID not found"})
	}

	log.Printf("Delete user success: %d", id)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "Delete successful",
		"deleted_id": id,
	})
}
