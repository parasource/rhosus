package api

import (
	"github.com/gorilla/mux"
	api_pb "github.com/parasource/rhosus/rhosus/pb/api"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/errors"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

func (a *Api) handleLogin(rw http.ResponseWriter, r *http.Request) {
	var req api_pb.LoginRequest
	err := a.decoder.Unmarshal(r.Body, &req)
	if err != nil {
		log.Debug().Err(err).Msg("error unmarshaling login request")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	var res api_pb.LoginResponse

	if authMethod, ok := a.Config.AuthMethods[req.Method]; ok {
		authRes, err := authMethod.Login(prepareLoginRequest(req))
		if err != nil {
			log.Error().Err(err).Msg("error conducting login operation")
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		res = api_pb.LoginResponse{
			Token:   authRes.Token,
			Success: authRes.Success,
			Message: authRes.Message,
		}

		switch res.Success {
		case true:
			rw.WriteHeader(http.StatusOK)
		case false:
			rw.WriteHeader(http.StatusBadRequest)
		}

	} else {
		res = api_pb.LoginResponse{
			Success: false,
			Message: "unknown login method",
		}

		rw.WriteHeader(http.StatusBadRequest)
	}

	a.encoder.Marshal(rw, &res)
}

func (a *Api) handleCreateTestUser(rw http.ResponseWriter, r *http.Request) {
	roleID, _ := uuid.NewV4()
	passw, _ := bcrypt.GenerateFromPassword([]byte("Mypassword"), bcrypt.DefaultCost)
	role := &control_pb.Role{
		ID:          roleID.String(),
		Name:        "egor",
		Permissions: []string{},
		Password:    string(passw),
	}
	err := a.registry.Storage.StoreRole(role)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		log.Error().Err(err).Msg("error storing role")
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (a *Api) handleMakeDir(rw http.ResponseWriter, r *http.Request) {
	var msg api_pb.MakeDirRequest
	err := a.decoder.Unmarshal(r.Body, &msg)
	if err != nil {
		log.Error().Err(err).Msg("error unmarshaling sys request")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := a.registry.HandleMakeDir(&msg)
	if err != nil {
		log.Error().Err(err).Msg("error making directory")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	a.encoder.Marshal(rw, res)
}

func (a *Api) handleCreateUpdatePolicy(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyName, ok := vars["name"]
	if !ok {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	var msg api_pb.CreatePolicyRequest
	msg.Name = policyName
	err := a.decoder.Unmarshal(r.Body, &msg)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := a.registry.HandleCreatePolicy(&msg)
	if err != nil {
		log.Error().Err(err).Msg("error creating policy")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	a.encoder.Marshal(rw, res)
}

func (a *Api) handleGetPolicy(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyName, ok := vars["name"]
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if policyName == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := a.registry.HandleGetPolicy(&api_pb.GetPolicyRequest{Name: policyName})
	if err != nil {
		log.Error().Err(err).Msg("error getting policy")
		switch err {
		case errors.ErrEntryNotFound:
			rw.WriteHeader(http.StatusNotFound)
		default:
			rw.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if res == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	rw.WriteHeader(http.StatusOK)
	a.encoder.Marshal(rw, res)
}

func (a *Api) handleDeletePolicy(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	policyName, ok := vars["name"]
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	res, err := a.registry.HandleDeletePolicy(&api_pb.DeletePolicyRequest{Name: policyName})
	if err != nil {
		log.Error().Err(err).Msg("error deleting policy")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	a.encoder.Marshal(rw, res)
}

func (a *Api) handleListPolicies(rw http.ResponseWriter, r *http.Request) {
	res, err := a.registry.HandleListPolicies(&api_pb.ListPoliciesRequest{})
	if err != nil {
		log.Error().Err(err).Msg("error listing policies")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	a.encoder.Marshal(rw, res)
}

func (a *Api) handleCreateToken(rw http.ResponseWriter, r *http.Request) {
	var msg api_pb.CreateTokenRequest
	err := a.decoder.Unmarshal(r.Body, &msg)
	if err != nil {
		log.Error().Err(err).Msg("cant unmarshal")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := a.registry.HandleCreateToken(&msg)
	if err != nil {
		log.Error().Err(err).Msg("error creating token")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	a.encoder.Marshal(rw, res)
}

func (a *Api) handleGetToken(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accessor, ok := vars["accessor"]
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if accessor == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := a.registry.HandleGetToken(&api_pb.GetTokenRequest{Accessor: accessor})
	if err != nil {
		switch err {
		case errors.ErrEntryNotFound:
			rw.WriteHeader(http.StatusNotFound)
		default:
			log.Error().Err(err).Msg("error getting token")
			rw.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusOK)
	a.encoder.Marshal(rw, res)
}

func (a *Api) handleListTokens(rw http.ResponseWriter, r *http.Request) {
	res, err := a.registry.HandleListTokens(&api_pb.ListTokensRequest{})
	if err != nil {
		log.Error().Err(err).Msg("error listing tokens")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
	a.encoder.Marshal(rw, res)
}

func (a *Api) handleRevokeToken(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accessor, ok := vars["accessor"]
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	if accessor == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := a.registry.HandleRevokeToken(&api_pb.RevokeTokenRequest{Accessor: accessor})
	if err != nil {
		switch err {
		case errors.ErrEntryNotFound:
			rw.WriteHeader(http.StatusNotFound)
		default:
			log.Error().Err(err).Msg("error revoking token")
			rw.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	rw.WriteHeader(http.StatusOK)
	a.encoder.Marshal(rw, res)
}
