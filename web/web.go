// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/models"
	"github.com/KiloProjects/Kilonova/internal/util"
	"github.com/KiloProjects/Kilonova/kndb"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/markbates/pkger"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
	"gorm.io/gorm"
)

var templates *template.Template
var minifier *minify.M

type templateData struct {
	Title    string
	Params   map[string]string
	User     models.User
	LoggedIn bool

	// Page-specific data
	// it is easier to just put this stuff here instead of in a `Data` interface
	Problems []models.Problem
	Problem  models.Problem

	ContentUser models.User

	Tasks []models.Task

	Task   models.Task
	TaskID uint

	ProblemID uint

	Version string

	Test   models.Test
	TestID uint

	// ProblemEditor tells us if the authed .User is able to edit the .Problem
	ProblemEditor bool

	// TaskEditor tells us if the authed .User is able to change visibility of the .Task
	TaskEditor bool

	Sidebar bool

	Changelog string

	// OpenGraph stuff
	OGTitle string
	OGType  string
	OGUrl   string
	OGImage string
	OGDesc  string
}

// Web is the struct representing this whole package
type Web struct {
	dm     datamanager.Manager
	db     *kndb.DB
	logger *log.Logger
	debug  bool
}

func (rt *Web) newTemplate() *template.Template {
	// table for gradient, initialize here so it panics if we make a mistake
	colorTable := gTable{
		{mustParseHex("#f45d64"), 0.0},
		{mustParseHex("#eaf200"), 0.5},
		{mustParseHex("#64ce3a"), 1.0},
	}

	return template.Must(parseAllTemplates(template.New("web").Funcs(template.FuncMap{
		"dumpStruct":   spew.Sdump,
		"getTestData":  rt.getTestData,
		"getFullTests": rt.getFullTestData,
		"taskStatus": func(id int) template.HTML {
			switch id {
			case models.StatusWaiting:
				return template.HTML("În așteptare...")
			case models.StatusWorking:
				return template.HTML("În lucru...")
			case models.StatusDone:
				return template.HTML("Finalizată")
			default:
				return template.HTML("Stare necunoscută")
			}
		},
		"KBtoMB": func(kb uint64) float64 {
			return float64(kb) / 1024.0
		},
		"gradient": func(score, maxscore int) template.CSS {
			return gradient(int(score), maxscore, colorTable)
		},
		"zeroto100": func() []int {
			var v []int = make([]int, 0)
			for i := 0; i <= 100; i++ {
				v = append(v, i)
			}
			return v
		},
		"taskScore": func(problem models.Problem, user models.User) string {
			score, err := rt.db.MaxScoreFor(user.ID, problem.ID)
			if err != nil || score < 0 {
				return "-"
			}
			return fmt.Sprint(score)
		},
		"problemTasks": func(problem models.Problem, user models.User) []models.Task {
			tasks, err := rt.db.UserTasksOnProblem(user.ID, problem.ID)
			if err != nil {
				return nil
			}
			return tasks
		},
	}), root))
}

func (rt *Web) build(w http.ResponseWriter, r *http.Request, name string, temp templateData) {
	if err := templates.ExecuteTemplate(w, name, temp); err != nil {
		rt.logger.Printf("%s: %v\n", temp.OGUrl, err)
	}
}

