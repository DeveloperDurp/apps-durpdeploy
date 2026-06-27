package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"durpdeploy/internal/db"
	"durpdeploy/internal/repository"
	"durpdeploy/views/pages"
)

type ProjectHandler struct {
	repo *repository.Repository
}

func NewProjectHandler(repo *repository.Repository) *ProjectHandler {
	return &ProjectHandler{repo: repo}
}

func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.repo.Queries.ListProjects(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := pages.ProjectsList(projects).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if err := pages.ProjectsListPage(projects, r.URL.Path).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *ProjectHandler) NewProject(w http.ResponseWriter, r *http.Request) {
	project := db.Project{}
	if err := pages.ProjectFormPage(project, false, "", r.URL.Path).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	desc := r.FormValue("description")

	if name == "" {
		WriteFormError(w, r, pages.ProjectForm(db.Project{}, false, "Name is required"), pages.ProjectFormPage(db.Project{}, false, "Name is required", r.URL.Path))
		return
	}

	params := db.CreateProjectParams{
		Name: name,
		Description: sql.NullString{
			String: desc,
			Valid:  desc != "",
		},
	}

	if _, err := h.repo.Queries.CreateProject(r.Context(), params); err != nil {
		if IsUniqueViolation(err) {
			WriteFormError(w, r, pages.ProjectForm(db.Project{Name: name}, false, "A project with this name already exists"), pages.ProjectFormPage(db.Project{Name: name}, false, "A project with this name already exists", r.URL.Path))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseProjectID(r)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	project, err := h.repo.Queries.GetProject(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	steps, err := h.repo.Queries.ListStepsByProject(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := pages.ProjectDetail(project, steps).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if err := pages.ProjectDetailPage(project, steps, r.URL.Path).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *ProjectHandler) EditProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseProjectID(r)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	project, err := h.repo.Queries.GetProject(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := pages.ProjectFormPage(project, true, "", r.URL.Path).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseProjectID(r)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	desc := r.FormValue("description")

	if name == "" {
		project := db.Project{ID: id, Name: name}
		WriteFormError(w, r, pages.ProjectForm(project, true, "Name is required"), pages.ProjectFormPage(project, true, "Name is required", r.URL.Path))
		return
	}

	params := db.UpdateProjectParams{
		ID:   id,
		Name: name,
		Description: sql.NullString{
			String: desc,
			Valid:  desc != "",
		},
	}

	if _, err = h.repo.Queries.UpdateProject(r.Context(), params); err != nil {
		if IsUniqueViolation(err) {
			project := db.Project{ID: id, Name: name}
			WriteFormError(w, r, pages.ProjectForm(project, true, "A project with this name already exists"), pages.ProjectFormPage(project, true, "A project with this name already exists", r.URL.Path))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		projects, err := h.repo.Queries.ListProjects(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := pages.ProjectsList(projects).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
	}
}

func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseProjectID(r)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	if err = h.repo.Queries.DeleteProject(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	projects, err := h.repo.Queries.ListProjects(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := pages.ProjectsList(projects).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseProjectID(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	return strconv.ParseInt(idStr, 10, 64)
}
