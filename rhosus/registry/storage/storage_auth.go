package storage

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
)

const (
	defaultRolesTableName  = "__roles"
	defaultTokensTableName = "__tokens"

	EntryTypeRole  EntryType = "role"
	EntryTypeToken EntryType = "token"
)

func (s *Storage) StoreRole(role *control_pb.Role) error {
	txn := s.db.Txn(true)

	err := txn.Insert(defaultRolesTableName, role)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	roleBytes, err := role.Marshal()
	if err != nil {
		return err
	}

	return s.backend.Put(EntryTypeRole, []*Entry{
		{
			Key:   role.ID,
			Value: roleBytes,
		},
	})
}

func (s *Storage) GetRole(name string) (*control_pb.Role, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultFilesTableName, "name", name)
	if err != nil {
		return nil, err
	}

	switch raw.(type) {
	case *control_pb.Role:
		return raw.(*control_pb.Role), nil
	default:
		return nil, nil
	}
}

func (s *Storage) GetRoleByID(roleID string) (*control_pb.Role, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultFilesTableName, "id", roleID)
	if err != nil {
		return nil, err
	}

	switch raw.(type) {
	case *control_pb.Role:
		return raw.(*control_pb.Role), nil
	default:
		return nil, nil
	}
}

func (s *Storage) DeleteRole(role *control_pb.Role) error {
	txn := s.db.Txn(true)

	err := txn.Delete(defaultRolesTableName, role)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	return s.backend.Delete(EntryTypeRole, []string{role.ID})
}

////////////////////////
// Tokens methods

func (s *Storage) StoreToken(token *control_pb.Token) error {
	txn := s.db.Txn(true)

	err := txn.Insert(defaultTokensTableName, token)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	tokenBytes, err := token.Marshal()
	if err != nil {
		return err
	}
	return s.backend.Put(EntryTypeToken, []*Entry{
		{
			Key:   token.Token,
			Value: tokenBytes,
		},
	})
}

func (s *Storage) GetToken(token string) (*control_pb.Token, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultTokensTableName, "id", token)
	if err != nil {
		return nil, err
	}

	switch raw.(type) {
	case *control_pb.Token:
		return raw.(*control_pb.Token), nil
	default:
		return nil, nil
	}
}

func (s *Storage) GetAllRoleTokens(roleID string) ([]*control_pb.Token, error) {
	txn := s.db.Txn(false)

	var tokens []*control_pb.Token
	res, err := txn.Get(defaultTokensTableName, "role_id", roleID)
	if err != nil {
		return nil, err
	}

	for obj := res.Next(); obj != nil; obj = res.Next() {
		tokens = append(tokens, obj.(*control_pb.Token))
	}

	return tokens, nil
}

func (s *Storage) GetAllTokens() ([]*control_pb.Token, error) {
	txn := s.db.Txn(false)

	var tokens []*control_pb.Token
	res, err := txn.Get(defaultTokensTableName, "id")
	if err != nil {
		return nil, err
	}

	for obj := res.Next(); obj != nil; obj = res.Next() {
		tokens = append(tokens, obj.(*control_pb.Token))
	}

	return tokens, nil
}

func (s *Storage) RevokeToken(token *control_pb.Token) error {
	txn := s.db.Txn(true)

	err := txn.Delete(defaultTokensTableName, token)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	return s.backend.Delete(EntryTypeToken, []string{token.Token})
}
