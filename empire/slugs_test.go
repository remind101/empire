package empire

type mockExtractor struct {
	ExtractFunc func(Image) (CommandMap, error)
}

func (e *mockExtractor) Extract(image Image) (CommandMap, error) {
	if e.ExtractFunc != nil {
		return e.ExtractFunc(image)
	}

	return CommandMap{}, nil
}

type mockslugsService struct {
	slugsService // Just to satisfy the interface.

	SlugsCreateByImageFunc func(Image) (*Slug, error)
}

func (s *mockslugsService) SlugsCreateByImage(image Image) (*Slug, error) {
	if s.SlugsCreateByImageFunc != nil {
		return s.SlugsCreateByImageFunc(image)
	}

	return nil, nil
}
