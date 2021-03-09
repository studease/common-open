package avc

import (
	"fmt"

	"github.com/studease/common/av/utils"
	"github.com/studease/common/log"
)

// PPS (Picture Parameter Set)
type PPS struct {
	logger log.ILogger
	sps    *SPS

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

// Init this class.
func (me *PPS) Init(sps *SPS, logger log.ILogger) *PPS {
	me.sps = sps
	me.logger = logger
	return me
}

func (me *PPS) parse(data []byte) error {
	rbsp := ebsp2rbsp(data)
	gb := new(utils.Golomb).Init(rbsp)
	me.Data = rbsp

	me.ID = uint32(gb.ReadUE())
	if me.ID >= MAX_PPS_COUNT {
		return fmt.Errorf("PPS ID %d out of range", me.ID)
	}

	me.SpsID = uint32(gb.ReadUE())
	if me.SpsID >= MAX_SPS_COUNT {
		return fmt.Errorf("SPS ID %d out of range", me.SpsID)
	}

	if me.sps.BitDepthLuma > 14 {
		return fmt.Errorf("invalid BitDepthLuma %d", me.sps.BitDepthLuma)
	} else if me.sps.BitDepthLuma == 11 || me.sps.BitDepthLuma == 13 {
		return fmt.Errorf("unimplemented BitDepthLuma %d", me.sps.BitDepthLuma)
	}

	me.EntropyCodingModeFlag = uint8(gb.ReadBits(1))
	me.PicOrderPresentFlag = uint8(gb.ReadBits(1))
	me.NumSliceGroups = uint32(gb.ReadUE()) + 1
	if me.NumSliceGroups > 1 {
		me.SliceGroupMapType = uint32(gb.ReadUE())
		me.logger.Warnf("FMO not supported")

		switch me.SliceGroupMapType {
		case 0:
			/*
				for (i = 0; i <= num_slice_groups_minus1; i++)  |   |      |
				run_length[i]                                   |1  |ue(v) |
			*/

		case 2:
			/*
				for (i = 0; i < num_slice_groups_minus1; i++) { |   |      |
					top_left_mb[i]                              |1  |ue(v) |
					bottom_right_mb[i]                          |1  |ue(v) |
				}                                               |   |      |
			*/

		case 3:
			fallthrough
		case 4:
			fallthrough
		case 5:
			/*
				slice_group_change_direction_flag               |1  |u(1)  |
				slice_group_change_rate_minus1                  |1  |ue(v) |
			*/

		case 6:
			/*
				slice_group_id_cnt_minus1                       |1  |ue(v) |
				for (i = 0; i <= slice_group_id_cnt_minus1; i++)|   |      |
					slice_group_id[i]                           |1  |u(v)  |
			*/

		}
	}

	me.NumRefIdx[0] = uint32(gb.ReadUE()) + 1
	me.NumRefIdx[1] = uint32(gb.ReadUE()) + 1
	if me.NumRefIdx[0]-1 > 32-1 || me.NumRefIdx[1]-1 > 32-1 {
		return fmt.Errorf("reference overflow (pps)")
	}

	qpBdOffset := 6 * (me.sps.BitDepthLuma - 8)

	me.WeightedPredFlag = uint8(gb.ReadBits(1))
	me.WeightedBipredIdc = uint8(gb.ReadBits(2))
	me.PicInitQp = uint32(gb.ReadSE()) + 26 + qpBdOffset
	me.PicInitQs = uint32(gb.ReadSE()) + 26 + qpBdOffset
	me.ChromaQpIndexOffset[0] = int32(gb.ReadSE())
	me.DeblockingFilterControlPresentFlag = uint8(gb.ReadBits(1))
	me.ConstrainedIntraPredFlag = uint8(gb.ReadBits(1))
	me.RedundantPicCntPresentFlag = uint8(gb.ReadBits(1))

	me.Transform8x8ModeFlag = 0

	if gb.Left() > 0 && me.moreRBSPInPPS() {
		me.Transform8x8ModeFlag = uint8(gb.ReadBits(1))
		me.PicScalingMatrixPresentFlag = uint8(gb.ReadBits(1))
		if me.PicScalingMatrixPresentFlag != 0 {
			n := 2
			if me.sps.ChromaFormatIdc == 3 {
				n = 6
			}

			gb.ReadBits(6 + n*int(me.Transform8x8ModeFlag))
		}

		me.ChromaQpIndexOffset[1] = int32(gb.ReadSE()) // second_chroma_qp_index_offset
	} else {
		me.ChromaQpIndexOffset[1] = me.ChromaQpIndexOffset[0]
	}

	if me.ChromaQpIndexOffset[0] != me.ChromaQpIndexOffset[1] {
		me.ChromaQpDiff = 1
	}

	return nil
}

func (me *PPS) moreRBSPInPPS() bool {
	if (me.sps.ProfileIdc == 66 || me.sps.ProfileIdc == 77 || me.sps.ProfileIdc == 88) && (me.sps.ConstraintSetFlags&7) != 0 {
		me.logger.Warnf("current profile doesn't provide more RBSP data in PPS, skipping")
		return false
	}
	return true
}
