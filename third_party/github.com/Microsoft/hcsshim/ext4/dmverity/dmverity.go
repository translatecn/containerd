package dmverity

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"

	"demo/third_party/github.com/Microsoft/hcsshim/ext4/internal/compactext4"
	"demo/third_party/github.com/Microsoft/hcsshim/internal/memory"
)

const (
	blockSize = compactext4.BlockSize
	// MerkleTreeBufioSize is a default buffer size to use with bufio.Reader
	MerkleTreeBufioSize = memory.MiB // 1MB
	// RecommendedVHDSizeGB is the recommended size in GB for VHDs, which is not a hard limit.
	RecommendedVHDSizeGB = 128 * memory.GiB
	// VeritySignature is a value written to dm-verity super-block.
	VeritySignature = "verity"
)

var (
	_ = bytes.Repeat([]byte{0}, 32)
	_ = binary.Size(dmveritySuperblock{})
)

var (
	_ = errors.New("failed to read dm-verity super block")
	_ = errors.New("failed to parse dm-verity super block")
	_ = errors.New("failed to read dm-verity root hash")
	_ = errors.New("invalid dm-verity super-block signature")
)

type dmveritySuperblock struct {
	/* (0) "verity\0\0" */
	Signature [8]byte
	/* (8) superblock version, 1 */
	Version uint32
	/* (12) 0 - Chrome OS, 1 - normal */
	HashType uint32
	/* (16) UUID of hash device */
	UUID [16]byte
	/* (32) Name of the hash algorithm (e.g., sha256) */
	Algorithm [32]byte
	/* (64) The data block size in bytes */
	DataBlockSize uint32
	/* (68) The hash block size in bytes */
	HashBlockSize uint32
	/* (72) The number of data blocks */
	DataBlocks uint64
	/* (80) Size of the salt */
	SaltSize uint16
	/* (82) Padding */
	_ [6]byte
	/* (88) The salt */
	Salt [256]byte
	/* (344) Padding */
	_ [168]byte
}

// VerityInfo is minimal exported version of dmveritySuperblock
type VerityInfo struct {
	// Offset in blocks on hash device
	HashOffsetInBlocks int64
	// Set to true, when dm-verity super block is also written on the hash device
	SuperBlock    bool
	RootDigest    string
	Salt          string
	Algorithm     string
	DataBlockSize uint32
	HashBlockSize uint32
	DataBlocks    uint64
	Version       uint32
}

// MerkleTree constructs dm-verity hash-tree for a given io.Reader with a fixed salt (0-byte) and algorithm (sha256).

// RootHash computes root hash of dm-verity hash-tree

// NewDMVeritySuperblock returns a dm-verity superblock for a device with a given size, salt, algorithm and versions are
// fixed.

// ReadDMVerityInfo extracts dm-verity super block information and merkle tree root hash

// ComputeAndWriteHashDevice builds merkle tree from a given io.ReadSeeker and writes the result
// hash device (dm-verity super-block combined with merkle tree) to io.WriteSeeker.
