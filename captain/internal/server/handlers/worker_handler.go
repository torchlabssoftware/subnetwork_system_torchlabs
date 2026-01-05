package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"

	functions "github.com/torchlabssoftware/subnetwork_system/internal/server/functions"
	middleware "github.com/torchlabssoftware/subnetwork_system/internal/server/middleware"
	server "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	"github.com/torchlabssoftware/subnetwork_system/internal/server/service"
	wsm "github.com/torchlabssoftware/subnetwork_system/internal/server/websocket"
)

type WorkerHandler struct {
	workerService service.WorkerService
	wsManager     *wsm.WebsocketManager
}

func NewWorkerHandler(workerService service.WorkerService, wsManager *wsm.WebsocketManager) *WorkerHandler {
	w := &WorkerHandler{
		workerService: workerService,
		wsManager:     wsManager,
	}
	return w
}

func (wh *WorkerHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.AdminAuthentication)
	r.Post("/", wh.AddWorker)
	r.Get("/", wh.GetAllWorkers)
	r.Get("/{name}", wh.GetWorkerByName)
	r.Delete("/{name}", wh.DeleteWorker)
	r.Post("/{name}/domains", wh.AddWorkerDomain)
	r.Delete("/{name}/domains", wh.DeleteWorkerDomain)
	return r
}

func (wh *WorkerHandler) WorkerRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.WorkerAuthentication)
	r.Post("/ws/login", wh.login)
	r.Get("/ws/serve", wh.serveWS)
	return r
}

func (wh *WorkerHandler) login(w http.ResponseWriter, r *http.Request) {
	var req server.WorkerLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if req.WorkerId == nil || *req.WorkerId == uuid.Nil {
		functions.RespondwithError(w, http.StatusBadRequest, "WorkerId is required", fmt.Errorf("worker_id is required"))
		return
	}
	code, message, err := wh.workerService.Login(r.Context(), *req.WorkerId)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	newOTP := wh.wsManager.OtpMap.NewOTP(*req.WorkerId)
	resp := &server.WorkerLoginResponce{
		Otp: newOTP.Key,
	}
	functions.RespondwithJSON(w, code, resp)
}

func (wh *WorkerHandler) serveWS(w http.ResponseWriter, r *http.Request) {
	otp := r.URL.Query().Get("otp")
	if otp == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "OTP is required", fmt.Errorf("otp is required"))
		return
	}
	valid, workerID := wh.wsManager.OtpMap.VerifyOTP(otp)
	if !valid {
		functions.RespondwithError(w, http.StatusUnauthorized, "Invalid OTP", fmt.Errorf("invalid otp"))
		return
	}
	wh.wsManager.ServeWS(w, r, workerID)
}

func (wh *WorkerHandler) AddWorker(w http.ResponseWriter, r *http.Request) {
	var req server.AddWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.RegionName == nil || *req.RegionName == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "RegionName is required", fmt.Errorf("region_name is required"))
		return
	}
	if req.IPAddress == nil || *req.IPAddress == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "IPAddress is required", fmt.Errorf("ip_address is required"))
		return
	}
	if req.Port == nil || *req.Port == 0 {
		functions.RespondwithError(w, http.StatusBadRequest, "Port is required", fmt.Errorf("port is required"))
		return
	}
	if req.PoolId == nil || *req.PoolId == uuid.Nil {
		functions.RespondwithError(w, http.StatusBadRequest, "PoolId is required", fmt.Errorf("pool_id is required"))
		return
	}

	worker, code, message, err := wh.workerService.CreateWorker(r.Context(), &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, code, worker)

}

func (wh *WorkerHandler) GetAllWorkers(w http.ResponseWriter, r *http.Request) {
	workers, code, message, err := wh.workerService.GetWorkers(r.Context())
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, code, workers)
}

func (wh *WorkerHandler) GetWorkerByName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Worker name is required", fmt.Errorf("name is required"))
		return
	}

	worker, code, message, err := wh.workerService.GetWorkerByName(r.Context(), name)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, code, worker)
}

func (wh *WorkerHandler) DeleteWorker(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Worker name is required", fmt.Errorf("name is required"))
		return
	}

	code, message, err := wh.workerService.DeleteWorker(r.Context(), name)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: message,
	}

	functions.RespondwithJSON(w, code, res)
}

func (wh *WorkerHandler) AddWorkerDomain(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Worker name is required", fmt.Errorf("name is required"))
		return
	}

	var req server.AddWorkerDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if len(req.Domain) == 0 {
		functions.RespondwithError(w, http.StatusBadRequest, "Domains are required", fmt.Errorf("domains are required"))
		return
	}

	code, message, err := wh.workerService.AddWorkerDomain(r.Context(), name, &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, code, map[string]string{"message": message})
}

func (wh *WorkerHandler) DeleteWorkerDomain(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		functions.RespondwithError(w, http.StatusBadRequest, "Worker name is required", fmt.Errorf("name is required"))
		return
	}

	var req server.DeleteWorkerDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		functions.RespondwithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if len(req.Domain) == 0 {
		functions.RespondwithError(w, http.StatusBadRequest, "Domains are required", fmt.Errorf("domains are required"))
		return
	}

	code, message, err := wh.workerService.DeleteWorkerDomain(r.Context(), name, &req)
	if err != nil {
		functions.RespondwithError(w, code, message, err)
		return
	}

	functions.RespondwithJSON(w, code, map[string]string{"message": message})
}
