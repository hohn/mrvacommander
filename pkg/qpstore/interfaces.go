package qpstore

type Storage interface {
	SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error)
}
