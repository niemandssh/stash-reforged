package models

import "context"

// PerformerProfileImageGetter provides methods to get performer profile images by ID.
type PerformerProfileImageGetter interface {
	Find(ctx context.Context, id int) (*PerformerProfileImage, error)
}

// PerformerProfileImageFinder provides methods to find performer profile images.
type PerformerProfileImageFinder interface {
	PerformerProfileImageGetter
	FindByPerformerID(ctx context.Context, performerID int) ([]*PerformerProfileImage, error)
}

// PerformerProfileImageCounter provides methods to count performer profile images.
type PerformerProfileImageCounter interface {
	Count(ctx context.Context) (int, error)
}

// PerformerProfileImageCreator provides methods to create performer profile images.
type PerformerProfileImageCreator interface {
	Create(ctx context.Context, newImage *CreatePerformerProfileImageInput) (*PerformerProfileImage, error)
}

// PerformerProfileImageUpdater provides methods to update performer profile images.
type PerformerProfileImageUpdater interface {
	UpdatePartial(ctx context.Context, id int, updatedImage PerformerProfileImagePartial) (*PerformerProfileImage, error)
	UpdateImage(ctx context.Context, id int, image []byte) error
}

// PerformerProfileImageDestroyer provides methods to destroy performer profile images.
type PerformerProfileImageDestroyer interface {
	Destroy(ctx context.Context, id int) error
}

// PerformerProfileImageReader provides all methods to read performer profile images.
type PerformerProfileImageReader interface {
	PerformerProfileImageFinder
	PerformerProfileImageCounter

	GetImage(ctx context.Context, id int) ([]byte, error)
	HasImage(ctx context.Context, id int) (bool, error)
}

// PerformerProfileImageWriter provides all methods to modify performer profile images.
type PerformerProfileImageWriter interface {
	PerformerProfileImageCreator
	PerformerProfileImageUpdater
	PerformerProfileImageDestroyer
}

// PerformerProfileImageReaderWriter provides all performer profile image methods.
type PerformerProfileImageReaderWriter interface {
	PerformerProfileImageReader
	PerformerProfileImageWriter
}
