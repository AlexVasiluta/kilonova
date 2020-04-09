package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
)

// /api/problem
func (s *API) RegisterProblemRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/getAll", s.GetAllProblems)
	r.Get("/getByID", s.GetProblemByID)
	return r
}

func (s *API) GetAllProblems(w http.ResponseWriter, r *http.Request) {
	var problems []models.Problem
	s.db.Preload("Tests").Find(&problems)
	json.NewEncoder(w).Encode(problems)
}

func (s *API) GetProblemByID(w http.ResponseWriter, r *http.Request) {
	var problem models.Problem
	idstr := r.FormValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		fmt.Fprintln(w, "Invalid ID")
	}
	s.db.Where("id = ?", id).Preload("Tests").First(&problem)
	json.NewEncoder(w).Encode(problem)
}
