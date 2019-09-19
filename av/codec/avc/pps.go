package avc

import (
	"github.com/studease/common/av/utils"
)

// PPS (Picture Parameter Set)
type PPS struct {
	Golomb                             utils.Golomb
	ID                                 uint32 // pic_parameter_set_id
	SpsID                              uint32 // seq_parameter_set_id
	EntropyCodingModeFlag              uint8  // 1 bits
	PicOrderPresentFlag                uint8  // 1 bits, bottom_field_pic_order_in_frame_present_flag
	NumSliceGroups                     uint32 // num_slice_groups_minus1 + 1
	SliceGroupMapType                  uint32
	NumRefIdx                          [2]uint32 // num_ref_idx_l0/1_default_active_minus1 + 1
	WeightedPredFlag                   uint8     // 1 bits
	WeightedBipredIdc                  uint8     // 1 bits
	PicInitQp                          uint32    // pic_init_qp_minus26 + 26
	PicInitQs                          uint32    // pic_init_qs_minus26 + 26
	ChromaQpIndexOffset                [2]int32
	DeblockingFilterControlPresentFlag uint8 // 1 bits
	ConstrainedIntraPredFlag           uint8 // 1 bits
	RedundantPicCntPresentFlag         uint8 // 1 bits
	Transform8x8ModeFlag               uint8 // 1 bits
	PicScalingMatrixPresentFlag        uint8 // 1 bits
	ChromaQpDiff                       int32
	Data                               []byte
}
