package storage

import (
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
)

const (
	defaultPoliciesTableName           = "__policies"
	EntryTypePolicy          EntryType = "policy"
)

func (s *Storage) StorePolicy(p *control_pb.Policy) error {
	err := s.storePolicyInMemory(p)
	if err != nil {
		return fmt.Errorf("error storing policy in memory: %w", err)
	}

	policyBytes, err := p.Marshal()
	if err != nil {
		return fmt.Errorf("error marshalling policy: %w", err)
	}

	return s.backend.Put(EntryTypePolicy, []*Entry{
		{
			Key:   p.Name,
			Value: policyBytes,
		},
	})
}

func (s *Storage) storePolicyInMemory(p *control_pb.Policy) error {
	txn := s.db.Txn(true)

	err := txn.Insert(defaultPoliciesTableName, p)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	return nil
}

func (s *Storage) GetPolicy(name string) (*control_pb.Policy, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultPoliciesTableName, "name", name)
	if err != nil {
		return nil, err
	}

	switch raw.(type) {
	case *control_pb.Policy:
		return raw.(*control_pb.Policy), nil
	default:
		return nil, nil
	}
}

func (s *Storage) ListPolicies() ([]*control_pb.Policy, error) {
	txn := s.db.Txn(false)

	res, err := txn.Get(defaultPoliciesTableName, "name")
	if err != nil {
		return nil, err
	}

	var policies []*control_pb.Policy
	for obj := res.Next(); obj != nil; obj = res.Next() {
		policies = append(policies, obj.(*control_pb.Policy))
	}

	return policies, nil
}

func (s *Storage) DeletePolicy(p *control_pb.Policy) error {
	txn := s.db.Txn(true)

	err := txn.Delete(defaultPoliciesTableName, p)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	return s.backend.Delete(EntryTypePolicy, []string{p.Name})
}