// GetRouter returns a chi.Router
// TODO: Split routes in functions
func (rt *Web) GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)

	templates = rt.newTemplate()

	if rt.debug {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				templates = rt.newTemplate()
				next.ServeHTTP(w, r)
			})
		})
	}

	r.Mount("/static", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean(r.RequestURI)
		if !strings.HasPrefix(p, "/static") {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		file, err := pkger.Open(p)
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		fstat, err := file.Stat()
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		http.ServeContent(w, r, fstat.Name(), fstat.ModTime(), file)
	}))

	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		file, err := pkger.Open("/static/favicon.ico")
		if err != nil {
			rt.logger.Println("CAN'T OPEN FAVICON")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		fstat, err := file.Stat()
		if err != nil {
			rt.logger.Println("CAN'T STAT FAVICON")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		http.ServeContent(w, r, fstat.Name(), fstat.ModTime(), file)
	})

	r.With(rt.getUser).Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			problems, err := rt.db.GetAllVisibleProblems(util.UserFromContext(r))
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				rt.logger.Println("/", err)
				http.Error(w, http.StatusText(500), 500)
				return
			}
			templ := rt.hydrateTemplate(r)
			templ.Problems = problems
			rt.build(w, r, "index", templ)
		})

		r.Get("/changelog", func(w http.ResponseWriter, r *http.Request) {
			file, err := pkger.Open("/CHANGELOG.md")
			if err != nil {
				rt.logger.Println("CAN'T OPEN CHANGELOG")
				http.Error(w, http.StatusText(500), 500)
				return
			}
			changelog, _ := ioutil.ReadAll(file)
			templ := rt.hydrateTemplate(r)
			templ.Changelog = string(changelog)
			rt.build(w, r, "changelog", templ)
		})

		r.Route("/probleme", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				problems, err := rt.db.GetAllVisibleProblems(util.UserFromContext(r))
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					rt.logger.Println("/probleme/", err)
					http.Error(w, http.StatusText(500), 500)
					return
				}
				templ := rt.hydrateTemplate(r)
				templ.Title = "Probleme"
				templ.Problems = problems
				rt.build(w, r, "probleme", templ)
			})
			r.With(rt.mustBeProposer).Get("/create", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r)
				templ.Title = "Creare problemă"
				rt.build(w, r, "createpb", templ)
			})
			r.Route("/{id}", func(r chi.Router) {
				r.Use(rt.ValidateProblemID)
				r.Use(rt.ValidateVisible)
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					problem := util.ProblemFromContext(r)

					templ := rt.hydrateTemplate(r)
					templ.Title = fmt.Sprintf("#%d: %s", problem.ID, problem.Name)
					rt.build(w, r, "problema", templ)
				})
				r.Route("/edit", func(r chi.Router) {
					r.Use(rt.mustBeEditor)
					r.Get("/", func(w http.ResponseWriter, r *http.Request) {
						problem := util.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("EDIT | #%d: %s", problem.ID, problem.Name)
						rt.build(w, r, "edit/index", templ)
					})
					r.Get("/enunt", func(w http.ResponseWriter, r *http.Request) {
						problem := util.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("ENUNT - EDIT | #%d: %s", problem.ID, problem.Name)
						rt.build(w, r, "edit/enunt", templ)
					})
					r.Get("/limite", func(w http.ResponseWriter, r *http.Request) {
						problem := util.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("LIMITE - EDIT | #%d: %s", problem.ID, problem.Name)
						rt.build(w, r, "edit/limite", templ)
					})
					r.Route("/teste", func(r chi.Router) {
						r.Get("/", func(w http.ResponseWriter, r *http.Request) {
							problem := util.ProblemFromContext(r)
							templ := rt.hydrateTemplate(r)
							templ.Title = fmt.Sprintf("TESTE - EDIT | #%d: %s", problem.ID, problem.Name)
							templ.Sidebar = true
							rt.build(w, r, "edit/testAdd", templ)
						})
						r.With(rt.ValidateTestID).Get("/{tid}", func(w http.ResponseWriter, r *http.Request) {
							test := util.TestFromContext(r)
							problem := util.ProblemFromContext(r)
							templ := rt.hydrateTemplate(r)
							templ.Title = fmt.Sprintf("Teste - EDIT %d | #%d: %s", test.VisibleID, problem.ID, problem.Name)
							templ.Sidebar = true
							rt.build(w, r, "edit/testEdit", templ)
						})
					})
				})
			})
		})

		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				tasks, err := rt.db.GetAllTasks()
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					rt.logger.Println("/tasks/", err)
					http.Error(w, http.StatusText(500), 500)
					return
				}
				templ := rt.hydrateTemplate(r)
				templ.Title = "Tasks"
				templ.Tasks = tasks
				rt.build(w, r, "tasks", templ)
			})
			r.With(rt.ValidateTaskID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r)
				templ.Title = fmt.Sprintf("Task %d", templ.Task.ID)
				rt.build(w, r, "task", templ)
			})
		})

		r.With(rt.mustBeAdmin).Get("/admin", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Admin switches"
			rt.build(w, r, "admin", templ)
		})

		r.With(rt.mustBeVisitor).Get("/login", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Log In"
			rt.build(w, r, "login", templ)
		})
		r.With(rt.mustBeVisitor).Get("/signup", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Sign Up"
			rt.build(w, r, "signup", templ)
		})

		r.With(rt.mustBeAuthed).Get("/logout", func(w http.ResponseWriter, r *http.Request) {
			// i could redirect to /api/auth/logout, but it's easier to do it like this
			common.RemoveSessionCookie(w)
			http.Redirect(w, r, "/", http.StatusFound)
		})
	})

	return r
}

// NewWeb returns a new web instance
func NewWeb(dm datamanager.Manager, db *kndb.DB, logger *log.Logger, debug bool) *Web {
	return &Web{dm, db, logger, debug}
}

func init() {
	pkger.Include("/include")
	pkger.Include("/static")

	minifier = minify.New()
	minifier.AddFunc("text/html", html.Minify)
}
