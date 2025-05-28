package main

import (
	"log"
	"net/http"
	"os"
	_ "people-service/docs"
	"people-service/internal/db"
	"people-service/internal/enrich"
	"people-service/internal/models"
	"strconv"
	"time"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// @title People Service API
// @version 1.0
// @description API для управления данными людей с обогащением из внешних источников

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
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

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("Server failed:", err)
	}
}

// @Summary Добавить человека
// @Description Создает новую запись с обогащением данных
// @Tags people
// @Accept json
// @Produce json
// @Param input body models.Person true "Данные человека"
// @Success 201 {object} models.Person
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /people [post]
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

// @Summary Получить список людей
// @Description Возвращает список с возможностью фильтрации и пагинации
// @Tags people
// @Accept json
// @Produce json
// @Param name query string false "Фильтр по имени"
// @Param surname query string false "Фильтр по фамилии"
// @Param age query int false "Фильтр по возрасту"
// @Param gender query string false "Фильтр по полу"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Лимит записей" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /people [get]
func getPeople(c *gin.Context) {
	var people []models.Person
	query := dbConn.Model(&models.Person{})

	// Фильтрация по query-параметрам
	if name := c.Query("name"); name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%") // Поиск по частичному совпадению
	}
	if surname := c.Query("surname"); surname != "" {
		query = query.Where("surname ILIKE ?", "%"+surname+"%")
	}
	if age := c.Query("age"); age != "" {
		query = query.Where("age = ?", age)
	}
	if gender := c.Query("gender"); gender != "" {
		query = query.Where("gender = ?", gender)
	}

	// Пагинация
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	if err := query.Offset(offset).Limit(limit).Find(&people).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  people,
		"page":  page,
		"limit": limit,
	})
}

func updatePerson(c *gin.Context) {
	id := c.Param("id")
	var person models.Person

	if err := dbConn.First(&person, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	if err := c.ShouldBindJSON(&person); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dbConn.Save(&person).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, person)
}

// @Summary Обновить данные человека
// @Tags people
// @Accept json
// @Produce json
// @Param id path int true "ID человека"
// @Param input body models.Person true "Обновленные данные"
// @Success 200 {object} models.Person
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /people/{id} [put]
func deletePerson(c *gin.Context) {
	id := c.Param("id")

	result := dbConn.Delete(&models.Person{}, id)

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
