package qpstore

type Visibles struct{}

type StorageQP struct{}

func NewStore(v *Visibles) (Storage, error) {
	s := StorageQP{}

	return &s, nil
}

func (s *StorageQP) SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error) {
	// TODO implement
	return "", nil
}
