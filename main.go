package main

import (
	"log"
	"net/http"
	"os"
	"people-service/internal/db"
	"people-service/internal/enrich"
	"people-service/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var dbConn *gorm.DB

func main() {
	// Инициализация Redis
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "redis"
	}

	if err := enrich.InitRedis(redisHost + ":6379"); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Подключение к БД
	var err error
	for i := 0; i < 5; i++ {
		dbConn, err = db.InitDB()
		if err == nil {
			break
		}
		log.Printf("DB connection attempt %d failed: %v", i+1, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Миграции
	if err := dbConn.AutoMigrate(&models.Person{}); err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Роутер
	r := gin.Default()
	r.POST("/people", createPerson)
	r.GET("/people", getPeople)
	r.PUT("/people/:id", updatePerson)
	r.DELETE("/people/:id", deletePerson)

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func createPerson(c *gin.Context) {
	var person models.Person
	if err := c.ShouldBindJSON(&person); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := person.Enrich(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enrich data: " + err.Error()})
		return
	}

	if err := dbConn.Create(&person).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, person)
}

func getPeople(c *gin.Context) {
	var people []models.Person
	if err := dbConn.Find(&people).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, people)
}

//Обновление существующей записи

func updatePerson(c *gin.Context) {
	id := c.Param("id") // Получаем ID из URL
	var person models.Person

	// 1. Находим запись в БД
	if err := dbConn.First(&person, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	// 2. Парсим обновленные данные из JSON
	if err := c.ShouldBindJSON(&person); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. Сохраняем обновленные данные
	if err := dbConn.Save(&person).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, person)
}

// Удаление записи по ID
func deletePerson(c *gin.Context) {
	id := c.Param("id")

	// 1. Пытаемся удалить
	result := dbConn.Delete(&models.Person{}, id)

	// 2. Проверяем, была ли удалена запись
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Person deleted"})
}
