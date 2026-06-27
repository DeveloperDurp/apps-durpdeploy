package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"durpdeploy/internal/db"
	"durpdeploy/internal/repository"
	"durpdeploy/internal/runner"
	"durpdeploy/views/pages"
)

type DeploymentHandler struct {
	repo   *repository.Repository
	runner *runner.DeploymentRunner
}

func NewDeploymentHandler(repo *repository.Repository, runner *runner.DeploymentRunner) *DeploymentHandler {
	return &DeploymentHandler{repo: repo, runner: runner}
}

func (h *DeploymentHandler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	releaseID, err := strconv.ParseInt(r.FormValue("release_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid release ID", http.StatusBadRequest)
		return
	}

	environmentID, err := strconv.ParseInt(r.FormValue("environment_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid environment ID", http.StatusBadRequest)
		return
	}

	deployment, err := h.repo.Queries.CreateDeployment(r.Context(), db.CreateDeploymentParams{
		ReleaseID:     releaseID,
		EnvironmentID: environmentID,
		Status:        "pending",
		StartedAt:     sql.NullInt64{},
		FinishedAt:    sql.NullInt64{},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go h.runner.Run(context.Background(), deployment.ID, releaseID, environmentID)

	http.Redirect(w, r, fmt.Sprintf("/deployments/%d", deployment.ID), http.StatusSeeOther)
}

func (h *DeploymentHandler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	deployment, err := h.repo.Queries.GetDeployment(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Deployment not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	release, err := h.repo.Queries.GetRelease(r.Context(), deployment.ReleaseID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	project, err := h.repo.Queries.GetProject(r.Context(), release.ProjectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	environment, err := h.repo.Queries.GetEnvironment(r.Context(), deployment.EnvironmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logs, err := h.repo.Queries.ListDeploymentLogsByDeployment(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := pages.DeploymentDetail(project, release, environment, deployment, logs).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if err := pages.DeploymentDetailPage(project, release, environment, deployment, logs, r.URL.Path).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *DeploymentHandler) GetDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	deployment, err := h.repo.Queries.GetDeployment(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Deployment not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := pages.StatusBadgeContainer(deployment).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *DeploymentHandler) CancelDeployment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	deployment, err := h.repo.Queries.GetDeployment(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Deployment not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if deployment.Status != "running" {
		http.Error(w, "Deployment is not running", http.StatusBadRequest)
		return
	}

	if err := h.runner.Cancel(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deployment, err = h.repo.Queries.GetDeployment(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := pages.StatusBadgeContainer(deployment).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.Redirect(w, r, fmt.Sprintf("/deployments/%d", deployment.ID), http.StatusSeeOther)
	}
}

func (h *DeploymentHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	deployments, err := h.repo.Queries.ListDeployments(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]pages.DeploymentListItem, len(deployments))
	for i, d := range deployments {
		release, err := h.repo.Queries.GetRelease(r.Context(), d.ReleaseID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		project, err := h.repo.Queries.GetProject(r.Context(), release.ProjectID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		env, err := h.repo.Queries.GetEnvironment(r.Context(), d.EnvironmentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items[i] = pages.DeploymentListItem{
			Deployment:      d,
			ProjectName:     project.Name,
			ReleaseVersion:  release.Version,
			EnvironmentName: env.Name,
		}
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := pages.DeploymentsList(items).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		if err := pages.DeploymentsListPage(items, r.URL.Path).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
