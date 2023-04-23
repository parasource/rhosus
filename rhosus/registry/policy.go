package registry

import control_pb "github.com/parasource/rhosus/rhosus/pb/control"

const (
	DenyCapabilityInt uint32 = 1 << iota
	CreateCapabilityInt
	ReadCapabilityInt
	UpdateCapabilityInt
	DeleteCapabilityInt
	ListCapabilityInt
	SudoCapabilityInt
)

var (
	cap2Int = map[control_pb.Policy_Capability]uint32{
		control_pb.Policy_Deny:   DenyCapabilityInt,
		control_pb.Policy_Create: CreateCapabilityInt,
		control_pb.Policy_Read:   ReadCapabilityInt,
		control_pb.Policy_Update: UpdateCapabilityInt,
		control_pb.Policy_Delete: DeleteCapabilityInt,
		control_pb.Policy_List:   ListCapabilityInt,
		control_pb.Policy_Sudo:   SudoCapabilityInt,
	}
)
