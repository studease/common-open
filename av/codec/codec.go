package codec

import (
	"github.com/studease/common/av"
	"github.com/studease/common/log"
	"github.com/studease/common/utils"
)

/*
 * Chromaticity coordinates of the source primaries.
 */ //
const (
	COL_PRI_RESERVED0    uint8 = 0
	COL_PRI_BT709        uint8 = 1 // also ITU-R BT1361 / IEC 61966-2-4 / SMPTE RP177 Annex B
	COL_PRI_UNSPECIFIED  uint8 = 2
	COL_PRI_RESERVED     uint8 = 3
	COL_PRI_BT470M       uint8 = 4  // also FCC Title 47 Code of Federal Regulations 73.682 (a)(20)
	COL_PRI_BT470BG      uint8 = 5  // also ITU-R BT601-6 625 / ITU-R BT1358 625 / ITU-R BT1700 625 PAL & SECAM
	COL_PRI_SMPTE170M    uint8 = 6  // also ITU-R BT601-6 525 / ITU-R BT1358 525 / ITU-R BT1700 NTSC
	COL_PRI_SMPTE240M    uint8 = 7  // functionally identical to above
	COL_PRI_FILM         uint8 = 8  // colour filters using Illuminant C
	COL_PRI_BT2020       uint8 = 9  // ITU-R BT2020
	COL_PRI_SMPTEST428_1 uint8 = 10 // SMPTE ST 428-1 (CIE 1931 XYZ)
	COL_PRI_NB           uint8 = 11 // Not part of ABI
)

/*
 * Color Transfer Characteristic.
 */
const (
	COL_TRC_RESERVED0    uint8 = 0
	COL_TRC_BT709        uint8 = 1 // also ITU-R BT1361
	COL_TRC_UNSPECIFIED  uint8 = 2
	COL_TRC_RESERVED     uint8 = 3
	COL_TRC_GAMMA22      uint8 = 4 // also ITU-R BT470M / ITU-R BT1700 625 PAL & SECAM
	COL_TRC_GAMMA28      uint8 = 5 // also ITU-R BT470BG
	COL_TRC_SMPTE170M    uint8 = 6 // also ITU-R BT601-6 525 or 625 / ITU-R BT1358 525 or 625 / ITU-R BT1700 NTSC
	COL_TRC_SMPTE240M    uint8 = 7
	COL_TRC_LINEAR       uint8 = 8  // "Linear transfer characteristics"
	COL_TRC_LOG          uint8 = 9  // "Logarithmic transfer characteristic (100:1 range)"
	COL_TRC_LOG_SQRT     uint8 = 10 // "Logarithmic transfer characteristic (100 * Sqrt(10) : 1 range)"
	COL_TRC_IEC61966_2_4 uint8 = 11 // IEC 61966-2-4
	COL_TRC_BT1361_ECG   uint8 = 12 // ITU-R BT1361 Extended Colour Gamut
	COL_TRC_IEC61966_2_1 uint8 = 13 // IEC 61966-2-1 (sRGB or sYCC)
	COL_TRC_BT2020_10    uint8 = 14 // ITU-R BT2020 for 10-bit system
	COL_TRC_BT2020_12    uint8 = 15 // ITU-R BT2020 for 12-bit system
	COL_TRC_SMPTEST2084  uint8 = 16 // SMPTE ST 2084 for 10-, 12-, 14- and 16-bit systems
	COL_TRC_SMPTEST428_1 uint8 = 17 // SMPTE ST 428-1
	COL_TRC_ARIB_STD_B67 uint8 = 18 // ARIB STD-B67, known as "Hybrid log-gamma"
	COL_TRC_NB           uint8 = 19 // Not part of ABI
)

/*
 * YUV colorspace type.
 */
const (
	COL_SPC_RGB         uint8 = 0 // order of coefficients is actually GBR, also IEC 61966-2-1 (sRGB)
	COL_SPC_BT709       uint8 = 1 // also ITU-R BT1361 / IEC 61966-2-4 xvYCC709 / SMPTE RP177 Annex B
	COL_SPC_UNSPECIFIED uint8 = 2
	COL_SPC_RESERVED    uint8 = 3
	COL_SPC_FCC         uint8 = 4 // FCC Title 47 Code of Federal Regulations 73.682 (a)(20)
	COL_SPC_BT470BG     uint8 = 5 // also ITU-R BT601-6 625 / ITU-R BT1358 625 / ITU-R BT1700 625 PAL & SECAM / IEC 61966-2-4 xvYCC601
	COL_SPC_SMPTE170M   uint8 = 6 // also ITU-R BT601-6 525 / ITU-R BT1358 525 / ITU-R BT1700 NTSC
	COL_SPC_SMPTE240M   uint8 = 7 // functionally identical to above
	COL_SPC_YCOCG       uint8 = 8 // Used by Dirac / VC-2 and H.264 FRext, see ITU-T SG16
	COL_SPC_YCGCO       uint8 = 8
	COL_SPC_BT2020_NCL  uint8 = 9  // ITU-R BT2020 non-constant luminance system
	COL_SPC_BT2020_CL   uint8 = 10 // ITU-R BT2020 constant luminance system
	COL_SPC_NB          uint8 = 11 // Not part of ABI
)

/*
 * MPEG vs JPEG YUV range.
 */
const (
	COL_RANGE_UNSPECIFIED uint8 = 0
	COL_RANGE_MPEG        uint8 = 1 // the normal 219*2^(n-8) "MPEG" YUV ranges
	COL_RANGE_JPEG        uint8 = 2 // the normal     2^n-1   "JPEG" YUV ranges
	COL_RANGE_NB          uint8 = 2 // Not part of ABI
)

/*
 * Location of chroma samples.
 *
 * Illustration showing the location of the first (top left) chroma sample of the
 * image, the left shows only luma, the right
 * shows the location of the chroma sample, the 2 could be imagined to overlay
 * each other but are drawn separately due to limitations of ASCII
 *
 *                 1st 2nd       1st 2nd horizontal luma sample positions
 *                  v   v         v   v
 *                 ______        ______
 * 1st luma line > |X   X ...    |3 4 X ...     X are luma samples,
 *                 |             |1 2           1-6 are possible chroma positions
 * 2nd luma line > |X   X ...    |5 6 X ...     0 is undefined/unknown position
 */
const (
	CHROMA_LOC_UNSPECIFIED uint8 = 0
	CHROMA_LOC_LEFT        uint8 = 1 // MPEG-2/4 4:2:0, H.264 default for 4:2:0
	CHROMA_LOC_CENTER      uint8 = 2 // MPEG-1 4:2:0, JPEG 4:2:0, H.263 4:2:0
	CHROMA_LOC_TOPLEFT     uint8 = 3 // ITU-R 601, SMPTE 274M 296M S314M(DV 4:1:1), mpeg2 4:2:2
	CHROMA_LOC_TOP         uint8 = 4
	CHROMA_LOC_BOTTOMLEFT  uint8 = 5
	CHROMA_LOC_BOTTOM      uint8 = 6
	CHROMA_LOC_NB          uint8 = 7 // Not part of ABI
)

var (
	r = utils.NewRegister()
)

// Register an IMediaStreamTrackSource with the given codec.
func Register(codec string, source interface{}) {
	r.Add(codec, source)
}

// New creates a registered IMediaStreamTrackSource by the codec.
func New(codec string, info *av.Information, factory log.ILoggerFactory) av.IMediaStreamTrackSource {
	if source := r.New(codec); source != nil {
		return source.(av.IMediaStreamTrackSource).Init(info, factory.NewLogger(codec))
	}
	return nil
}
