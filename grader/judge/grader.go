package judge

import (
	"context"
	"fmt"
	"time"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/jinzhu/gorm"
)

// Grader is the *actual* high-level grader
// Grader
type Grader struct {
	// These are the channels that are propagated to the box managers
	MasterTasks   chan common.Task
	MasterUpdater chan common.Updater
	DataManager   *datamanager.Manager
	Managers      []*BoxManager

	ctx context.Context
	db  *gorm.DB
}

// NewGrader returns a new Grader instance (note that, as of the current architectural design, there should be only one grader)
func NewGrader(ctx context.Context, db *gorm.DB, dataManager *datamanager.Manager) *Grader {
	taskChan := make(chan common.Task, 5)
	updateChan := make(chan common.Updater, 20)
	return &Grader{
		MasterTasks:   taskChan,
		MasterUpdater: updateChan,
		DataManager:   dataManager,
		db:            db,
		ctx:           ctx,
	}
}

// Start begins polling the DB for changes and sends them to the boxes
func (g *Grader) Start() {
	for _, mgr := range g.Managers {
		mgr.Start(g.ctx)
	}

	// DB Poller (pushes data to g.MasterTasks)
	go func() {
		// We don't want to use max CPU, so we poll every few seconds
		ticker := time.NewTicker(4 * time.Second)
		for {
			select {
			case <-ticker.C:
				// poll db
				var tasks []common.Task
				g.db.Where("status = ?", common.StatusWaiting).
					Preload("Problem").Preload("Tests").Preload("Tests.Test").
					Find(&tasks)

				if len(tasks) > 0 {
					fmt.Printf("Found %d tasks\n", len(tasks))
				}

				// announce update
				for _, task := range tasks {
					g.MasterTasks <- task
				}
			case <-g.ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	// DB Updater (fetches data from g.MasterUpdater)
	go func() {
		for {
			select {
			case update := <-g.MasterUpdater:
				if err := update.Update(g.db); err != nil {
					fmt.Println("GRADER DB UPDATE ERROR:", err)
				}
			case <-g.ctx.Done():
				return
			}
		}
	}()
}

// NewManager creates a new manager and assigns the master channels to it
func (g *Grader) NewManager(id int) error {
	mgr, err := NewBoxManager(id, g.DataManager, g.MasterTasks, g.MasterUpdater)
	if err != nil {
		return err
	}
	g.Managers = append(g.Managers, mgr)
	return nil
}
