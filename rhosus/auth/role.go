package auth

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/registry/storage"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RoleStore struct {
	storage *storage.Storage
}

func NewRoleManager(s *storage.Storage) (*RoleStore, error) {
	return &RoleStore{
		storage: s,
	}, nil
}

func (m *RoleStore) CreateRole(name string, password string, perms []string) (*control_pb.Role, error) {
	roleID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	passw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	role := &control_pb.Role{
		ID:          roleID.String(),
		Name:        name,
		Permissions: perms,
		Password:    string(passw),
	}
	err = m.storage.StoreRole(role)
	if err != nil {
		return nil, err
	}

	return role, nil
}

func (m *RoleStore) GetRole(name string) (*control_pb.Role, error) {
	return m.storage.GetRole(name)
}

func (m *RoleStore) GetRoleById(roleID string) (*control_pb.Role, error) {
	return m.storage.GetRoleByID(roleID)
}

func (m *RoleStore) UpdateRole() {

}

func (m *RoleStore) DeleteRole() {

}
