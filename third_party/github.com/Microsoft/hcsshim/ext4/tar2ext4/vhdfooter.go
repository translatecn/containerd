package tar2ext4

// Constants for the VHD footer
const (
	cookieMagic            = "conectix"
	featureMask            = 0x2
	fileFormatVersionMagic = 0x00010000
	fixedDataOffset        = -1
	creatorVersionMagic    = 0x000a0000
	diskTypeFixed          = 2
)
