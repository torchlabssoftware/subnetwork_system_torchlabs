package service

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	server "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
)

type WorkerService interface {
	Login(ctx context.Context, req uuid.UUID) (code int, message string, err error)
	CreateWorker(ctx context.Context, req *server.AddWorkerRequest) (res *server.AddWorkerResponse, code int, message string, err error)
	GetWorkers(ctx context.Context) (res []server.AddWorkerResponse, code int, message string, err error)
	GetWorkerByName(ctx context.Context, name string) (res *server.AddWorkerResponse, code int, message string, err error)
	DeleteWorker(ctx context.Context, name string) (code int, message string, err error)
	AddWorkerDomain(ctx context.Context, name string, req *server.AddWorkerDomainRequest) (code int, message string, err error)
	DeleteWorkerDomain(ctx context.Context, name string, req *server.DeleteWorkerDomainRequest) (code int, message string, err error)
}

type workerService struct {
	queries *repository.Queries
	db      *sql.DB
}

func NewWorkerService(queries *repository.Queries, db *sql.DB) WorkerService {
	workerService := &workerService{
		queries: queries,
		db:      db,
	}
	return workerService
}

func (s *workerService) Login(ctx context.Context, req uuid.UUID) (code int, message string, err error) {
	_, err = s.queries.GetWorkerById(ctx, req)
	if err != nil {
		if err == sql.ErrNoRows {
			return http.StatusNotFound, "Worker not found", err
		}
		return http.StatusInternalServerError, "Failed to get worker", err
	}
	return http.StatusOK, "", nil
}

func (s *workerService) CreateWorker(ctx context.Context, req *server.AddWorkerRequest) (res *server.AddWorkerResponse, code int, message string, err error) {
	id := uuid.New()
	var name string
	switch *req.RegionName {
	case "North America":
		name = "usa-" + id.String()
	case "Europe":
		name = "eu-" + id.String()
	case "Asia":
		name = "asia-" + id.String()
	default:
		name = "globe-" + id.String()
	}
	worker, err := s.queries.CreateWorker(ctx, repository.CreateWorkerParams{
		ID:         id,
		Name:       name,
		RegionName: *req.RegionName,
		IpAddress:  *req.IPAddress,
		Port:       *req.Port,
		PoolID:     *req.PoolId,
	})
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal Server Error", err
	}

	return &server.AddWorkerResponse{
		ID:         worker.ID.String(),
		Name:       worker.Name,
		RegionName: *req.RegionName,
		IpAddress:  worker.IpAddress,
		Status:     worker.Status,
		Port:       worker.Port,
		PoolId:     worker.PoolID,
		LastSeen:   worker.LastSeen.Format("2006-01-02T15:04:05.999999Z"),
		CreatedAt:  worker.CreatedAt.Format("2006-01-02T15:04:05.999999Z"),
		Domains:    []string{},
	}, http.StatusOK, "", nil
}

func (s *workerService) GetWorkers(ctx context.Context) (res []server.AddWorkerResponse, code int, message string, err error) {
	workers, err := s.queries.GetAllWorkers(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, "No workers found", nil
		}
		return nil, http.StatusInternalServerError, "Internal Server Error", err
	}

	var resp []server.AddWorkerResponse
	for _, worker := range workers {
		resp = append(resp, server.AddWorkerResponse{
			ID:         worker.ID.String(),
			Name:       worker.Name,
			RegionName: worker.RegionName,
			IpAddress:  worker.IpAddress,
			Status:     worker.Status,
			Port:       worker.Port,
			PoolId:     worker.PoolID,
			LastSeen:   worker.LastSeen.Format("2006-01-02T15:04:05.999999Z"),
			CreatedAt:  worker.CreatedAt.Format("2006-01-02T15:04:05.999999Z"),
			Domains:    worker.Domains,
		})
	}

	return resp, http.StatusOK, "", nil
}

func (s *workerService) GetWorkerByName(ctx context.Context, name string) (res *server.AddWorkerResponse, code int, message string, err error) {
	worker, err := s.queries.GetWorkerByName(ctx, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, "Worker not found", err
		}
		return nil, http.StatusInternalServerError, "Internal Server Error", err
	}

	return &server.AddWorkerResponse{
		ID:         worker.ID.String(),
		Name:       worker.Name,
		RegionName: worker.RegionName,
		IpAddress:  worker.IpAddress,
		Status:     worker.Status,
		Port:       worker.Port,
		PoolId:     worker.PoolID,
		LastSeen:   worker.LastSeen.Format("2006-01-02T15:04:05.999999Z"),
		CreatedAt:  worker.CreatedAt.Format("2006-01-02T15:04:05.999999Z"),
		Domains:    worker.Domains,
	}, http.StatusOK, "", nil
}

func (s *workerService) DeleteWorker(ctx context.Context, name string) (code int, message string, err error) {
	res, err := s.queries.DeleteWorkerByName(ctx, name)
	if err != nil {
		return http.StatusInternalServerError, "Internal Server Error", err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return http.StatusNotFound, "Worker not found", nil
	}
	return http.StatusOK, "worker deleted successfully", nil
}

func (s *workerService) AddWorkerDomain(ctx context.Context, name string, req *server.AddWorkerDomainRequest) (code int, message string, err error) {
	_, err = s.queries.AddWorkerDomain(ctx, repository.AddWorkerDomainParams{
		Name:    name,
		Column2: req.Domain,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return http.StatusNotFound, "Worker not found", err
		}
		return http.StatusInternalServerError, "Failed to add domain", err
	}
	return http.StatusCreated, "Domains added successfully", nil
}

func (s *workerService) DeleteWorkerDomain(ctx context.Context, name string, req *server.DeleteWorkerDomainRequest) (code int, message string, err error) {
	result, err := s.queries.DeleteWorkerDomain(ctx, repository.DeleteWorkerDomainParams{
		Name:    name,
		Column2: req.Domain,
	})
	if err != nil {
		return http.StatusInternalServerError, "Failed to delete domain", err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return http.StatusNotFound, "No domains deleted", nil
	}
	return http.StatusOK, "Domain deleted successfully", nil
}
