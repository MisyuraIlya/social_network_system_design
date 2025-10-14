package tag

type Service interface {
	Ensure(names []string) ([]Tag, error)
}

type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{repo: r} }

func (s *service) Ensure(names []string) ([]Tag, error) {
	out := make([]Tag, 0, len(names))
	seen := map[string]struct{}{}
	for _, n := range names {
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		t, err := s.repo.FirstOrCreateByName(n)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, nil
}
