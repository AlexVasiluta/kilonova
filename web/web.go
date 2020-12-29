// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/logic"
	"github.com/KiloProjects/Kilonova/internal/util"
	"github.com/go-chi/chi"
	"github.com/markbates/pkger"
	"github.com/tdewolff/minify/v2"
)

var templates *template.Template
var minifier *minify.M

type templateData struct {
	Version  string
	Title    string
	Params   map[string]string
	User     *db.User
	LoggedIn bool
	Debug    bool

	// for the status code page
	Code  string
	Error string

	// ProblemEditor tells us if the authed .User is able to edit the .Problem
	ProblemEditor bool

	// SubEditor tells us if the authed .User is able to change visibility of the .Submission
	SubEditor bool

	// Page-specific data
	// it is easier to just put this stuff here instead of in a `Data` interface
	Problems []*db.Problem

	Problem   *db.Problem
	ProblemID int64

	// for problem page
	Markdown         string
	IsPdfDescription bool

	ContentUser *db.User
	IsCUser     bool

	Submissions []*db.Submission

	Submission *db.Submission
	SubID      int64

	Test   *db.Test
	TestID int64

	Top100 []db.Top100Row

	// Since codemirror is a particulairly big library, we should load it only when needed
	Codemirror bool

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
	kn    *logic.Kilonova
	dm    datamanager.Manager
	rd    *Renderer
	debug bool
}

func (rt *Web) status(w http.ResponseWriter, r *http.Request, statusCode int, err string) {
	code := fmt.Sprintf("%d: %s", statusCode, http.StatusText(statusCode))
	templ := rt.hydrateTemplate(r, code)
	templ.Code = code
	templ.Error = err

	w.WriteHeader(statusCode)
	rt.build(w, r, "statusCode", templ)
}

func (rt *Web) notFound(w http.ResponseWriter, r *http.Request) {
	rt.status(w, r, 404, "")
}

