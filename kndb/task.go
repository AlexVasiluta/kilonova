package kndb

import (
	"github.com/KiloProjects/Kilonova/grader/proto"
	"github.com/KiloProjects/Kilonova/internal/models"
	"gorm.io/gorm"
)

// GetTaskByID returns a task with the specified ID
func (d *DB) GetTaskByID(id uint) (*models.Task, error) {
	var task models.Task
	if err := d.DB.
		Preload("Problem").Preload("User").Preload("Tests", func(db *gorm.DB) *gorm.DB {
		return db.Order("eval_tests.id")
	}).Preload("Tests.Test").First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (d *DB) UserTasksOnProblem(userid uint, problemid uint) ([]models.Task, error) {
	var tasks []models.Task
	if err := d.DB.Preload("Problem").Preload("User").Preload("Tests", func(db *gorm.DB) *gorm.DB {
		return db.Order("eval_tests.id")
	}).Preload("Tests.Test").Where("user_id = ? and problem_id = ?", userid, problemid).
		Order("id desc").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (d *DB) MaxScoreFor(userid uint, problemid uint) (int, error) {
	tasks, err := d.UserTasksOnProblem(userid, problemid)
	if err != nil {
		return -1, err
	}
	maxscore := -1
	for _, task := range tasks {
		if task.Score > maxscore {
			maxscore = task.Score
		}
	}
	return maxscore, nil
}

// GetAllTasks returns all tasks
// TODO: Pagination
func (d *DB) GetAllTasks() ([]models.Task, error) {
	var tasks []models.Task
	if err := d.DB.Preload("Problem").Preload("User").Order("id desc").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// For use by the judge

func (d *DB) GetWaitingTasks() ([]models.Task, error) {
	var tasks []models.Task
	err := d.DB.Where("status = ?", models.StatusWaiting).
		Preload("Tests").Preload("Problem").Preload("Tests.Test").
		Find(&tasks).Error

	if len(tasks) == 0 {
		return nil, err
	}
	return tasks, err
}

func (d *DB) UpdateTaskVisibility(id uint, visible bool) error {
	var tmp models.Task
	tmp.ID = id
	return d.DB.Model(&tmp).Update("visible", visible).Error
}

func (d *DB) UpdateCompilation(c proto.CResponse) error {
	var tmp models.Task
	tmp.ID = uint(c.ID)
	return d.DB.Model(&tmp).Updates(map[string]interface{}{
		"compile_error":   !c.Success,
		"compile_message": c.Output,
	}).Error
}

func (d *DB) UpdateStatus(id uint, status, score int) error {
	var tmp models.Task
	tmp.ID = id
	return d.DB.Model(&tmp).Updates(map[string]interface{}{
		"status": status,
		"score":  score,
	}).Error
}

func (d *DB) UpdateEvalTest(r proto.STResponse, score int) error {
	var tmp models.EvalTest
	tmp.ID = uint(r.TID)
	return d.DB.Model(&tmp).Updates(map[string]interface{}{
		"output": r.Comments,
		"time":   r.Time,
		"memory": r.Memory,
		"score":  score,
		"done":   true,
	}).Error
}
