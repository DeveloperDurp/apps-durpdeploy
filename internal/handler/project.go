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
	lifecycles, err := h.repo.Queries.ListLifecycles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := pages.ProjectFormPage(db.Project{}, false, "", lifecycles, r.URL.Path).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	desc := r.FormValue("description")

	if name == "" {
		lifecycles, _ := h.repo.Queries.ListLifecycles(r.Context())
		WriteFormError(w, r, pages.ProjectForm(db.Project{}, false, "Name is required", lifecycles), pages.ProjectFormPage(db.Project{}, false, "Name is required", lifecycles, r.URL.Path))
		return
	}

	params := db.CreateProjectParams{
		Name: name,
		Description: sql.NullString{
			String: desc,
			Valid:  desc != "",
		},
	}

	created, err := h.repo.Queries.CreateProject(r.Context(), params)
	if err != nil {
		if IsUniqueViolation(err) {
			lifecycles, _ := h.repo.Queries.ListLifecycles(r.Context())
			WriteFormError(w, r, pages.ProjectForm(db.Project{Name: name}, false, "A project with this name already exists", lifecycles), pages.ProjectFormPage(db.Project{Name: name}, false, "A project with this name already exists", lifecycles, r.URL.Path))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.applyLifecycleSelection(r, created.ID); err != nil {
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

	panel, err := h.buildLifecyclePanel(r, project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := pages.ProjectDetail(project, steps, panel).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if err := pages.ProjectDetailPage(project, steps, panel, r.URL.Path).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// buildLifecyclePanel returns the lifecycle panel data for a project's detail page.
// If the project has no lifecycle, returns an empty panel. For each stage, loads the
// most recent successful deployment (if any) so the panel can show "No deployments"
// or the deployed version.
func (h *ProjectHandler) buildLifecyclePanel(r *http.Request, project db.Project) (pages.LifecyclePanel, error) {
	if !project.LifecycleID.Valid {
		return pages.LifecyclePanel{}, nil
	}
	lc, err := h.repo.Queries.GetLifecycle(r.Context(), project.LifecycleID.Int64)
	if err != nil {
		return pages.LifecyclePanel{}, err
	}
	stages, err := h.repo.Queries.ListLifecycleStages(r.Context(), lc.ID)
	if err != nil {
		return pages.LifecyclePanel{}, err
	}
	envsByID, err := h.envsByID(r)
	if err != nil {
		return pages.LifecyclePanel{}, err
	}
	releasesByID, err := h.releasesByID(r, project.ID)
	if err != nil {
		return pages.LifecyclePanel{}, err
	}
	views := make([]pages.LifecyclePanelStage, len(stages))
	for i, s := range stages {
		v := pages.LifecyclePanelStage{
			Stage:       s,
			Environment: envsByID[s.EnvironmentID],
		}
		dep, derr := h.repo.Queries.GetLatestSuccessfulDeploymentForEnv(r.Context(), s.EnvironmentID)
		if derr == nil {
			if rel, ok := releasesByID[dep.ReleaseID]; ok {
				v.LatestDeployment = &dep
				v.LatestVersion = rel.Version
			}
		}
		views[i] = v
	}
	return pages.LifecyclePanel{Lifecycle: lc, Stages: views}, nil
}

func (h *ProjectHandler) envsByID(r *http.Request) (map[int64]db.Environment, error) {
	envs, err := h.repo.Queries.ListEnvironments(r.Context())
	if err != nil {
		return nil, err
	}
	m := make(map[int64]db.Environment, len(envs))
	for _, e := range envs {
		m[e.ID] = e
	}
	return m, nil
}

func (h *ProjectHandler) releasesByID(r *http.Request, projectID int64) (map[int64]db.Release, error) {
	rels, err := h.repo.Queries.ListReleasesByProject(r.Context(), projectID)
	if err != nil {
		return nil, err
	}
	m := make(map[int64]db.Release, len(rels))
	for _, rel := range rels {
		m[rel.ID] = rel
	}
	return m, nil
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

	lifecycles, err := h.repo.Queries.ListLifecycles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := pages.ProjectFormPage(project, true, "", lifecycles, r.URL.Path).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseProjectID(r)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	desc := r.FormValue("description")

	if name == "" {
		project := db.Project{ID: id, Name: name}
		lifecycles, _ := h.repo.Queries.ListLifecycles(r.Context())
		WriteFormError(w, r, pages.ProjectForm(project, true, "Name is required", lifecycles), pages.ProjectFormPage(project, true, "Name is required", lifecycles, r.URL.Path))
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
			lifecycles, _ := h.repo.Queries.ListLifecycles(r.Context())
			WriteFormError(w, r, pages.ProjectForm(project, true, "A project with this name already exists", lifecycles), pages.ProjectFormPage(project, true, "A project with this name already exists", lifecycles, r.URL.Path))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.applyLifecycleSelection(r, id); err != nil {
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

// applyLifecycleSelection reads the lifecycle_id form field and updates the
// project's lifecycle accordingly. Empty string clears the lifecycle.
func (h *ProjectHandler) applyLifecycleSelection(r *http.Request, projectID int64) error {
	lifecycleStr := strings.TrimSpace(r.FormValue("lifecycle_id"))
	if lifecycleStr == "" {
		return h.repo.Queries.ClearProjectLifecycle(r.Context(), projectID)
	}
	id, err := strconv.ParseInt(lifecycleStr, 10, 64)
	if err != nil {
		return err
	}
	return h.repo.Queries.SetProjectLifecycle(r.Context(), db.SetProjectLifecycleParams{
		LifecycleID: sql.NullInt64{Int64: id, Valid: true},
		ID:          projectID,
	})
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
