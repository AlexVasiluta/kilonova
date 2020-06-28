package common

import "github.com/jinzhu/gorm"

const (
	_ = iota
	cpp
	c
	golang
	python
)

// Config is the main information for the platform
type Config struct {
	SecretKey string `json:"secretKey"`
	Debug     bool   `json:"debugMode"`
}

// Session represents the data storred in a session cookie
type Session struct {
	IsAdmin bool `json:"isAdmin"`
	UserID  uint `json:"userID"`
}

// MOTD presents a random message of the day
type MOTD struct {
	gorm.Model
	Motd string `json:"motd,omitempty"`
}

// User represents a user profile
type User struct {
	gorm.Model
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isAdmin,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"-"`
}

// Problem is the main object for problem
type Problem struct {
	gorm.Model
	Name     string `json:"title"`
	Text     string `json:"description"`
	Tests    []Test `json:"tests,omitempty"`
	TestName string `json:"testName"`
	// User is the author
	User         User   `json:"author"`
	UserID       uint   `json:"authorID"`
	Limits       Limits `json:"limits"`
	LimitsID     uint   `json:"limitsID"`
	SourceSize   int64  `json:"sourceSize"`
	ConsoleInput bool   `json:"consoleInput"`
}

// Test is the type for sample test
// NOTE: When Score < 0, it means that an error occured
type Test struct {
	gorm.Model
	Score     int  `json:"score"`
	ProblemID uint `json:"problemID"`
	VisibleID uint `json:"visibleID"`
}

// EvalTest is the type for tests meant for evaluation
// NOTE: When Score < 0, it means that an error occured
type EvalTest struct {
	gorm.Model
	Done bool `json:"done"`

	// Output is the text displayed on the frontend (like `Fatal signal 11` or `Missing output file`)
	Output string `json:"resultinfo"`
	Score  int    `json:"score"`
	Test   Test   `json:"test"`
	TestID uint   `json:"testID"`
	UserID uint   `json:"userID"`
	TaskID uint   `json:"taskID"`
}

// Task is the type for user-submitted tasks
type Task struct {
	gorm.Model
	SourceCode     string     `json:"code,omitempty"`
	User           User       `json:"user"`
	UserID         uint       `json:"userid"`
	Tests          []EvalTest `json:"tests,omitempty"`
	Problem        Problem    `json:"problem"`
	ProblemID      uint       `json:"problemid"`
	Language       string     `json:"language"`
	Status         int        `json:"status"`
	CompileError   bool       `json:"compileError"`
	CompileMessage string     `json:"compileMessage"`
	Score          int        `json:"score"`
}

const (
	// These represent the different possible statuses of a task

	// StatusWaiting is the initial state, the Task hasn't been picked up yet
	StatusWaiting = iota
	// StatusWorking is the state when a Task has been picked up by a box but hasn't yet finished
	StatusWorking
	// StatusDone is the state when a Task has been fully graded
	StatusDone
)
