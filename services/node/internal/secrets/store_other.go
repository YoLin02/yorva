//go:build !windows

package secrets

func newStore(string) (*Store, error) {
	return nil, ErrUnsupported
}

func (*Store) inspect(Reference) (bool, error) {
	return false, ErrUnsupported
}

func (*Store) get(Reference) ([]byte, error) {
	return nil, ErrUnsupported
}

func (*Store) put(Reference, []byte) error {
	return ErrUnsupported
}

func (*Store) delete(Reference) error {
	return ErrUnsupported
}
