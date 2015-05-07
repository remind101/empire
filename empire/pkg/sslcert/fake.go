package sslcert

type FakeManager struct{}

func NewFakeManager() *FakeManager {
	return &FakeManager{}
}

func (m *FakeManager) Add(name string, crt string, key string) (string, error) {
	return "fake", nil
}

func (m *FakeManager) Remove(id string) error {
	return nil
}

func (m *FakeManager) MetaData(id string) (map[string]string, error) {
	return map[string]string{}, nil
}
