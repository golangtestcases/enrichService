package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	_ "people-service/docs"
	"people-service/internal/api"
	"people-service/internal/db"
	"people-service/internal/enrich"
	"people-service/internal/models"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	dbConn   *gorm.DB
	validate *validator.Validate
)

func main() {
	// Инициализация Redis
	if err := initRedis(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Подключение к БД
	if err := initDatabase(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Инициализация валидатора
	validate = validator.New()

	// Роутер
	r := setupRouter()

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func initRedis() error {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "redis"
	}
	return enrich.InitRedis(redisHost + ":6379")
}

func initDatabase() error {
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
		return err
	}

	// Миграции
	if err := dbConn.AutoMigrate(&models.Person{}); err != nil {
		log.Printf("Migration error: %v", err)
		return err
	}
	return nil

}

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(api.ErrorMiddleware())

	r.POST("/people", createPerson)
	r.GET("/people", getPeople)
	r.PUT("/people/:id", updatePerson)
	r.DELETE("/people/:id", deletePerson)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

// @Summary Добавить человека
// @Description Создает новую запись с обогащением данных
// @Tags people
// @Accept json
// @Produce json
// @Param input body models.Person true "Данные человека"
// @Success 201 {object} models.Person
// @Failure 400 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /people [post]
func createPerson(c *gin.Context) {
	var person models.Person
	if err := c.ShouldBindJSON(&person); err != nil {
		api.HandleError(c, api.NewError(
			api.ErrorTypeValidation,
			http.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if err := validate.Struct(person); err != nil {
		api.HandleError(c, err)
		return
	}

	if err := person.Enrich(); err != nil {
		api.HandleError(c, api.NewError(
			api.ErrorTypeExternal,
			http.StatusFailedDependency,
			"Failed to enrich data",
			err.Error(),
		))
		return
	}

	if err := dbConn.Create(&person).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			api.HandleError(c, api.NewError(
				api.ErrorTypeConflict,
				http.StatusConflict,
				"Person already exists",
				nil,
			))
		} else {
			api.HandleError(c, api.ErrDBOperation)
		}
		return
	}

	c.JSON(http.StatusCreated, person)
}

// @Summary Получить список людей
// @Description Возвращает список с возможностью фильтрации и пагинацией
// @Tags people
// @Accept json
// @Produce json
// @Param name query string false "Фильтр по имени"
// @Param surname query string false "Фильтр по фамилии"
// @Param age query int false "Фильтр по возрасту"
// @Param gender query string false "Фильтр по полу"
// @Param nationality query string false "Фильтр по национальности"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Лимит записей" default(10)
// @Success 200 {object} models.PeopleListResponse
// @Failure 400 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /people [get]
func getPeople(c *gin.Context) {
	var filter models.PersonFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		api.HandleError(c, api.NewError(
			api.ErrorTypeValidation,
			http.StatusBadRequest,
			"Invalid query parameters",
			err.Error(),
		))
		return
	}

	people, total, err := models.GetPeople(dbConn, filter)
	if err != nil {
		api.HandleError(c, api.ErrDBOperation)
		return
	}

	response := models.PeopleListResponse{
		Data:  people,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Обновить данные человека
// @Tags people
// @Accept json
// @Produce json
// @Param id path int true "ID человека"
// @Param input body models.Person true "Обновленные данные"
// @Success 200 {object} models.Person
// @Failure 400 {object} api.ErrorResponse
// @Failure 404 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /people/{id} [put]
func updatePerson(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		api.HandleError(c, api.NewError(
			api.ErrorTypeValidation,
			http.StatusBadRequest,
			"Invalid ID format",
			nil,
		))
		return
	}

	var person models.Person
	if err := dbConn.First(&person, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.HandleError(c, api.ErrNotFound)
		} else {
			api.HandleError(c, api.ErrDBOperation)
		}
		return
	}

	if err := c.ShouldBindJSON(&person); err != nil {
		api.HandleError(c, api.NewError(
			api.ErrorTypeValidation,
			http.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if err := validate.Struct(person); err != nil {
		api.HandleError(c, err)
		return
	}

	if err := dbConn.Save(&person).Error; err != nil {
		api.HandleError(c, api.ErrDBOperation)
		return
	}

	c.JSON(http.StatusOK, person)
}

// @Summary Удалить человека
// @Tags people
// @Accept json
// @Produce json
// @Param id path int true "ID человека"
// @Success 200 {object} map[string]string
// @Failure 404 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /people/{id} [delete]
func deletePerson(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		api.HandleError(c, api.NewError(
			api.ErrorTypeValidation,
			http.StatusBadRequest,
			"Invalid ID format",
			nil,
		))
		return
	}

	result := dbConn.Delete(&models.Person{}, id)
	if result.Error != nil {
		api.HandleError(c, api.ErrDBOperation)
		return
	}

	if result.RowsAffected == 0 {
		api.HandleError(c, api.ErrNotFound)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Person deleted successfully"})
}
