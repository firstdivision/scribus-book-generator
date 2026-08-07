package images

type Metadata struct {
	Path     string
	Caption  string
	Width    int
	Height   int
	Priority int
}

func Analyze(path string) (Metadata, error) {
	return Metadata{Path: path}, nil
}