// Router returns a chi.Router
// TODO: Split routes in functions
func (rt *Web) Router() chi.Router {
	r := chi.NewRouter()

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
			log.Println("CAN'T OPEN FAVICON")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		fstat, err := file.Stat()
		if err != nil {
			log.Println("CAN'T STAT FAVICON")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		http.ServeContent(w, r, fstat.Name(), fstat.ModTime(), file)
	})

	r.Group(func(r chi.Router) {
		r.Use(rt.getUser)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			problems, err := rt.kn.DB.VisibleProblems(r.Context(), util.User(r))
			if err != nil {
				log.Println("/:", err)
				rt.status(w, r, 500, "")
				return
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				log.Println("/", err)
				rt.status(w, r, 500, "")
				return
			}
			templ := rt.hydrateTemplate(r, "")
			templ.Problems = problems
			rt.build(w, r, "index", templ)
		})

		r.Route("/profile", func(r chi.Router) {
			r.With(rt.mustBeAuthed).Get("/", func(w http.ResponseWriter, r *http.Request) {
				user := util.User(r)
				templ := rt.hydrateTemplate(r, fmt.Sprintf("Profil %s", user.Name))
				templ.ContentUser = user
				templ.IsCUser = true
				rt.build(w, r, "profile", templ)
			})
			r.Route("/{user}", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					user, err := rt.kn.DB.UserByName(r.Context(), chi.URLParam(r, "user"))
					if err != nil {
						if errors.Is(err, sql.ErrNoRows) {
							rt.status(w, r, 404, "")
							return
						}
						fmt.Println(err)
						rt.status(w, r, 500, "")
						return
					}

					templ := rt.hydrateTemplate(r, fmt.Sprintf("Profil %s", user.Name))
					templ.ContentUser = user
					rt.build(w, r, "profile", templ)
				})
			})
		})

		r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r, "Setări")
			rt.build(w, r, "settings", templ)
		})

		r.Get("/changelog", func(w http.ResponseWriter, r *http.Request) {
			file, err := pkger.Open("/CHANGELOG.md")
			if err != nil {
				log.Println("CAN'T OPEN CHANGELOG")
				rt.status(w, r, 500, "Can't load changelog")
				return
			}
			changelog, _ := io.ReadAll(file)
			ch, err := rt.rd.Render(changelog)
			if err != nil {
				log.Println("CAN'T RENDER CHANGELOG")
				rt.status(w, r, 500, "Can't render changelog")
				return
			}

			templ := rt.hydrateTemplate(r, "Changelog")
			templ.Markdown = ch.String()
			rt.build(w, r, "mdrender", templ)
		})

		r.Get("/todo", func(w http.ResponseWriter, r *http.Request) {
			file, err := pkger.Open("/TODO.md")
			if err != nil {
				log.Println("CAN'T OPEN TODO")
				rt.status(w, r, 500, "Can't load todo list")
				return
			}

			todo, _ := io.ReadAll(file)
			t, err := rt.rd.Render(todo)
			if err != nil {
				log.Println("CAN'T RENDER TODO")
				rt.status(w, r, 500, "Can't render todo list")
				return
			}

			templ := rt.hydrateTemplate(r, "Todo list")
			templ.Markdown = t.String()
			rt.build(w, r, "mdrender", templ)
		})

		r.Get("/about", func(w http.ResponseWriter, r *http.Request) {
			file, err := pkger.Open("/ABOUT.md")
			if err != nil {
				log.Println("CAN'T OPEN ABOUT")
				rt.status(w, r, 500, "Can't load About page")
				return
			}

			about, _ := io.ReadAll(file)
			t, err := rt.rd.Render(about)
			if err != nil {
				log.Println("CAN'T RENDER ABOUT")
				rt.status(w, r, 500, "Can't render About page")
				return
			}

			templ := rt.hydrateTemplate(r, "To do list")
			templ.Markdown = t.String()
			rt.build(w, r, "mdrender", templ)
		})

		r.Get("/top100", func(w http.ResponseWriter, r *http.Request) {
			top100, err := rt.kn.DB.Top100(r.Context())
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				fmt.Println(err)
				rt.status(w, r, 500, err.Error())
				return
			}
			templ := rt.hydrateTemplate(r, "Top 100")
			templ.Top100 = top100
			rt.build(w, r, "top100", templ)
		})

		r.Route("/probleme", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				problems, err := rt.kn.DB.VisibleProblems(r.Context(), util.User(r))
				if err != nil {
					fmt.Println(err)
					rt.status(w, r, 500, "")
					return
				}
				templ := rt.hydrateTemplate(r, "Probleme")
				templ.Problems = problems
				rt.build(w, r, "probleme", templ)
			})
			r.With(rt.mustBeProposer).Get("/create", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r, "Creare problemă")
				rt.build(w, r, "createpb", templ)
			})
			r.Route("/{id}", func(r chi.Router) {
				r.Use(rt.ValidateProblemID)
				r.Use(rt.ValidateVisible)
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					problem := util.Problem(r)

					buf, err := rt.rd.Render([]byte(problem.Description))
					if err != nil {
						log.Println(err)
					}
					templ := rt.hydrateTemplate(r, fmt.Sprintf("Problema #%d: %s", problem.ID, problem.Name))
					templ.Codemirror = true
					templ.Markdown = buf.String()
					rt.build(w, r, "problema", templ)
				})
				r.Route("/edit", func(r chi.Router) {
					r.Use(rt.mustBeEditor)
					r.Get("/", func(w http.ResponseWriter, r *http.Request) {
						problem := util.Problem(r)
						templ := rt.hydrateTemplate(r, fmt.Sprintf("EDIT | Problema #%d: %s", problem.ID, problem.Name))
						rt.build(w, r, "edit/index", templ)
					})
					r.Get("/enunt", func(w http.ResponseWriter, r *http.Request) {
						problem := util.Problem(r)
						templ := rt.hydrateTemplate(r, fmt.Sprintf("EDITARE ENUNȚ | Problema #%d: %s", problem.ID, problem.Name))
						templ.Codemirror = true
						rt.build(w, r, "edit/enunt", templ)
					})
					r.Get("/limite", func(w http.ResponseWriter, r *http.Request) {
						problem := util.Problem(r)
						templ := rt.hydrateTemplate(r, fmt.Sprintf("EDITARE LIMITE | Problema #%d: %s", problem.ID, problem.Name))
						rt.build(w, r, "edit/limite", templ)
					})
					r.Route("/teste", func(r chi.Router) {
						r.Get("/", func(w http.ResponseWriter, r *http.Request) {
							problem := util.Problem(r)
							templ := rt.hydrateTemplate(r, fmt.Sprintf("CREARE TEST | Problema #%d: %s", problem.ID, problem.Name))
							templ.Sidebar = true
							templ.Codemirror = true
							rt.build(w, r, "edit/testAdd", templ)
						})
						r.With(rt.ValidateTestID).Get("/{tid}", func(w http.ResponseWriter, r *http.Request) {
							test := util.Test(r)
							problem := util.Problem(r)
							templ := rt.hydrateTemplate(r, fmt.Sprintf("EDITARE TESTUL %d | Problema #%d: %s", test.VisibleID, problem.ID, problem.Name))
							templ.Sidebar = true
							templ.Codemirror = true
							rt.build(w, r, "edit/testEdit", templ)
						})
					})
				})
			})
		})

		r.Route("/submissions", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r, "Submisii")
				rt.build(w, r, "submissions", templ)
			})
			r.With(rt.ValidateSubmissionID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r, fmt.Sprintf("Submisia %d", util.Submission(r).ID))
				rt.build(w, r, "submission", templ)
			})
		})

		r.With(rt.mustBeAdmin).Get("/admin", rt.simpleTempl("Interfață Admin", "admin"))

		r.With(rt.mustBeVisitor).Get("/login", rt.simpleTempl("Log In", "login"))
		r.With(rt.mustBeVisitor).Get("/signup", rt.simpleTempl("Înregistrare", "signup"))

		r.With(rt.mustBeAuthed).Get("/logout", func(w http.ResponseWriter, r *http.Request) {
			// i could redirect to /api/auth/logout, but it's easier to do it like this
			rt.kn.RemoveSessionCookie(w, r)
			http.Redirect(w, r, "/", http.StatusFound)
		})

	})

	r.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean(r.RequestURI)
		p = path.Join("/static", p)
		file, err := pkger.Open(p)
		if err != nil {
			rt.notFound(w, r)
			return
		}
		defer file.Close()
		fstat, err := file.Stat()
		if err != nil {
			rt.notFound(w, r)
			return
		}
		http.ServeContent(w, r, fstat.Name(), fstat.ModTime(), file)
	}))
	r.NotFound(rt.notFound)

	return r
}

func (rt *Web) simpleTempl(title, templName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		templ := rt.hydrateTemplate(r, title)
		rt.build(w, r, templName, templ)
	}
}

// NewWeb returns a new web instance
func NewWeb(kn *logic.Kilonova) *Web {
	rd, err := NewRenderer()
	if err != nil {
		panic(err)
	}
	return &Web{kn, kn.DM, rd, kn.Debug}
}

func init() {
	pkger.Include("/web/templ")
	pkger.Include("/static")
}
