package grokdisk

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"os"
)

const (
	// MBRPartitionTableOffset is the absolute location of the start of the MBR
	MBRPartitionTableOffset = 0x1BE
	// MBRPartitionTableSize is the length of the partition table in bytes
	MBRPartitionTableSize = 0x10
)

// AnalyzeImageFile enumerates partitions, determining information necessary
// to mount the image locally. This includes:
// * Type of partition table (GUID vs. MBR)
// * Sector size of imaged device
// * Byte offset of start of each partition
// The `mount` utility will then determine the filesystem of each
func AnalyzeImageFile(path string) (*ImageFileMetadata, error) {
	imageFile, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not open image file")
	}
	defer imageFile.Close()

	if _, err := imageFile.Seek(MBRPartitionTableOffset, os.SEEK_SET); err != nil {
		return nil, errors.Wrap(err, "could not seek partition table")
	}

	imageMetadata := &ImageFileMetadata{
		SectorSize: 512,
		Filepath:   path,
		Partitions: make([]*Partition, 0, 4),
	}

	// TODO: Expand for "extended partitions"
	// TODO: Account for GPT
	// Iterate over the four possible logical partitions in MBR
	for partitionIndex := 0; partitionIndex < 4; partitionIndex++ {
		metadata := &PartitionMetadata{}
		err = binary.Read(imageFile, binary.LittleEndian, metadata)
		if err != nil {
			return nil, errors.Wrap(err, "could not read partition entry")
		}
		partition := &Partition{
			PartitionMetadata: metadata,
			ImageFile:         imageMetadata,
		}
		imageMetadata.Partitions = append(imageMetadata.Partitions, partition)
	}

	return imageMetadata, nil
}

// ImageFileMetadata contains the list of 16-byte partition table entries
// and metadata associated with their use
type ImageFileMetadata struct {
	SectorSize uint16
	Filepath   string
	Partitions []*Partition
}

// Partition encapsulates the low level data from
// PartitionMetadata and provides additional computed data
type Partition struct {
	*PartitionMetadata
	// Pointer to image file in which this partition was found
	ImageFile *ImageFileMetadata
}

// PartitionMetadata represents the 16-byte partition table entry from
// the image MBR record
type PartitionMetadata struct {
	Status         byte
	StartHead      byte
	StartSector    byte
	StartCylinder  byte
	PartitionType  byte
	EndHead        byte
	EndSector      byte
	EndCylinder    byte
	FirstSectorLBA uint32
	SectorCount    uint32
}

// Start computes start byte (offset) of partition
func (p *Partition) Start() uint64 {
	return (uint64(p.FirstSectorLBA) * uint64(p.ImageFile.SectorSize))
}

// Size computes the length of the partition
func (p *Partition) Size() uint64 {
	return (uint64(p.SectorCount) * uint64(p.ImageFile.SectorSize))
}

func (p *Partition) String() string {
	return fmt.Sprintf(
		"status: %v type: %v, start: %v sectors (%v B), length: %v sectors (%v B)",
		p.Status, p.PartitionType,
		p.FirstSectorLBA, p.Start(),
		p.SectorCount, p.Size())
}
