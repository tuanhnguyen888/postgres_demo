package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/go-redis/redis"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/tuanhnguyen888/postgres_bai1/models"
	"github.com/tuanhnguyen888/postgres_bai1/storage"
	"gorm.io/gorm"
)

type Groups struct {
	ID            int     `json:"id"`
	GroupName     *string `json:"groupName"`
	ParentGroupID int     `json:"parentGroupID"`
}

type JsonResponse struct {
	Data   []Groups `json:"data"`
	Source string   `json:"source"`
}

type GroupsTree struct {
	Id         int     `json:"id"`
	GroupName  *string `json:"groupName"`
	GroupLevel int     `json:"groupLevel"`
	EmpId      string  `json:"emp_id"`
}

type Staffs struct {
	UserName  *string `json:"userName"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	GroupID   uint    `json:"groupID"`
}

type Repository struct {
	DB *gorm.DB
}

func (r *Repository) CreateGroup(context *fiber.Ctx) error {
	groups := Groups{}
	err := context.BodyParser(&groups)
	if err != nil {
		context.Status(http.StatusUnprocessableEntity).JSON(
			&fiber.Map{"message": "not request"})
		return err
	}

	err = r.DB.Create(&groups).Error
	if err != nil {
		context.Status(http.StatusBadRequest).JSON(
			&fiber.Map{"message": "could not create group"})
		return err
	}

	context.Status(http.StatusOK).JSON(
		&fiber.Map{
			"message": "group has been added",
		})
	return nil

}

func (r *Repository) UpdateGroup(context *fiber.Ctx) error {
	id := context.Params("id")
	group := Groups{}
	err := context.BodyParser(&group)
	if err != nil {
		context.Status(http.StatusUnprocessableEntity).JSON(
			&fiber.Map{"message": "not request"})
		return err
	}

	r.DB.Where("id = ?", id).Updates(&group)

	context.Status(http.StatusOK).JSON(
		&fiber.Map{
			"message": "group has been update",
		})
	return nil
}

func (r *Repository) CreateStaff(context *fiber.Ctx) error {
	staffs := Staffs{}
	err := context.BodyParser(&staffs)
	if err != nil {
		context.Status(http.StatusUnprocessableEntity).JSON(
			&fiber.Map{"message": "not request"})
		return err
	}

	err = r.DB.Create(&staffs).Error
	if err != nil {
		context.Status(http.StatusBadRequest).JSON(
			&fiber.Map{"message": "could not staff group"})
		return err
	}

	context.Status(http.StatusOK).JSON(
		&fiber.Map{
			"message": "group has been added",
		})
	return nil
}

// -------
func (r *Repository) GetGroupsByWidth(context *fiber.Ctx) error {

	groupsModels := []models.Groups{}
	r.DB.Find(&groupsModels)

	flatGroups := []models.Groups{}
	rootGroups := []models.Groups{}
	r.DB.Where("parent_group_id = 0").Find(&rootGroups)
	flatGroups = append(flatGroups, rootGroups...)
	for _, group := range groupsModels {
		tempGroup := []models.Groups{}
		r.DB.Where("parent_group_id = ?", group.ID).Find(&tempGroup)
		flatGroups = append(flatGroups, tempGroup...)
	}

	context.Status(http.StatusOK).JSON(
		&fiber.Map{
			"message": "Group fetched",
			"data":    flatGroups,
		})
	return nil
}

func (r *Repository) findSub(parentGroup models.Groups, level int, empId string, flatGroups *[]GroupsTree) error {
	ternGroups := []models.Groups{}
	r.DB.Where("parent_group_id = ?", parentGroup.ID).Find(&ternGroups)
	if ternGroups == nil {
		return nil
	}
	level = level + 1
	for _, ternGroup := range ternGroups {
		newEmpId := empId + "-" + strconv.Itoa(ternGroup.ID)
		tempGroup1 := GroupsTree{ternGroup.ID, ternGroup.GroupName, level, newEmpId}
		*flatGroups = append(*flatGroups, tempGroup1)
		err := r.findSub(ternGroup, level, newEmpId, flatGroups)
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func structsToExcel(flatGroups []GroupsTree) {
	f := excelize.NewFile()

	index := f.NewSheet("Sheet1")

	f.SetCellValue("Sheet1", "A1", "ID")
	f.SetCellValue("Sheet1", "B1", "GroupName")
	f.SetCellValue("Sheet1", "C1", "GroupLevel")
	f.SetCellValue("Sheet1", "D1", "EmpID")

	// set trang hoat donog
	f.SetActiveSheet(index)

	for i, group := range flatGroups {
		idByte, err := json.Marshal(group.Id)
		if err != nil {
			panic(err)
		}
		GNByte, err := json.Marshal(group.GroupName)
		if err != nil {
			panic(err)
		}

		GLByte, err := json.Marshal(group.GroupLevel)
		if err != nil {
			panic(err)
		}

		empID, err := json.Marshal(group.EmpId)
		if err != nil {
			panic(err)
		}

		f.SetCellValue("Sheet1", "A"+strconv.Itoa(i+2), string(idByte))
		f.SetCellValue("Sheet1", "B"+strconv.Itoa(i+2), string(GNByte))
		f.SetCellValue("Sheet1", "C"+strconv.Itoa(i+2), string(GLByte))
		f.SetCellValue("Sheet1", "D"+strconv.Itoa(i+2), string(empID))

	}

	// save xlsx file by the given path
	if err := f.SaveAs("GroupTree.xlsx"); err != nil {
		panic(err)
	}

}

func (r *Repository) GetGroupsByDepth(context *fiber.Ctx) error {
	// Truy xuất dữ liệu trong bộ nhớ cache
	// redis - connect
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "redis-db:6379",
		Password: "",
		DB:       0,
	})

	// SU dung Redis
	cachedRootGroups, err := redisClient.Get("rootGroups").Bytes()

	//  neu khong co trong cache
	if err != nil {
		dbRootGroups := []models.Groups{}
		r.DB.Where("parent_group_id = 0").Find(&dbRootGroups)

		cachedRootGroups, err = json.Marshal(dbRootGroups)
		if err != nil {
			return err
		}

		err = redisClient.Set("rootGroups", cachedRootGroups, 10*time.Second).Err()
		if err != nil {
			return err
		}
	}

	rootGroups := []models.Groups{}
	err = json.Unmarshal(cachedRootGroups, &rootGroups)
	if err != nil {
		return err
	}

	flatGroups := []GroupsTree{}
	for _, rootGroup := range rootGroups {
		level := 0
		empId := strconv.Itoa(rootGroup.ID)
		tempGroup := GroupsTree{rootGroup.ID, rootGroup.GroupName, level, empId}
		flatGroups = append(flatGroups, tempGroup)
		err := r.findSub(rootGroup, level, empId, &flatGroups)
		if err != nil {
			panic(err)
		}
	}
	structsToExcel(flatGroups)

	context.Status(http.StatusOK).JSON(
		&fiber.Map{
			"message": "Group fetched",
			"data":    flatGroups,
		})
	return nil
}

func (r *Repository) DeleteGroup(context *fiber.Ctx) error {
	groupModel := models.Groups{}
	id := context.Params("id")
	if id == "" {
		context.Status(http.StatusInternalServerError).JSON(
			&fiber.Map{"message": "ID EMPTY"})
		return nil
	}

	err := r.DB.Delete(groupModel, id).Error
	if err != nil {
		context.Status(http.StatusBadRequest).JSON(
			&fiber.Map{"message": "could not delete Book"})
		return err
	}

	context.Status(http.StatusOK).JSON(
		&fiber.Map{"message": "deleted"})
	return nil
}

// --------------  Cách cũ query thuần
// func (r *Repository) GetGroupsByID(context *fiber.Ctx) error {
// 	// groupModels := &[]models.Groups{}
// 	id := context.Params("id")
// 	q := ` with RECURSIVE groupTree AS
// 	(
// 	   select
// 			e1.id ,
// 			e1.group_name,
// 			e1.parent_group_id,
// 			0 as groupLevel,
// 			e1.id::varchar as emp_id
// 		from groups e1 where e1.id = ?

// 		UNION ALL
// 		select
// 			e2.id,
// 			e2.group_name,
// 			e2.parent_group_id,
// 			groupLevel + 1,
// 			emp_id || '-' || e2.id::varchar
// 		from groups e2
// 			INNER join groupTree et ON et.id = e2.parent_group_id)
// 	 SELECT id, group_name, groupLevel, emp_id from groupTree order by emp_id, groupLevel
// 	`
// 	rows, err := r.DB.Raw(q, id).Rows()
// 	if err != nil {
// 		context.Status(http.StatusBadRequest).JSON(
// 			&fiber.Map{"message": "could get groups"})
// 		return err
// 	}

// 	// groupOutput := []models.Groups{}
// 	defer rows.Close()
// 	// fmt.Print(groupOutput)

// 	rowDatas := []GroupsTree{}
// 	for rows.Next() {

// 		var id int
// 		var groupName *string
// 		var groupLevel int
// 		var empId string

// 		rows.Scan(&id, &groupName, &groupLevel, &empId)
// 		rowData := GroupsTree{id, groupName, groupLevel, empId}
// 		rowDatas = append(rowDatas, rowData)
// 	}

// 	context.Status(http.StatusOK).JSON(
// 		&fiber.Map{
// 			"message": "Group fetched",
// 			"data":    rowDatas,
// 		})
// 	return nil
// }

// ---- Dùng Đệ quy Golang
func (r *Repository) GetGroupsByID(context *fiber.Ctx) error {
	id := context.Params("id")
	flatGroups := []GroupsTree{}
	rootGroup := models.Groups{}
	r.DB.Where("id = ?", id).Find(&rootGroup)
	level := 0
	empId := strconv.Itoa(rootGroup.ID)
	tempGroup := GroupsTree{rootGroup.ID, rootGroup.GroupName, level, empId}
	flatGroups = append(flatGroups, tempGroup)
	err := r.findSub(rootGroup, level, empId, &flatGroups)
	if err != nil {
		panic(err)
	}
	structsToExcel(flatGroups)
	context.Status(http.StatusOK).JSON(
		&fiber.Map{
			"message": "Group fetched",
			"data":    flatGroups,
		})
	return nil
}

func (r *Repository) SetupRoutes(app *fiber.App) {
	app.Post("/create_group", r.CreateGroup)
	app.Post("/create_staff", r.CreateStaff)

	app.Post("/update_group/:id", r.UpdateGroup)
	// app.Post("/staff", r.UpdateStaff)

	app.Get("/groupsByWidth", r.GetGroupsByWidth)
	app.Get("/groupsByDepth", r.GetGroupsByDepth)
	// app.Get("/staff/:id", r.GetStaffByID)

	app.Get("/groups/:id", r.GetGroupsByID)

	app.Delete("/DeleteGroup/:id", r.DeleteGroup)
	// app.Delete("/DeleteStaff/:id", r.DeleteStaff)

}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	config := &storage.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Password: os.Getenv("DB_PASS"),
		User:     os.Getenv("DB_USER"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:   os.Getenv("DB_NAME"),
	}

	db, err := storage.NewInit(config)
	if err != nil {
		fmt.Println("can not connect")
		panic(err)
	}

	err = models.MigrateGroups(db)
	if err != nil {
		panic(err)
	}

	err = models.MigrateStaffs(db)
	if err != nil {
		panic(err)
	}
	r := Repository{
		DB: db,
	}

	app := fiber.New()
	r.SetupRoutes(app)
	app.Listen(":3000")

}
